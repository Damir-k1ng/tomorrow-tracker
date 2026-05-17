package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
	"github.com/damirkabdulla/tomorrow-tracker/internal/utils"
)

// ErrSessionAlreadyActive is returned when a user tries to start a session
// while another one is still running.
var ErrSessionAlreadyActive = errors.New("session already active")

// ErrNoActiveSession is returned when a user tries to end a session that does
// not exist.
var ErrNoActiveSession = errors.New("no active session")

// Session-lifecycle errors for the id-addressed finish flow (Mini App).
var (
	// ErrSessionNotFound is returned when the session id does not exist.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionNotOwned is returned when the session exists but belongs to a
	// different user — a user may only finish their own sessions.
	ErrSessionNotOwned = errors.New("session belongs to another user")
	// ErrSessionAlreadyFinished is returned when the target session is no
	// longer active — finished sessions are immutable from user flows.
	ErrSessionAlreadyFinished = errors.New("session already finished")
)

// Progress is a snapshot of a user's study totals at a given moment.
type Progress struct {
	TodayMinutes     int
	WeekMinutes      int
	RemainingMinutes int // until WeeklyTargetHours; never negative
	HasActiveSession bool
	ActiveStartedAt  time.Time // zero if no active session
}

// SessionService implements the rules around starting/finishing sessions
// and computing weekly progress.
type SessionService struct {
	repo              repositories.SessionRepository
	location          *time.Location
	weeklyTargetHours int
}

// NewSessionService wires a SessionService.
func NewSessionService(repo repositories.SessionRepository, loc *time.Location, weeklyTargetHours int) *SessionService {
	return &SessionService{
		repo:              repo,
		location:          loc,
		weeklyTargetHours: weeklyTargetHours,
	}
}

// Location exposes the configured timezone so handlers can format times consistently.
func (s *SessionService) Location() *time.Location { return s.location }

// WeeklyTargetMinutes is the weekly study goal expressed in minutes. The Mini
// App uses it to render progress toward the goal without hardcoding 30h.
func (s *SessionService) WeeklyTargetMinutes() int { return s.weeklyTargetHours * 60 }

// Start opens a new session if none is active. The returned session uses the
// service's local timezone for its StartedAt for display convenience.
func (s *SessionService) Start(ctx context.Context, userID int64) (*models.Session, error) {
	if existing, err := s.repo.GetActive(ctx, userID); err == nil && existing != nil {
		return nil, ErrSessionAlreadyActive
	} else if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("check active: %w", err)
	}

	now := time.Now().In(s.location)
	session, err := s.repo.Create(ctx, userID, now)
	if err != nil {
		// The partial unique index rejected a concurrent duplicate start —
		// the pre-check above lost a race. Surface the same friendly error.
		if errors.Is(err, repositories.ErrActiveSessionExists) {
			return nil, ErrSessionAlreadyActive
		}
		return nil, fmt.Errorf("create session: %w", err)
	}
	session.StartedAt = session.StartedAt.In(s.location)
	return session, nil
}

// FinishSession closes one specific session, addressed by its id, on behalf of
// userID. Unlike Finish (which the bot uses to close "the" active session),
// this enforces the full Mini App contract:
//
//   - the session must exist            → ErrSessionNotFound
//   - the session must belong to userID → ErrSessionNotOwned
//   - the session must still be active  → ErrSessionAlreadyFinished
//
// The duration is computed server-side from started_at to now; the client
// timer is never trusted. The final UPDATE re-checks ownership and is_active
// atomically, so a concurrent finish loses cleanly with ErrSessionAlreadyFinished.
func (s *SessionService) FinishSession(ctx context.Context, userID, sessionID int64) (*FinishResult, error) {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	if sess.UserID != userID {
		return nil, ErrSessionNotOwned
	}
	if !sess.IsActive {
		return nil, ErrSessionAlreadyFinished
	}

	now := time.Now().In(s.location)
	startedLocal := sess.StartedAt.In(s.location)
	duration := utils.MinutesBetween(startedLocal, now)

	if err := s.repo.FinishOwned(ctx, sessionID, userID, now, duration); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			// Lost a race: the session was finished between our check and the
			// UPDATE. Idempotent outcome — report it as already finished.
			return nil, ErrSessionAlreadyFinished
		}
		return nil, fmt.Errorf("finish session: %w", err)
	}

	progress, err := s.Progress(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("compute progress: %w", err)
	}

	return &FinishResult{
		SessionMinutes: duration,
		EndedAt:        now,
		Progress:       progress,
	}, nil
}

