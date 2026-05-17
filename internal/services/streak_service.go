package services

import (
	"context"
	"fmt"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
	"github.com/damirkabdulla/tomorrow-tracker/internal/utils"
)

// MinStreakSessionMinutes is the minimum session length (in minutes) that
// counts toward a user's daily streak. Shorter sessions are ignored entirely
// — they do not advance, reset, or refresh the streak. The canonical value
// lives in the repositories layer so the streak service and the admin
// streak-recomputation query share a single source of truth.
const MinStreakSessionMinutes = repositories.MinStreakSessionMinutes

// StreakUpdate is the outcome of evaluating a completed session against the
// streak rules. Counted=false means the session did not qualify (too short)
// and the user's streak is unchanged.
type StreakUpdate struct {
	Counted   bool
	Current   int  // streak value after this update
	Best      int  // best-ever streak after this update
	SameDay   bool // user already had a counted session today; no change
	Continued bool // streak grew (today == last + 1) or started from zero
	Broken    bool // streak reset to 1 because of a missed day
	NewRecord bool // best_streak just got bumped on this update
}

// StreakState is the persisted streak fields a service needs to make a
// decision. Kept separate from models.User so tests can construct it directly.
type StreakState struct {
	Current     int
	Best        int
	LastStudyAt *time.Time
}

// StreakService owns daily-streak business logic. It loads the user's current
// streak state, runs the pure decision function, and persists the result.
type StreakService struct {
	users    repositories.UserRepository
	location *time.Location
}

// NewStreakService wires a StreakService.
func NewStreakService(users repositories.UserRepository, loc *time.Location) *StreakService {
	return &StreakService{users: users, location: loc}
}

// Snapshot returns the user's current streak fields for display.
func (s *StreakService) Snapshot(ctx context.Context, userID int64) (current, best int, err error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("get user: %w", err)
	}
	return u.CurrentStreak, u.BestStreak, nil
}

// RecordCompletedSession evaluates one finished session against the streak
// rules and persists any change. Idempotent: calling it twice with the same
// (or any same-day) endedAt yields SameDay=true on the second call and does
// not modify state.
//
// All day comparisons happen in the configured location, so a session that
// crosses midnight is attributed to the day in which it *ended* — per spec.
func (s *StreakService) RecordCompletedSession(ctx context.Context, userID int64, sessionMinutes int, endedAt time.Time) (*StreakUpdate, error) {
	if sessionMinutes < MinStreakSessionMinutes {
		// Session too short to count. Return current state for display.
		cur, best, err := s.Snapshot(ctx, userID)
		if err != nil {
			return nil, err
		}
		return &StreakUpdate{Counted: false, Current: cur, Best: best}, nil
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	state := StreakState{
		Current:     u.CurrentStreak,
		Best:        u.BestStreak,
		LastStudyAt: u.LastStudyAt,
	}
	upd := computeStreakUpdate(state, endedAt, s.location)

	if upd.SameDay {
		// Nothing to persist — explicit no-op keeps the system idempotent
		// even if the upstream call retries.
		return &upd, nil
	}

	if err := s.users.UpdateStreak(ctx, userID, upd.Current, upd.Best, endedAt); err != nil {
		return nil, fmt.Errorf("persist streak: %w", err)
	}
	return &upd, nil
}

// computeStreakUpdate is the pure decision function. It has no I/O so every
// branch is unit-tested directly. Always returns Counted=true; callers must
// gate on the 30-minute minimum before invoking it.
func computeStreakUpdate(prev StreakState, endedAt time.Time, loc *time.Location) StreakUpdate {
	// First-ever qualifying session.
	if prev.LastStudyAt == nil {
		current := 1
		best := prev.Best
		newRecord := false
		if current > best {
			best = current
			newRecord = true
		}
		return StreakUpdate{
			Counted:   true,
			Current:   current,
			Best:      best,
			Continued: true,
			NewRecord: newRecord,
		}
	}

	diff := utils.LocalCalendarDayDiff(*prev.LastStudyAt, endedAt, loc)

	switch {
	case diff <= 0:
		// Same calendar day (or, defensively, an out-of-order/older end-time).
		// Keep state unchanged — this is the idempotency anchor.
		return StreakUpdate{
			Counted: true,
			Current: prev.Current,
			Best:    prev.Best,
			SameDay: true,
		}

	case diff == 1:
		// Consecutive day → streak grows.
		current := prev.Current + 1
		best := prev.Best
		newRecord := false
		if current > best {
			best = current
			newRecord = true
		}
		return StreakUpdate{
			Counted:   true,
			Current:   current,
			Best:      best,
			Continued: true,
			NewRecord: newRecord,
		}

	default:
		// Missed at least one day → restart at 1.
		// Note: starting a fresh streak does not bump best (1 cannot exceed
		// any previous best when one exists, and prev.LastStudyAt != nil
		// implies best >= 1).
		return StreakUpdate{
			Counted: true,
			Current: 1,
			Best:    prev.Best,
			Broken:  true,
		}
	}
}
