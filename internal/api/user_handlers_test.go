package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
	"github.com/damirkabdulla/tomorrow-tracker/internal/services"
	"github.com/damirkabdulla/tomorrow-tracker/pkg/logger"
)

// testUserID is the in-memory user id assigned by apiFakeRepo to the FIRST
// Telegram user it sees — sequential ids start at 1. Every user-handler test
// authenticates exactly one user, so its sessions are seeded under this id.
const testUserID int64 = 1

// fakeSessionRepo is an in-memory repositories.SessionRepository for
// HTTP-level user-handler tests. Only the read paths the user API exercises
// are meaningfully implemented; write paths are inert stubs. When readErr is
// set every read returns it, which drives the 500 error-path tests.
type fakeSessionRepo struct {
	byUser  map[int64][]models.Session // newest-first per user
	weekly  []models.WeeklyTotal
	readErr error
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{byUser: map[int64][]models.Session{}}
}

func (r *fakeSessionRepo) GetActive(_ context.Context, userID int64) (*models.Session, error) {
	if r.readErr != nil {
		return nil, r.readErr
	}
	for _, s := range r.byUser[userID] {
		if s.IsActive {
			cp := s
			return &cp, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (r *fakeSessionRepo) ListByUserPaged(_ context.Context, userID int64, limit, offset int) ([]models.Session, error) {
	if r.readErr != nil {
		return nil, r.readErr
	}
	all := r.byUser[userID]
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return append([]models.Session(nil), all[offset:end]...), nil
}

func (r *fakeSessionRepo) CountByUser(_ context.Context, userID int64) (int64, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	return int64(len(r.byUser[userID])), nil
}

func (r *fakeSessionRepo) CompletedSessionCount(_ context.Context, userID int64) (int64, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	var n int64
	for _, s := range r.byUser[userID] {
		if !s.IsActive {
			n++
		}
	}
	return n, nil
}

func (r *fakeSessionRepo) TotalCompletedMinutes(_ context.Context, userID int64) (int, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	total := 0
	for _, s := range r.byUser[userID] {
		if !s.IsActive {
			total += s.DurationMinutes
		}
	}
	return total, nil
}

func (r *fakeSessionRepo) WeeklyTotals(_ context.Context, _, _ time.Time) ([]models.WeeklyTotal, error) {
	if r.readErr != nil {
		return nil, r.readErr
	}
	return r.weekly, nil
}

// --- inert write/unused paths ------------------------------------------------

func (r *fakeSessionRepo) Create(context.Context, int64, time.Time) (*models.Session, error) {
	return nil, errors.New("not implemented in fake")
}
func (r *fakeSessionRepo) Finish(context.Context, int64, time.Time, int) error {
	return errors.New("not implemented in fake")
}
func (r *fakeSessionRepo) ListOverlapping(context.Context, int64, time.Time, time.Time) ([]models.Session, error) {
	return nil, nil
}
func (r *fakeSessionRepo) GetByID(context.Context, int64) (*models.Session, error) {
	return nil, repositories.ErrNotFound
}
func (r *fakeSessionRepo) ListByUser(context.Context, int64, int) ([]models.Session, error) {
	return nil, nil
}

// newTestUserAPI wires a UserAPIService over the given fake session repo.
func newTestUserAPI(repo repositories.SessionRepository) *services.UserAPIService {
	return services.NewUserAPIService(repo, services.NewLeaderboardService(repo, time.UTC))
}

// newUserTestServer builds a routable API server whose user endpoints are
// backed by the given fake session repo.
func newUserTestServer(sessionRepo *fakeSessionRepo) http.Handler {
	userRepo := newAPIFakeRepo()
	users := services.NewUserService(userRepo, 0) // 0 → no admin auto-promotion
	admin := services.NewAdminService(nil, nil, nil)
	s := New("0", testBotToken, "*", users, newTestUserAPI(sessionRepo), admin, nil, logger.New("error"))
	return s.routes()
}

// authGet performs an authenticated GET as a single fixed Telegram user.
func authGet(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	initData := buildInitData(testBotToken, TelegramUser{ID: 555, FirstName: "Aru"}, time.Now())
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header = authHeader(initData)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeData unmarshals a success envelope's data into v, failing the test on
// any error or a non-success envelope.
func decodeData(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	var env Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%q)", err, rec.Body.String())
	}
	if !env.Success {
		t.Fatalf("expected success envelope, got error %q", env.Error)
	}
	raw, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("re-marshal data: %v", err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("decode data into %T: %v", v, err)
	}
}

// --- GET /api/v1/user/me -----------------------------------------------------

func TestUserMe_ReturnsProfileWithActiveSession(t *testing.T) {
	repo := newFakeSessionRepo()
	repo.byUser[testUserID] = []models.Session{
		{ID: 9, UserID: testUserID, IsActive: true, StartedAt: time.Now()},
		{ID: 2, UserID: testUserID, DurationMinutes: 45, IsValid: true, EndedAt: time.Now()},
		{ID: 1, UserID: testUserID, DurationMinutes: 30, IsValid: true, EndedAt: time.Now()},
	}

	rec := authGet(t, newUserTestServer(repo), "/api/v1/user/me")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200 (body=%q)", rec.Code, rec.Body.String())
	}

	var me userMeResponse
	decodeData(t, rec, &me)
	if me.TotalMinutes != 75 {
		t.Errorf("total_minutes: got %d want 75", me.TotalMinutes)
	}
	if me.TotalSessions != 2 {
		t.Errorf("total_sessions: got %d want 2 (completed only)", me.TotalSessions)
	}
	if me.ActiveSession == nil || !me.ActiveSession.IsActive || me.ActiveSession.ID != 9 {
		t.Errorf("active_session: got %+v want the active session id 9", me.ActiveSession)
	}
	if me.FirstName != "Aru" {
		t.Errorf("first_name: got %q want Aru", me.FirstName)
	}
}

func TestUserMe_NoActiveSession(t *testing.T) {
	repo := newFakeSessionRepo()
	repo.byUser[testUserID] = []models.Session{
		{ID: 1, UserID: testUserID, DurationMinutes: 30, EndedAt: time.Now()},
	}

	rec := authGet(t, newUserTestServer(repo), "/api/v1/user/me")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var me userMeResponse
	decodeData(t, rec, &me)
	if me.ActiveSession != nil {
		t.Errorf("active_session: got %+v want null", me.ActiveSession)
	}
}

func TestUserMe_RequiresAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	newUserTestServer(newFakeSessionRepo()).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /me: got %d want 401", rec.Code)
	}
	assertStructuredError(t, rec)
}

