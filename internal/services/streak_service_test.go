package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// almaty is the project timezone; loaded once for all tests.
var almaty = mustLoad("Asia/Almaty")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// at builds a time in Asia/Almaty for terse table-test setup.
func at(y, m, d, hh, mm int) time.Time {
	return time.Date(y, time.Month(m), d, hh, mm, 0, 0, almaty)
}

// ptrT is a pointer helper for assigning *time.Time fields inline.
func ptrT(t time.Time) *time.Time { return &t }

// --- Pure-function tests for computeStreakUpdate -----------------------------
//
// These cover the algorithm in isolation, no DB / no service. Every required
// edge case from the spec lives here as a separate row.

func TestComputeStreakUpdate(t *testing.T) {
	cases := []struct {
		name    string
		state   StreakState
		endedAt time.Time
		want    StreakUpdate
	}{
		{
			name:    "first ever session",
			state:   StreakState{Current: 0, Best: 0, LastStudyAt: nil},
			endedAt: at(2026, 5, 16, 21, 30),
			want:    StreakUpdate{Counted: true, Current: 1, Best: 1, Continued: true, NewRecord: true},
		},
		{
			name:    "consecutive day grows streak",
			state:   StreakState{Current: 6, Best: 6, LastStudyAt: ptrT(at(2026, 5, 15, 22, 0))},
			endedAt: at(2026, 5, 16, 21, 30),
			want:    StreakUpdate{Counted: true, Current: 7, Best: 7, Continued: true, NewRecord: true},
		},
		{
			name:    "consecutive day, best already higher",
			state:   StreakState{Current: 3, Best: 11, LastStudyAt: ptrT(at(2026, 5, 15, 22, 0))},
			endedAt: at(2026, 5, 16, 21, 30),
			want:    StreakUpdate{Counted: true, Current: 4, Best: 11, Continued: true, NewRecord: false},
		},
		{
			name:    "same day duplicate session — no change",
			state:   StreakState{Current: 4, Best: 11, LastStudyAt: ptrT(at(2026, 5, 16, 8, 0))},
			endedAt: at(2026, 5, 16, 21, 30),
			want:    StreakUpdate{Counted: true, Current: 4, Best: 11, SameDay: true},
		},
		{
			name:    "same exact timestamp (idempotency)",
			state:   StreakState{Current: 4, Best: 11, LastStudyAt: ptrT(at(2026, 5, 16, 21, 30))},
			endedAt: at(2026, 5, 16, 21, 30),
			want:    StreakUpdate{Counted: true, Current: 4, Best: 11, SameDay: true},
		},
		{
			name:    "missed one day — reset to 1",
			state:   StreakState{Current: 7, Best: 11, LastStudyAt: ptrT(at(2026, 5, 14, 22, 0))},
			endedAt: at(2026, 5, 16, 9, 0),
			want:    StreakUpdate{Counted: true, Current: 1, Best: 11, Broken: true},
		},
		{
			name:    "missed many days — reset to 1",
			state:   StreakState{Current: 7, Best: 11, LastStudyAt: ptrT(at(2026, 1, 1, 12, 0))},
			endedAt: at(2026, 5, 16, 9, 0),
			want:    StreakUpdate{Counted: true, Current: 1, Best: 11, Broken: true},
		},
		{
			name:    "session ends past midnight — counted on end-day",
			state:   StreakState{Current: 5, Best: 5, LastStudyAt: ptrT(at(2026, 5, 15, 23, 30))},
			endedAt: at(2026, 5, 16, 0, 30), // ended 30 min after midnight → next day
			want:    StreakUpdate{Counted: true, Current: 6, Best: 6, Continued: true, NewRecord: true},
		},
		{
			name: "out-of-order end timestamp (defensive) — treated as same day",
			// LastStudyAt is in the future relative to endedAt; diff <= 0 path.
			state:   StreakState{Current: 9, Best: 9, LastStudyAt: ptrT(at(2026, 5, 16, 23, 0))},
			endedAt: at(2026, 5, 16, 8, 0),
			want:    StreakUpdate{Counted: true, Current: 9, Best: 9, SameDay: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeStreakUpdate(tc.state, tc.endedAt, almaty)
			if got != tc.want {
				t.Fatalf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

// --- Service-level tests via fake UserRepository ----------------------------
//
// These verify the >=30 minute gate, persistence behavior, and the same-day
// idempotency contract end-to-end.

type fakeUserRepo struct {
	user      *models.User
	updates   int
	updateErr error
}

func (r *fakeUserRepo) GetByTelegramID(_ context.Context, _ int64) (*models.User, error) {
	return nil, errors.New("not used")
}

func (r *fakeUserRepo) GetByID(_ context.Context, id int64) (*models.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, repositories.ErrNotFound
	}
	cp := *r.user
	if r.user.LastStudyAt != nil {
		t := *r.user.LastStudyAt
		cp.LastStudyAt = &t
	}
	return &cp, nil
}

func (r *fakeUserRepo) Upsert(_ context.Context, telegramID int64, username, firstName string) (*models.User, error) {
	if r.user != nil && r.user.TelegramID == telegramID {
		r.user.Username = username
		r.user.FirstName = firstName
	} else {
		id := int64(1)
		role := models.RoleUser
		r.user = &models.User{
			ID: id, TelegramID: telegramID, Username: username,
			FirstName: firstName, Role: role,
		}
	}
	cp := *r.user
	return &cp, nil
}

func (r *fakeUserRepo) UpdateRole(_ context.Context, userID int64, role string) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.user == nil || r.user.ID != userID {
		return repositories.ErrNotFound
	}
	r.user.Role = role
	return nil
}

func (r *fakeUserRepo) UpdateStreak(_ context.Context, userID int64, current, best int, lastStudyAt time.Time) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.user == nil || r.user.ID != userID {
		return repositories.ErrNotFound
	}
	r.updates++
	r.user.CurrentStreak = current
	r.user.BestStreak = best
	t := lastStudyAt
	r.user.LastStudyAt = &t
	return nil
}

func TestRecordCompletedSession_BelowMinimumDoesNotPersist(t *testing.T) {
	repo := &fakeUserRepo{user: &models.User{ID: 1, CurrentStreak: 5, BestStreak: 5}}
	svc := NewStreakService(repo, almaty)

	upd, err := svc.RecordCompletedSession(context.Background(), 1, 29, at(2026, 5, 16, 21, 0))
	if err != nil {
		t.Fatal(err)
	}
	if upd.Counted {
		t.Error("expected Counted=false for sub-30-minute session")
	}
	if repo.updates != 0 {
		t.Errorf("expected 0 persistence calls, got %d", repo.updates)
	}
	if upd.Current != 5 || upd.Best != 5 {
		t.Errorf("snapshot mismatch: current=%d best=%d", upd.Current, upd.Best)
	}
}

func TestRecordCompletedSession_FirstSessionPersists(t *testing.T) {
	repo := &fakeUserRepo{user: &models.User{ID: 1}}
	svc := NewStreakService(repo, almaty)

	upd, err := svc.RecordCompletedSession(context.Background(), 1, 45, at(2026, 5, 16, 21, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !upd.Counted || upd.Current != 1 || upd.Best != 1 || !upd.NewRecord {
		t.Errorf("first session expected Current=Best=1 NewRecord=true Counted=true, got %+v", upd)
	}
	if repo.updates != 1 {
		t.Errorf("expected 1 persistence call, got %d", repo.updates)
	}
}

func TestRecordCompletedSession_SameDayIdempotent(t *testing.T) {
	repo := &fakeUserRepo{user: &models.User{ID: 1}}
	svc := NewStreakService(repo, almaty)
	ctx := context.Background()

	// Day 1: first session → persists.
	if _, err := svc.RecordCompletedSession(ctx, 1, 60, at(2026, 5, 16, 9, 0)); err != nil {
		t.Fatal(err)
	}
	persistedAfterFirst := repo.updates

	// Same day, second session → must NOT persist again.
	upd, err := svc.RecordCompletedSession(ctx, 1, 60, at(2026, 5, 16, 21, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !upd.SameDay {
		t.Errorf("expected SameDay=true on second same-day session, got %+v", upd)
	}
	if repo.updates != persistedAfterFirst {
		t.Errorf("expected no extra persistence (still %d), got %d", persistedAfterFirst, repo.updates)
	}

	// Same exact timestamp — duplicate call (e.g. retry) — also a no-op.
	upd, err = svc.RecordCompletedSession(ctx, 1, 60, at(2026, 5, 16, 21, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !upd.SameDay {
		t.Errorf("expected SameDay=true on identical retry, got %+v", upd)
	}
	if repo.updates != persistedAfterFirst {
		t.Errorf("expected no extra persistence on retry, got %d", repo.updates)
	}
}

func TestRecordCompletedSession_ConsecutiveDays(t *testing.T) {
	repo := &fakeUserRepo{user: &models.User{ID: 1}}
	svc := NewStreakService(repo, almaty)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		end := at(2026, 5, 12+i, 21, 0)
		upd, err := svc.RecordCompletedSession(ctx, 1, 45, end)
		if err != nil {
			t.Fatal(err)
		}
		if upd.Current != i+1 {
			t.Errorf("day %d: expected current=%d, got %d", i, i+1, upd.Current)
		}
	}
	if repo.user.BestStreak != 5 {
		t.Errorf("expected best=5, got %d", repo.user.BestStreak)
	}
}

func TestRecordCompletedSession_MissedDayResets(t *testing.T) {
	repo := &fakeUserRepo{user: &models.User{
		ID:            1,
		CurrentStreak: 7,
		BestStreak:    11,
		LastStudyAt:   ptrT(at(2026, 5, 13, 22, 0)),
	}}
	svc := NewStreakService(repo, almaty)

	upd, err := svc.RecordCompletedSession(context.Background(), 1, 45, at(2026, 5, 16, 9, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !upd.Broken || upd.Current != 1 || upd.Best != 11 {
		t.Errorf("expected Broken=true Current=1 Best=11, got %+v", upd)
	}
}
