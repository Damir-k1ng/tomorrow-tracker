package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
)

// Per-group rate limits (requests per minute, per client IP). Auth and admin
// surfaces are the most sensitive, so they are the most restricted.
const (
	authRatePerMin  = 15
	userRatePerMin  = 60
	adminRatePerMin = 30
)

// correctionRatePerMin is a stricter, per-admin cap applied ONLY to the
// session-correction endpoint (PATCH /admin/sessions/{id}). Corrections mutate
// streak state and append immutable audit rows, so they are deliberately held
// well below the general admin read budget.
const correctionRatePerMin = 10

// adminRequestTimeout bounds every admin request (stats, listings, exports,
// corrections) so a slow query can never hang the handler.
const adminRequestTimeout = 30 * time.Second

// userRequestTimeout bounds every user-facing request. User queries are small
// and read-only, so the budget is tighter than the admin one — a slow query
// fails fast instead of holding the Telegram WebView waiting.
const userRequestTimeout = 15 * time.Second

// Server is the HTTP layer: a Railway health endpoint plus the /api/v1 surface
// for the future Telegram Mini App and Admin Panel. It runs alongside — and
// independently of — the Telegram polling bot.
type Server struct {
	srv         *http.Server
	users       *services.UserService
	userAPI     *services.UserAPIService
	admin       *services.AdminService
	log         *slog.Logger
	botToken    string
	corsOrigins []string

	authLimiter       *rateLimiter
	userLimiter       *rateLimiter
	adminLimiter      *rateLimiter
	correctionLimiter *rateLimiter
	stopJanitor       chan struct{}

	// spa serves the embedded Telegram Mini App SPA. It is the catch-all "/"
	// handler; the /api/v1 routes sit on more specific patterns and always win.
	spa http.Handler
}

// New builds the API server. botToken is needed to verify Telegram initData;
// corsOrigins is a comma-separated allow-list ("*" permits any origin). spa is
// the embedded Mini App handler — when nil (e.g. in tests) a small stub stands
// in so routing still works.
func New(port, botToken, corsOrigins string, users *services.UserService, userAPI *services.UserAPIService, admin *services.AdminService, spa http.Handler, log *slog.Logger) *Server {
	if spa == nil {
		spa = http.HandlerFunc(handleSPAUnavailable)
	}
	s := &Server{
		users:             users,
		userAPI:           userAPI,
		admin:             admin,
		log:               log,
		botToken:          botToken,
		corsOrigins:       parseOrigins(corsOrigins),
		authLimiter:       newRateLimiter(authRatePerMin),
		userLimiter:       newRateLimiter(userRatePerMin),
		adminLimiter:      newRateLimiter(adminRatePerMin),
		correctionLimiter: newRateLimiter(correctionRatePerMin),
		stopJanitor:       make(chan struct{}),
		spa:               spa,
	}
	s.srv = &http.Server{
		Addr:              ":" + port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// routes wires the URL space. The /api/v1 groups each get their own middleware
// stack; recoverPanic and cors are outermost so panics are always caught and
// CORS preflight is answered before auth / rate-limiting runs.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Railway health checks.
	mux.HandleFunc("/health", handleHealth)

	// Embedded Mini App SPA: the catch-all "/" mount. Client-side routes and
	// deep links fall back to index.html inside the spa handler.
	mux.Handle("/", s.spa)

	// Any unmatched /api/ path returns a structured JSON 404 — the SPA fallback
	// must NEVER serve HTML for an API route.
	mux.HandleFunc("/api/", handleAPINotFound)

	// Mini App login — verifies initData itself, so no auth middleware here.
	mux.Handle("/api/v1/auth/verify", s.chain(
		http.HandlerFunc(s.handleAuthVerify),
		s.recoverPanic, s.cors, s.rateLimit(s.authLimiter),
	))

	// User group: any authenticated Telegram user. Every request is bounded by
	// a timeout so a slow query cannot hang the Telegram WebView.
	mux.Handle("/api/v1/user/", s.chain(
		s.userRoutes(),
		s.recoverPanic, s.cors, s.timeout(userRequestTimeout),
		s.rateLimit(s.userLimiter), s.requireTelegramAuth,
	))

	// Admin group: authenticated AND role == admin. Every request is also
	// bounded by a timeout so slow queries / exports cannot hang.
	mux.Handle("/api/v1/admin/", s.chain(
		s.adminRoutes(),
		s.recoverPanic, s.cors, s.timeout(adminRequestTimeout),
		s.rateLimit(s.adminLimiter), s.requireTelegramAuth, s.requireAdmin,
	))

	return mux
}

// userRoutes is the sub-router for the user-facing API. It is mounted behind
// the user middleware chain, so every handler here is guaranteed an
// authenticated Telegram caller in the request context.
func (s *Server) userRoutes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/user/me", s.handleUserMe)
	m.HandleFunc("GET /api/v1/user/sessions", s.handleUserSessions)
	m.HandleFunc("GET /api/v1/user/leaderboard", s.handleUserLeaderboard)
	// Structured 404 for any other /api/v1/user/* path.
	m.HandleFunc("/api/v1/user/", handleNotFound)
	return m
}

// adminRoutes is the sub-router for the Admin Panel API. It is mounted behind
// the admin middleware chain, so every handler here is guaranteed an
// authenticated admin caller.
func (s *Server) adminRoutes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/admin/stats", s.handleAdminStats)
	m.HandleFunc("GET /api/v1/admin/users", s.handleAdminListUsers)
	m.HandleFunc("GET /api/v1/admin/users/{id}", s.handleAdminUserDetails)
	// The correction endpoint carries an extra, stricter per-admin rate limit
	// on top of the shared admin limiter — it mutates streak state and writes
	// immutable audit rows. The limiter runs inside the admin chain, so the
	// authenticated admin is already in the request context.
	m.Handle("PATCH /api/v1/admin/sessions/{id}", s.chain(
		http.HandlerFunc(s.handleAdminPatchSession),
		s.rateLimitByAdmin(s.correctionLimiter),
	))
	m.HandleFunc("GET /api/v1/admin/export/users", s.handleAdminExportUsers)
	m.HandleFunc("GET /api/v1/admin/export/sessions", s.handleAdminExportSessions)
	m.HandleFunc("GET /api/v1/admin/audit-logs", s.handleAdminListAuditLogs)
	// Structured 404 for any other /api/v1/admin/* path.
	m.HandleFunc("/api/v1/admin/", handleNotFound)
	return m
}

// Start launches the HTTP server and the rate-limiter janitor in background
// goroutines. A bind failure is logged but not fatal — the bot still runs.
func (s *Server) Start() {
	go func() {
		s.log.Info("api server listening", slog.String("addr", s.srv.Addr))
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("api server error", slog.String("error", err.Error()))
		}
	}()
	go s.runJanitor()
}

// runJanitor periodically evicts idle rate-limiter buckets so their maps
// cannot grow without bound.
func (s *Server) runJanitor() {
	const (
		every   = time.Minute
		maxIdle = 10 * time.Minute
	)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-s.stopJanitor:
			return
		case <-t.C:
			for _, rl := range []*rateLimiter{s.authLimiter, s.userLimiter, s.adminLimiter, s.correctionLimiter} {
				rl.sweep(maxIdle)
			}
		}
	}
}

// Shutdown gracefully stops the server and the janitor, bounded by a timeout.
func (s *Server) Shutdown() {
	close(s.stopJanitor)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		s.log.Error("api server shutdown error", slog.String("error", err.Error()))
	}
}