func TestUserMe_RepoErrorReturns500(t *testing.T) {
	repo := newFakeSessionRepo()
	repo.readErr = errors.New("db down")

	rec := authGet(t, newUserTestServer(repo), "/api/v1/user/me")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("repo error: got %d want 500", rec.Code)
	}
	assertStructuredError(t, rec)
}

// --- GET /api/v1/user/sessions ----------------------------------------------

func TestUserSessions_Paginated(t *testing.T) {
	repo := newFakeSessionRepo()
	sessions := make([]models.Session, 5)
	for i := range sessions {
		sessions[i] = models.Session{ID: int64(5 - i), UserID: testUserID, DurationMinutes: 30}
	}
	repo.byUser[testUserID] = sessions
	srv := newUserTestServer(repo)

	rec := authGet(t, srv, "/api/v1/user/sessions?page=1&limit=2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var page sessionsResponse
	decodeData(t, rec, &page)
	if len(page.Items) != 2 {
		t.Errorf("items: got %d want 2", len(page.Items))
	}
	if page.Pagination.Total != 5 || page.Pagination.TotalPages != 3 {
		t.Errorf("pagination: got total=%d pages=%d want 5/3",
			page.Pagination.Total, page.Pagination.TotalPages)
	}
	if page.Items[0].ID != 5 {
		t.Errorf("first item id: got %d want 5 (newest first)", page.Items[0].ID)
	}

	// Page 2 must continue from the right offset.
	rec2 := authGet(t, srv, "/api/v1/user/sessions?page=2&limit=2")
	var page2 sessionsResponse
	decodeData(t, rec2, &page2)
	if len(page2.Items) != 2 || page2.Items[0].ID != 3 {
		t.Errorf("page 2: got %d items, first id %d; want 2 items starting at id 3",
			len(page2.Items), firstID(page2.Items))
	}
}

func TestUserSessions_EmptyIsStructuredEmptyList(t *testing.T) {
	rec := authGet(t, newUserTestServer(newFakeSessionRepo()), "/api/v1/user/sessions")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var page sessionsResponse
	decodeData(t, rec, &page)
	if page.Items == nil {
		t.Error("items must be an empty array, not null")
	}
	if page.Pagination.Total != 0 {
		t.Errorf("total: got %d want 0", page.Pagination.Total)
	}
}

func TestUserSessions_InvalidPaginationReturns400(t *testing.T) {
	rec := authGet(t, newUserTestServer(newFakeSessionRepo()), "/api/v1/user/sessions?page=0")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid page: got %d want 400", rec.Code)
	}
	assertStructuredError(t, rec)
}

