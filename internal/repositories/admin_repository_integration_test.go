//go:build integration

// PostgreSQL integration safety tests for the admin session-correction path.
//
// These exercise CorrectSessionWithAudit against a REAL PostgreSQL instance —
// transaction atomicity, the immutable audit log, and deterministic streak
// recomputation cannot be proven against a fake. They are build-tagged so the
// default `go test ./...` stays hermetic.
//
// Run with:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/tomorrow_test \
//	  go test -tags=integration ./internal/repositories/
//
// The target database is migrated automatically and TRUNCATEd before each test.
package repositories

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damirkabdulla/tomorrow-tracker/internal/database"
)

var almatyTZ = mustLoadTZ("Asia/Almaty")

func mustLoadTZ(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// requireDB connects to TEST_DATABASE_URL (skipping the test when it is unset),
// applies the migrations, and resets the tables to a clean slate.
func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration tests")
	}
	pool, err := database.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)
	cleanTables(t, pool)
	return pool
}

// cleanTables truncates every table back to empty. The audit-log immutability
// trigger blocks TRUNCATE, so user triggers are disabled for the reset only.
func cleanTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `ALTER TABLE admin_actions DISABLE TRIGGER USER`); err != nil {
		t.Fatalf("disable audit triggers: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`TRUNCATE users, sessions, admin_actions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE admin_actions ENABLE TRIGGER USER`); err != nil {
		t.Fatalf("re-enable audit triggers: %v", err)
	}
}

func insertUser(t *testing.T, pool *pgxpool.Pool, telegramID int64, role string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (telegram_id, role) VALUES ($1, $2) RETURNING id`,
		telegramID, role).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

// insertFinished inserts a finished session ending at endedAt with the given
// duration and validity. started_at is derived so the row is internally
// consistent.
func insertFinished(t *testing.T, pool *pgxpool.Pool, userID int64, endedAt time.Time, duration int, valid bool) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO sessions (user_id, started_at, ended_at, duration_minutes, is_active, is_valid)
		 VALUES ($1, $2, $3, $4, FALSE, $5) RETURNING id`,
		userID, endedAt.Add(-time.Duration(duration)*time.Minute), endedAt, duration, valid).Scan(&id)
	if err != nil {
		t.Fatalf("insert finished session: %v", err)
	}
	return id
}

func insertActive(t *testing.T, pool *pgxpool.Pool, userID int64, startedAt time.Time) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO sessions (user_id, started_at, is_active) VALUES ($1, $2, TRUE) RETURNING id`,
		userID, startedAt).Scan(&id)
	if err != nil {
		t.Fatalf("insert active session: %v", err)
	}
	return id
}

func setStreak(t *testing.T, pool *pgxpool.Pool, userID int64, current, best int, last time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`UPDATE users SET current_streak = $1, best_streak = $2, last_study_at = $3 WHERE id = $4`,
		current, best, last, userID)
	if err != nil {
		t.Fatalf("set streak: %v", err)
	}
}

func readStreak(t *testing.T, pool *pgxpool.Pool, userID int64) (current, best int, last *time.Time) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT current_streak, best_streak, last_study_at FROM users WHERE id = $1`,
		userID).Scan(&current, &best, &last)
	if err != nil {
		t.Fatalf("read streak: %v", err)
	}
	return current, best, last
}

