package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestUserRateLimit_PerUser_NotPerIP is the regression guard for the Mini App
// stall: many DISTINCT users sharing one Wi-Fi egress IP must each get their
// own budget. Before the fix the user group was IP-keyed at 60/min, so a
// school behind one NAT was throttled as a single caller.
func TestUserRateLimit_PerUser_NotPerIP(t *testing.T) {
	srv := newTestServer(1)

	// 200 distinct students, all from the same campus NAT IP, one request each.
	// 200 > the old 60/min IP bucket — this would 429 on the pre-fix code.
	const sharedNAT = "203.0.113.7:55000"
	for i := int64(1); i <= 200; i++ {
		initData := buildInitData(testBotToken, TelegramUser{ID: 100000 + i, FirstName: "S"}, time.Now())
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil)
		req.Header = authHeader(initData)
		req.RemoteAddr = sharedNAT

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("user %d throttled though each user is a distinct identity (shared NAT must not matter)", 100000+i)
		}
	}
}

// TestUserRateLimit_ThrottlesSingleUserOverBudget proves the per-user budget is
// still enforced: one user firing past userRatePerMin gets a 429.
func TestUserRateLimit_ThrottlesSingleUserOverBudget(t *testing.T) {
	srv := newTestServer(1)
	initData := buildInitData(testBotToken, TelegramUser{ID: 9001, FirstName: "S"}, time.Now())

	throttled := false
	for i := 0; i < userRatePerMin+5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil)
		req.Header = authHeader(initData)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			throttled = true
			break
		}
	}
	if !throttled {
		t.Fatalf("a single user exceeding the per-user budget (%d/min) must be throttled", userRatePerMin)
	}
}

// TestAuthVerifyRateLimit_PerInitDataUser proves the login endpoint is keyed by
// the Telegram user in the initData, not by IP: a whole class launching the
// Mini App from one Wi-Fi must all be able to log in.
func TestAuthVerifyRateLimit_PerInitDataUser(t *testing.T) {
	srv := newTestServer(1)
	const sharedNAT = "203.0.113.8:44000"

	for i := int64(1); i <= 50; i++ {
		initData := buildInitData(testBotToken, TelegramUser{ID: 200000 + i, FirstName: "S"}, time.Now())
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify", nil)
		req.Header = authHeader(initData)
		req.RemoteAddr = sharedNAT

		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("auth/verify throttled user %d — login must be per-user, not per shared IP", 200000+i)
		}
	}
}

func TestInitDataUserID(t *testing.T) {
	raw := buildInitData(testBotToken, TelegramUser{ID: 4242, FirstName: "S"}, time.Now())
	if got := initDataUserID(raw); got != 4242 {
		t.Fatalf("initDataUserID = %d, want 4242", got)
	}
	if got := initDataUserID("%ZZ-not-a-query"); got != 0 {
		t.Fatalf("initDataUserID(garbage) = %d, want 0", got)
	}
	if got := initDataUserID("auth_date=1&hash=x"); got != 0 {
		t.Fatalf("initDataUserID(no user field) = %d, want 0", got)
	}
}

// TestStatusRecorder_DefaultsTo200 covers the access-log status capture: a
// handler that writes a body without an explicit WriteHeader is a 200.
func TestStatusRecorder_DefaultsTo200(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	if _, err := rec.Write([]byte("ok")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if rec.status != http.StatusOK {
		t.Fatalf("implicit status = %d, want 200", rec.status)
	}
	if rec.bytes != 2 {
		t.Fatalf("byte count = %d, want 2", rec.bytes)
	}
}
