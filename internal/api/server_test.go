package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
	"github.com/damirkabdulla/tomorrow-tracker/pkg/logger"
)

// apiFakeRepo is an in-memory repositories.UserRepository for HTTP-level tests
// that exercise routing/middleware without a database.
type apiFakeRepo struct {
	byTG   map[int64]*models.User
	nextID int64
}

func newAPIFakeRepo() *apiFakeRepo { return &apiFakeRepo{byTG: map[int64]*models.User{}} }

func (r *apiFakeRepo) Upsert(_ context.Context, tg int64, username, firstName string) (*models.User, error) {
	u, ok := r.byTG[tg]
	if !ok {
		r.nextID++
		u = &models.User{ID: r.nextID, TelegramID: tg, Role: models.RoleUser}
		r.byTG[tg] = u
	}
	u.Username, u.FirstName = username, firstName
	cp := *u
	return &cp, nil
}

func (r *apiFakeRepo) GetByTelegramID(_ context.Context, tg int64) (*models.User, error) {
	if u, ok := r.byTG[tg]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, repositories.ErrNotFound
}

func (r *apiFakeRepo) GetByID(_ context.Context, id int64) (*models.User, error) {
	for _, u := range r.byTG {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (r *apiFakeRepo) UpdateStreak(context.Context, int64, int, int, time.Time) error { return nil }

func (r *apiFakeRepo) UpdateRole(_ context.Context, id int64, role string) error {
	for _, u := range r.byTG {
		if u.ID == id {
			u.Role = role
			return nil
		}
	}
	return repositories.ErrNotFound
}

// newTestServer builds an API server backed by the in-memory repo. adminID is
// the Telegram ID auto-promoted to admin. The AdminService is constructed with
// nil repos — these middleware/routing tests never reach admin business
// handlers, so it is never invoked. The UserAPIService is backed by an empty
// in-memory session repo so the user routes are wired but return empty data.
func newTestServer(adminID int64) http.Handler {
	repo := newAPIFakeRepo()
	users := services.NewUserService(repo, adminID)
	admin := services.NewAdminService(nil, nil, nil)
	userAPI := newTestUserAPI(newFakeSessionRepo(), repo)
	s := New("0", testBotToken, "*", users, userAPI, admin, nil, "", nil, logger.New("error"))
	return s.routes()
}

func authHeader(initData string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "tma "+initData)
	return h
}

func TestServer_HealthEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(1).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("health: got %d %q", rec.Code, rec.Body.String())
	}
}

func TestServer_CORSPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/anything", nil)
	req.Header.Set("Origin", "https://example.org")
	newTestServer(1).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status: got %d want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS allow-origin header")
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("missing CORS allow-methods header")
	}
}

func TestServer_UserRoute_RequiresAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(1).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/user/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated user route: got %d want 401", rec.Code)
	}
	assertStructuredError(t, rec)
}

func TestServer_AdminRoute_RejectsNonAdmin(t *testing.T) {
	const adminID, regularID = int64(100), int64(200)
	srv := newTestServer(adminID)

	// A valid, non-admin Telegram user must get 403 from the admin group.
	initData := buildInitData(testBotToken, TelegramUser{ID: regularID, FirstName: "Reg"}, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/x", nil)
	req.Header = authHeader(initData)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin on admin route: got %d want 403", rec.Code)
	}
	assertStructuredError(t, rec)
}

func TestServer_AdminRoute_AllowsAdmin(t *testing.T) {
	const adminID = int64(165146312)
	srv := newTestServer(adminID)

	// The configured admin passes auth + RequireAdmin; with no business
	// endpoint yet they reach the structured 404 — proving the full
	// middleware chain executed.
	initData := buildInitData(testBotToken, TelegramUser{ID: adminID, FirstName: "Damir"}, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/x", nil)
	req.Header = authHeader(initData)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("admin on admin route: got %d want 404 (passed middleware)", rec.Code)
	}
	assertStructuredError(t, rec)
}

func TestServer_AuthVerify_ReturnsUserWithRole(t *testing.T) {
	const adminID = int64(165146312)
	srv := newTestServer(adminID)

	initData := buildInitData(testBotToken, TelegramUser{ID: adminID, FirstName: "Damir", Username: "king"}, time.Now())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify", nil)
	req.Header = authHeader(initData)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth verify: got %d want 200", rec.Code)
	}

	var env Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if !env.Success {
		t.Fatalf("expected success envelope, got %+v", env)
	}
	data, _ := env.Data.(map[string]any)
	if data["role"] != models.RoleAdmin {
		t.Fatalf("expected admin role in response, got %v", data["role"])
	}
}

func TestServer_AuthVerify_RejectsInvalidInitData(t *testing.T) {
	srv := newTestServer(1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify", nil)
	req.Header = authHeader("auth_date=1&hash=bogus&user=%7B%22id%22%3A1%7D")

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid initData: got %d want 401", rec.Code)
	}
}

func TestServer_APIUnknownPath_StructuredJSON404(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(1).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown api path: got %d want 404", rec.Code)
	}
	// An unknown /api/ path must stay JSON — never fall through to the SPA.
	assertStructuredError(t, rec)
}

func TestServer_SPACatchAll(t *testing.T) {
	// No SPA is wired in tests, so New() substitutes the SPA stub (503). Any
	// non-API path reaching that stub proves the "/" catch-all routes client
	// routes to the SPA handler rather than to an API 404.
	for _, path := range []string{"/", "/leaderboard", "/admin/users"} {
		rec := httptest.NewRecorder()
		newTestServer(1).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: got %d, want 503 (SPA stub reached)", path, rec.Code)
		}
	}
}

func assertStructuredError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var env Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("response is not a JSON envelope: %v (body=%q)", err, rec.Body.String())
	}
	if env.Success || env.Error == "" {
		t.Fatalf("expected error envelope, got %+v", env)
	}
}
