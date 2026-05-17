package api

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// contextKey is unexported so only this package can read/write request-scoped
// values, preventing collisions with other packages' context keys.
type contextKey string

const userContextKey contextKey = "api.user"

// middleware is the standard wrapper signature used across the API.
type middleware func(http.Handler) http.Handler

// chain composes middleware around h. The first entry is the OUTERMOST wrapper,
// so it runs first on the way in and last on the way out.
func (s *Server) chain(h http.Handler, mw ...middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// recoverPanic converts a panic in any downstream handler into a 500 response
// instead of crashing the process — the same safety the bot loop already has.
func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("api: handler panic recovered",
					slog.Any("panic", rec),
					slog.String("path", r.URL.Path),
					slog.String("stack", string(debug.Stack())),
				)
				WriteError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// securityHeaders sets baseline hardening headers on every response — API,
// SPA and health alike. It is the outermost wrapper around the whole mux.
//
//   - X-Content-Type-Options: nosniff — no MIME sniffing.
//   - Referrer-Policy — never leak full URLs cross-origin.
//   - Strict-Transport-Security — pin HTTPS (Railway terminates TLS at edge).
//   - Content-Security-Policy: frame-ancestors — the Mini App must stay
//     embeddable by the Telegram clients but by NO other site, which blocks
//     clickjacking. X-Frame-Options cannot express an allow-list, so a CSP
//     directive is used instead of (not alongside) it.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy",
			"frame-ancestors 'self' https://web.telegram.org https://*.telegram.org")
		next.ServeHTTP(w, r)
	})
}

// timeout bounds every downstream handler with a deadline so a slow query or
// a large export can never hang a request indefinitely. The deadline rides on
// the request context, so all pgx queries inherit it automatically.
func (s *Server) timeout(d time.Duration) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requireTelegramAuth verifies the Telegram Mini App initData on the request,
// resolves (or registers) the corresponding user, and stores it in the request
// context. Any failure short-circuits with a generic 401.
func (s *Server) requireTelegramAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := extractInitData(r)
		if !ok {
			WriteError(w, http.StatusUnauthorized, "требуется авторизация Telegram")
			return
		}
		data, err := VerifyInitData(raw, s.botToken, initDataMaxAge)
		if err != nil {
			s.log.Warn("api: initData verification failed",
				slog.String("reason", err.Error()),
				slog.String("fields", initDataKeys(raw)),
				slog.Int("raw_len", len(raw)),
				slog.String("path", r.URL.Path))
			WriteError(w, http.StatusUnauthorized, "недействительные данные авторизации")
			return
		}
		user, err := s.users.EnsureUser(r.Context(), data.User.ID, data.User.Username, data.User.FirstName)
		if err != nil {
			s.log.Error("api: ensure user failed", slog.String("error", err.Error()))
			WriteError(w, http.StatusInternalServerError, "не удалось определить пользователя")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAdmin gates a route to admin users. It must run AFTER
// requireTelegramAuth, which places the resolved user in the context.
// Role is checked against models.RoleAdmin — never against a hardcoded ID.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := userFromContext(r.Context())
		if user == nil || user.Role != models.RoleAdmin {
			WriteError(w, http.StatusForbidden, "доступ только для администратора")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// userFromContext returns the authenticated user, or nil if the request did
// not pass through requireTelegramAuth.
func userFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(userContextKey).(*models.User)
	return u
}

// extractInitData pulls the raw initData from an "Authorization: tma <data>"
// header — the convention used by the Telegram Mini Apps ecosystem.
func extractInitData(r *http.Request) (string, bool) {
	const prefix = "tma "
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, prefix) {
		return "", false
	}
	if v := strings.TrimSpace(auth[len(prefix):]); v != "" {
		return v, true
	}
	return "", false
}
