//go:build integration

// PostgreSQL integration tests for the session-lifecycle invariants added in
// Phase 3C: the "one active session per user" partial unique index and the
// ownership-guarded finish path.
//
// These exercise the real database — a partial index and a concurrent INSERT
// race cannot be proven against a fake. Build-tagged so the default
// `go test ./...` stays hermetic.
//
// Run with:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5432/tomorrow_test \
//	  go test -tags=integration ./internal/repositories/
//
// Shared helpers (requireDB, insertUser, insertActive, ...) live in
// admin_repository_integration_test.go — same package, same build tag.
package repositories

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// --- migration integrity -----------------------------------------------------

// TestIntegration_OneActiveIndex_Definition confirms the migration created the
// partial unique index with exactly the expected shape: UNIQUE, on
// sessions(user_id), restricted to WHERE is_active.
func TestIntegration_OneActiveIndex_Definition(t *testing.T) {
	pool := requireDB(t)

	var indexDef string
	err := pool.QueryRow(context.Background(),
		`SELECT indexdef FROM pg_indexes WHERE indexname = 'uq_sessions_one_active'`).
		Scan(&indexDef)
	if err != nil {
		t.Fatalf("uq_sessions_one_active not found — migration did not apply: %v", err)
	}

	for _, want := range []string{"UNIQUE INDEX", "sessions", "user_id", "is_active"} {
		if !strings.Contains(indexDef, want) {
			t.Errorf("index definition missing %q\n  got: %s", want, indexDef)
		}
	}
}

// --- one active session per user --------------------------------------------

// TestIntegration_Create_RejectsSecondActive proves the repository surfaces a
// typed ErrActiveSessionExists when a user already has an active session,
// rather than silently opening a second one.
func TestIntegration_Create_RejectsSecondActive(t *testing.T) {
	pool := requireDB(t)
	r := NewSessionRepository(pool)
	ctx := context.Background()

	userID := insertUser(t, pool, 200, "user")
	if _, err := r.Create(ctx, userID, time.Now()); err != nil {
		t.Fatalf("first Create must succeed: %v", err)
	}

	_, err := r.Create(ctx, userID, time.Now())
	if !errors.Is(err, ErrActiveSessionExists) {
		t.Fatalf("second Create: got %v, want ErrActiveSessionExists", err)
	}

	if n := activeCount(t, pool, userID); n != 1 {
		t.Errorf("user must have exactly 1 active session, got %d", n)
	}
}

// TestIntegration_Create_ConcurrentRace fires many simultaneous Create calls
// for the same user. The partial unique index must guarantee exactly one
// winner — the rest fail with ErrActiveSessionExists, never a second row.
func TestIntegration_Create_ConcurrentRace(t *testing.T) {
	pool := requireDB(t)
	r := NewSessionRepository(pool)
	ctx := context.Background()

	userID := insertUser(t, pool, 200, "user")

	const goroutines = 16
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		conflicts int
		others    []error
	)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release all goroutines at once to maximise contention
			_, err := r.Create(ctx, userID, time.Now())
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, ErrActiveSessionExists):
				conflicts++
			default:
				others = append(others, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(others) > 0 {
		t.Fatalf("unexpected errors from concurrent Create: %v", others)
	}
	if succeeded != 1 {
		t.Errorf("exactly one Create must win, got %d winners", succeeded)
	}
	if conflicts != goroutines-1 {
		t.Errorf("losers: got %d, want %d", conflicts, goroutines-1)
	}
	if n := activeCount(t, pool, userID); n != 1 {
		t.Errorf("database must hold exactly 1 active session, got %d", n)
	}
}

// TestIntegration_Create_AllowsActiveAfterFinish proves the partial index only
// constrains ACTIVE rows: once a session is finished, a new one may start.
func TestIntegration_Create_AllowsActiveAfterFinish(t *testing.T) {
	pool := requireDB(t)
	r := NewSessionRepository(pool)
	ctx := context.Background()

	userID := insertUser(t, pool, 200, "user")
	first, err := r.Create(ctx, userID, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := r.FinishOwned(ctx, first.ID, userID, time.Now(), 60); err != nil {
		t.Fatalf("FinishOwned: %v", err)
	}
	if _, err := r.Create(ctx, userID, time.Now()); err != nil {
		t.Fatalf("Create after finish must succeed: %v", err)
	}
}

// --- ownership-guarded finish ------------------------------------------------

// TestIntegration_FinishOwned_Guards covers the three rejection paths and the
// happy path of the ownership-guarded UPDATE.
func TestIntegration_FinishOwned_Guards(t *testing.T) {
	pool := requireDB(t)
	r := NewSessionRepository(pool)
	ctx := context.Background()

	owner := insertUser(t, pool, 200, "user")
	intruder := insertUser(t, pool, 300, "user")
	sid := insertActive(t, pool, owner, time.Now().Add(-time.Hour))

	// Wrong owner — must not finish another user's session.
	if err := r.FinishOwned(ctx, sid, intruder, time.Now(), 60); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign finish: got %v, want ErrNotFound", err)
	}
	if n := activeCount(t, pool, owner); n != 1 {
		t.Errorf("owner's session must still be active after a foreign finish, got %d active", n)
	}

	// Unknown id.
	if err := r.FinishOwned(ctx, 999999, owner, time.Now(), 60); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id finish: got %v, want ErrNotFound", err)
	}

	// Correct owner — succeeds exactly once.
	if err := r.FinishOwned(ctx, sid, owner, time.Now(), 60); err != nil {
		t.Fatalf("owner finish must succeed: %v", err)
	}
	// Second finish of the same (now closed) session — idempotent rejection.
	if err := r.FinishOwned(ctx, sid, owner, time.Now(), 60); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double finish: got %v, want ErrNotFound", err)
	}
}

// --- helpers -----------------------------------------------------------------

// activeCount returns how many active sessions a user currently has.
func activeCount(t *testing.T, pool *pgxpool.Pool, userID int64) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM sessions WHERE user_id = $1 AND is_active`, userID).Scan(&n); err != nil {
		t.Fatalf("count active sessions: %v", err)
	}
	return n
}