// --- GET /api/v1/user/leaderboard -------------------------------------------

func TestUserLeaderboard_ReturnsTopAndCallerPosition(t *testing.T) {
	repo := newFakeSessionRepo()
	repo.weekly = []models.WeeklyTotal{
		{UserID: 7, FirstName: "Bek", Minutes: 300},
		{UserID: testUserID, FirstName: "Aru", Minutes: 180},
		{UserID: 8, FirstName: "Dana", Minutes: 60},
	}

	rec := authGet(t, newUserTestServer(repo), "/api/v1/user/leaderboard")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	var lb leaderboardResponse
	decodeData(t, rec, &lb)
	if len(lb.Top) != 3 {
		t.Fatalf("top: got %d entries want 3", len(lb.Top))
	}
	if lb.Top[0].UserID != 7 || lb.Top[0].Rank != 1 {
		t.Errorf("rank 1: got user=%d rank=%d want user 7 rank 1", lb.Top[0].UserID, lb.Top[0].Rank)
	}
	if !lb.Me.Found || lb.Me.Rank != 2 || lb.Me.Minutes != 180 {
		t.Errorf("me: got %+v want found rank 2 minutes 180", lb.Me)
	}
	// The caller's own row must be flagged in the Top list.
	if !lb.Top[1].IsCurrent {
		t.Error("caller's entry in Top must have is_current=true")
	}
}

func TestUserLeaderboard_CallerNotRanked(t *testing.T) {
	repo := newFakeSessionRepo()
	repo.weekly = []models.WeeklyTotal{{UserID: 7, FirstName: "Bek", Minutes: 300}}

	rec := authGet(t, newUserTestServer(repo), "/api/v1/user/leaderboard")
	var lb leaderboardResponse
	decodeData(t, rec, &lb)
	if lb.Me.Found {
		t.Errorf("me.found: got true want false (caller has no minutes this week)")
	}
}

// --- unknown path ------------------------------------------------------------

func TestUserRoute_UnknownPathStructured404(t *testing.T) {
	rec := authGet(t, newUserTestServer(newFakeSessionRepo()), "/api/v1/user/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown user path: got %d want 404", rec.Code)
	}
	assertStructuredError(t, rec)
}

// firstID is a tiny test helper for clearer failure messages.
func firstID(items []sessionResponse) int64 {
	if len(items) == 0 {
		return 0
	}
	return items[0].ID
}