// FinishResult is the data needed to render the "session ended" message.
// EndedAt is exposed so the streak service can attribute the session to the
// correct calendar day (in the configured timezone).
type FinishResult struct {
	SessionMinutes int
	EndedAt        time.Time
	Progress       Progress
}

// Finish closes the user's active session, returning the session length
// alongside today/week/remaining totals.
func (s *SessionService) Finish(ctx context.Context, userID int64) (*FinishResult, error) {
	active, err := s.repo.GetActive(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNoActiveSession
		}
		return nil, fmt.Errorf("get active: %w", err)
	}

	now := time.Now().In(s.location)
	startedLocal := active.StartedAt.In(s.location)
	duration := utils.MinutesBetween(startedLocal, now)

	if err := s.repo.Finish(ctx, active.ID, now, duration); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNoActiveSession
		}
		return nil, fmt.Errorf("finish: %w", err)
	}

	progress, err := s.Progress(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("compute progress: %w", err)
	}

	return &FinishResult{
		SessionMinutes: duration,
		EndedAt:        now,
		Progress:       progress,
	}, nil
}

// Progress computes today's, this week's, and remaining minutes for the user.
// All boundaries are evaluated in the configured timezone, so the week resets
// at local Monday 00:00 regardless of server location.
func (s *SessionService) Progress(ctx context.Context, userID int64) (Progress, error) {
	now := time.Now().In(s.location)

	dayStart := utils.StartOfDay(now)
	dayEnd := dayStart.AddDate(0, 0, 1)

	weekStart := utils.StartOfWeek(now)
	weekEnd := weekStart.AddDate(0, 0, 7)

	today, err := s.sumMinutes(ctx, userID, dayStart, dayEnd, now)
	if err != nil {
		return Progress{}, err
	}
	week, err := s.sumMinutes(ctx, userID, weekStart, weekEnd, now)
	if err != nil {
		return Progress{}, err
	}

	target := s.weeklyTargetHours * 60
	remaining := target - week
	if remaining < 0 {
		remaining = 0
	}

	out := Progress{
		TodayMinutes:     today,
		WeekMinutes:      week,
		RemainingMinutes: remaining,
	}

	active, err := s.repo.GetActive(ctx, userID)
	switch {
	case err == nil:
		out.HasActiveSession = true
		out.ActiveStartedAt = active.StartedAt.In(s.location)
	case errors.Is(err, repositories.ErrNotFound):
		// no-op: HasActiveSession stays false
	default:
		return Progress{}, fmt.Errorf("get active: %w", err)
	}

	return out, nil
}

// sumMinutes asks the repo for sessions overlapping [from, to) and computes
// the total elapsed minutes that fall inside that window. Active sessions
// contribute up to `now`.
func (s *SessionService) sumMinutes(ctx context.Context, userID int64, from, to, now time.Time) (int, error) {
	sessions, err := s.repo.ListOverlapping(ctx, userID, from, to)
	if err != nil {
		return 0, fmt.Errorf("list overlapping: %w", err)
	}

	total := 0
	for _, sess := range sessions {
		start := sess.StartedAt
		var end time.Time
		if sess.IsActive || sess.EndedAt.IsZero() {
			end = now
		} else {
			end = sess.EndedAt
		}

		// Clip to [from, to)
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		total += utils.MinutesBetween(start, end)
	}
	return total, nil
}