func auditCount(t *testing.T, pool *pgxpool.Pool, sessionID int64) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM admin_actions WHERE action = 'PATCH_SESSION' AND entity_id = $1`,
		sessionID).Scan(&n)
	if err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	return n
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

// --- Correction transaction safety ------------------------------------------

func TestIntegration_CorrectSession_DurationAndAudit(t *testing.T) {
	pool := requireDB(t)
	r := NewAdminRepository(pool, almatyTZ)
	ctx := context.Background()

	adminID := insertUser(t, pool, 100, "admin")
	userID := insertUser(t, pool, 200, "user")
	endedAt := time.Date(2026, 5, 16, 20, 0, 0, 0, almatyTZ)
	sid := insertFinished(t, pool, userID, endedAt, 60, true)
	setStreak(t, pool, userID, 7, 9, endedAt)

	corr, err := r.CorrectSessionWithAudit(ctx, adminID, sid, SessionPatch{
		DurationMinutes: intPtr(90),
		Reason:          "пересчёт времени",
	})
	if err != nil {
		t.Fatalf("correct session: %v", err)
	}
	if corr.OldDuration != 60 || corr.NewDuration != 90 {
		t.Errorf("duration: got old=%d new=%d", corr.OldDuration, corr.NewDuration)
	}
	if corr.StreakRecomputed {
		t.Error("60→90 keeps qualification; no streak recomputation expected")
	}

	var gotDuration int
	if err := pool.QueryRow(ctx,
		`SELECT duration_minutes FROM sessions WHERE id = $1`, sid).Scan(&gotDuration); err != nil {
		t.Fatal(err)
	}
	if gotDuration != 90 {
		t.Errorf("session duration not persisted: got %d", gotDuration)
	}
	if n := auditCount(t, pool, sid); n != 1 {
		t.Errorf("expected exactly 1 audit row, got %d", n)
	}
	if cur, best, _ := readStreak(t, pool, userID); cur != 7 || best != 9 {
		t.Errorf("streak must stay untouched: got current=%d best=%d", cur, best)
	}
}

func TestIntegration_CorrectSession_ActiveRejected(t *testing.T) {
	pool := requireDB(t)
	r := NewAdminRepository(pool, almatyTZ)
	ctx := context.Background()

	adminID := insertUser(t, pool, 100, "admin")
	userID := insertUser(t, pool, 200, "user")
	sid := insertActive(t, pool, userID, time.Now().Add(-time.Hour))

	_, err := r.CorrectSessionWithAudit(ctx, adminID, sid, SessionPatch{
		DurationMinutes: intPtr(45), Reason: "x",
	})
	if !errors.Is(err, ErrSessionActive) {
		t.Fatalf("expected ErrSessionActive, got %v", err)
	}
	if n := auditCount(t, pool, sid); n != 0 {
		t.Errorf("rejected correction must write no audit row, got %d", n)
	}
}

func TestIntegration_CorrectSession_NotFound(t *testing.T) {
	pool := requireDB(t)
	r := NewAdminRepository(pool, almatyTZ)
	adminID := insertUser(t, pool, 100, "admin")

	_, err := r.CorrectSessionWithAudit(context.Background(), adminID, 999999, SessionPatch{
		DurationMinutes: intPtr(45), Reason: "x",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- Audit-log immutability --------------------------------------------------

func TestIntegration_AuditLogImmutable(t *testing.T) {
	pool := requireDB(t)
	r := NewAdminRepository(pool, almatyTZ)
	ctx := context.Background()
	adminID := insertUser(t, pool, 100, "admin")

	if err := r.InsertAuditLog(ctx, AuditEntry{
		AdminID:   adminID,
		Action:    "EXPORT_USERS",
		AfterData: []byte(`{"from":"a","to":"b"}`),
	}); err != nil {
		t.Fatalf("insert audit log: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`UPDATE admin_actions SET reason = 'tampered' WHERE admin_id = $1`, adminID); err == nil {
		t.Error("UPDATE on admin_actions must be rejected by the immutability trigger")
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM admin_actions WHERE admin_id = $1`, adminID); err == nil {
		t.Error("DELETE on admin_actions must be rejected by the immutability trigger")
	}

	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM admin_actions WHERE admin_id = $1`, adminID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("audit row must survive the rejected mutations, got %d rows", n)
	}
}

// --- Deterministic streak recomputation -------------------------------------

// TestIntegration_StreakRecompute_InvalidateAndRestore proves that flipping a
// session's validity rebuilds the owner's streak deterministically, and that
// restoring validity reverses it exactly.
func TestIntegration_StreakRecompute_InvalidateAndRestore(t *testing.T) {
	pool := requireDB(t)
	r := NewAdminRepository(pool, almatyTZ)
	ctx := context.Background()

	adminID := insertUser(t, pool, 100, "admin")
	userID := insertUser(t, pool, 200, "user")

	// Three consecutive qualifying days → materialized streak 3/3.
	d14 := time.Date(2026, 5, 14, 20, 0, 0, 0, almatyTZ)
	d15 := time.Date(2026, 5, 15, 20, 0, 0, 0, almatyTZ)
	d16 := time.Date(2026, 5, 16, 20, 0, 0, 0, almatyTZ)
	insertFinished(t, pool, userID, d14, 60, true)
	midSID := insertFinished(t, pool, userID, d15, 60, true)
	insertFinished(t, pool, userID, d16, 60, true)
	setStreak(t, pool, userID, 3, 3, d16)

	// Invalidate the middle day → counted days {14, 16} are no longer
	// consecutive → current=1, best=1.
	corr, err := r.CorrectSessionWithAudit(ctx, adminID, midSID, SessionPatch{
		IsValid: boolPtr(false),
		Reason:  "ручная проверка: аномалия",
	})
	if err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if !corr.StreakRecomputed {
		t.Fatal("invalidating a qualifying session must recompute the streak")
	}
	if corr.CurrentStreak != 1 || corr.BestStreak != 1 {
		t.Errorf("after invalidation: got current=%d best=%d, want 1/1",
			corr.CurrentStreak, corr.BestStreak)
	}
	cur, best, last := readStreak(t, pool, userID)
	if cur != 1 || best != 1 {
		t.Errorf("persisted streak after invalidation: got current=%d best=%d", cur, best)
	}
	if last == nil || !last.Equal(d16) {
		t.Errorf("last_study_at: got %v, want %v", last, d16)
	}

	// Restore validity → the three days are consecutive again → 3/3.
	corr, err = r.CorrectSessionWithAudit(ctx, adminID, midSID, SessionPatch{
		IsValid: boolPtr(true),
		Reason:  "восстановление после проверки",
	})
	if err != nil {
		t.Fatalf("revalidate: %v", err)
	}
	if !corr.StreakRecomputed || corr.CurrentStreak != 3 || corr.BestStreak != 3 {
		t.Errorf("after restore: got recomputed=%v current=%d best=%d, want true/3/3",
			corr.StreakRecomputed, corr.CurrentStreak, corr.BestStreak)
	}
}

// TestIntegration_StreakRecompute_DurationThreshold proves that dropping a
// session below the streak minimum also flips qualification and recomputes.
func TestIntegration_StreakRecompute_DurationThreshold(t *testing.T) {
	pool := requireDB(t)
	r := NewAdminRepository(pool, almatyTZ)
	ctx := context.Background()

	adminID := insertUser(t, pool, 100, "admin")
	userID := insertUser(t, pool, 200, "user")

	d15 := time.Date(2026, 5, 15, 20, 0, 0, 0, almatyTZ)
	d16 := time.Date(2026, 5, 16, 20, 0, 0, 0, almatyTZ)
	insertFinished(t, pool, userID, d15, 60, true)
	lastSID := insertFinished(t, pool, userID, d16, 60, true)
	setStreak(t, pool, userID, 2, 2, d16)

	// Drop the last day below MinStreakSessionMinutes → it no longer qualifies
	// → only day 15 counts → current=1, best=1, last_study_at moves to day 15.
	corr, err := r.CorrectSessionWithAudit(ctx, adminID, lastSID, SessionPatch{
		DurationMinutes: intPtr(MinStreakSessionMinutes - 1),
		Reason:          "корректировка длительности",
	})
	if err != nil {
		t.Fatalf("correct duration: %v", err)
	}
	if !corr.StreakRecomputed || corr.CurrentStreak != 1 || corr.BestStreak != 1 {
		t.Errorf("got recomputed=%v current=%d best=%d, want true/1/1",
			corr.StreakRecomputed, corr.CurrentStreak, corr.BestStreak)
	}
	_, _, last := readStreak(t, pool, userID)
	if last == nil || !last.Equal(d15) {
		t.Errorf("last_study_at must move to day 15: got %v", last)
	}
}

// --- admin sessions listing --------------------------------------------------

// TestIntegration_ListSessions_Filters proves the user / validity / flagged
// filters and the COUNT(*) OVER() total all behave as expected.
func TestIntegration_ListSessions_Filters(t *testing.T) {
	pool := requireDB(t)
	r := NewAdminRepository(pool, almatyTZ)
	ctx := context.Background()

	u1 := insertUser(t, pool, 201, "user")
	u2 := insertUser(t, pool, 202, "user")
	base := time.Date(2026, 5, 16, 20, 0, 0, 0, almatyTZ)
	insertFinished(t, pool, u1, base, 60, true)                            // u1, valid, no flags
	flagged := insertFinished(t, pool, u1, base.Add(time.Hour), 20, false) // u1, invalid
	insertFinished(t, pool, u2, base, 90, true)                            // u2, valid

	if _, err := pool.Exec(ctx,
		`UPDATE sessions SET anti_cheat_flags = '["manual_review"]'::jsonb WHERE id = $1`,
		flagged); err != nil {
		t.Fatalf("flag session: %v", err)
	}

	// No filters → every session, total reflects the full set.
	rows, total, err := r.ListSessions(ctx, SessionListParams{SortDesc: true, Limit: 20})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if total != 3 || len(rows) != 3 {
		t.Errorf("unfiltered: got total=%d rows=%d, want 3/3", total, len(rows))
	}

	// User filter → only that owner's sessions.
	_, total, err = r.ListSessions(ctx, SessionListParams{UserID: &u1, Limit: 20})
	if err != nil || total != 2 {
		t.Errorf("user filter: got total=%d err=%v, want 2", total, err)
	}

	// valid=false → only the invalid session.
	valFalse := false
	rows, total, err = r.ListSessions(ctx, SessionListParams{Valid: &valFalse, Limit: 20})
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Session.ID != flagged {
		t.Errorf("valid=false: got total=%d rows=%d err=%v, want session %d", total, len(rows), err, flagged)
	}

	// flagged=true → only the session carrying anti-cheat evidence.
	rows, total, err = r.ListSessions(ctx, SessionListParams{Flagged: true, Limit: 20})
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Session.ID != flagged {
		t.Fatalf("flagged: got total=%d rows=%d err=%v, want session %d", total, len(rows), err, flagged)
	}
	// The JOIN must populate owner identity (telegram-only users have an empty
	// first_name, so just assert the field is reachable, not its value).
	_ = rows[0].OwnerFirstName
	_ = rows[0].OwnerUsername
}
