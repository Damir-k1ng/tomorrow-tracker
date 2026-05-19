package services

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// maxUserSessionsPage is a defensive upper bound on a single page of the user
// sessions listing. The request layer already bounds ?limit; this guards the
// service against any other caller.
const maxUserSessionsPage = 100

// UserProfile is the composed read-model behind GET /api/v1/user/me: the
// caller's identity and streak (both carried on models.User) plus their
// lifetime study totals, the active session (if one is running), and their
// today/week/remaining progress toward the weekly goal.
type UserProfile struct {
	User                *models.User
	TotalMinutes        int
	TotalSessions       int64
	ActiveSession       *models.Session
	Progress            Progress
	WeeklyTargetMinutes int
}

// StopSessionResult is the outcome of stopping a study session from the Mini
// App. It mirrors the bot's session-end orchestration: the finished session's
// length, the refreshed progress totals, and the streak evaluation.
//
// Streak is nil and StreakErr is set when the streak update failed — that is
// deliberately non-fatal (the session itself was saved), so the caller logs
// StreakErr but still reports success, exactly as the bot handler does.
type StopSessionResult struct {
	SessionMinutes      int
	Progress            Progress
	WeeklyTargetMinutes int
	Streak              *StreakUpdate
	StreakErr           error
}

// UserAPIService backs the user-facing API (/api/v1/user/*). It composes the
// session repository, the leaderboard service, and — for the Phase 3C session
// lifecycle — the session and streak services. Write paths (start/stop) reuse
// the exact same SessionService/StreakService logic the Telegram bot uses, so
// the two surfaces can never diverge.
type UserAPIService struct {
	sessions    repositories.SessionRepository
	leaderboard *LeaderboardService
	sessionSvc  *SessionService
	streaks     *StreakService
}

// NewUserAPIService wires a UserAPIService.
func NewUserAPIService(
	sessions repositories.SessionRepository,
	leaderboard *LeaderboardService,
	sessionSvc *SessionService,
	streaks *StreakService,
) *UserAPIService {
	return &UserAPIService{
		sessions:    sessions,
		leaderboard: leaderboard,
		sessionSvc:  sessionSvc,
		streaks:     streaks,
	}
}

// Profile composes the caller's overview. user is the already-authenticated
// caller resolved by the API middleware, so no extra identity lookup is needed.
func (s *UserAPIService) Profile(ctx context.Context, user *models.User) (*UserProfile, error) {
	// The four reads behind /user/me are independent — fan them out so the
	// request pays the latency of the slowest one, not the sum of all four.
	var (
		totalMinutes  int
		totalSessions int64
		active        *models.Session
		progress      Progress
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		totalMinutes, err = s.sessions.TotalCompletedMinutes(gctx, user.ID)
		if err != nil {
			return fmt.Errorf("total minutes: %w", err)
		}
		return nil
	})
	g.Go(func() (err error) {
		totalSessions, err = s.sessions.CompletedSessionCount(gctx, user.ID)
		if err != nil {
			return fmt.Errorf("total sessions: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		a, err := s.sessions.GetActive(gctx, user.ID)
		switch {
		case err == nil:
			active = a
		case errors.Is(err, repositories.ErrNotFound):
			// no active session — active stays nil
		default:
			return fmt.Errorf("active session: %w", err)
		}
		return nil
	})
	g.Go(func() (err error) {
		progress, err = s.sessionSvc.Progress(gctx, user.ID)
		if err != nil {
			return fmt.Errorf("progress: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &UserProfile{
		User:                user,
		TotalMinutes:        totalMinutes,
		TotalSessions:       totalSessions,
		ActiveSession:       active,
		Progress:            progress,
		WeeklyTargetMinutes: s.sessionSvc.WeeklyTargetMinutes(),
	}, nil
}

// StartSession opens a new study session for the caller. It returns
// ErrSessionAlreadyActive (unchanged from SessionService) when one is already
// running, so the handler can map it to a 409.
func (s *UserAPIService) StartSession(ctx context.Context, userID int64) (*models.Session, error) {
	return s.sessionSvc.Start(ctx, userID)
}

// FinishSession closes the session identified by sessionID on behalf of userID
// and evaluates their streak, mirroring the bot's EndSession orchestration.
// Ownership and state guards live in SessionService.FinishSession, so this
// surfaces ErrSessionNotFound / ErrSessionNotOwned / ErrSessionAlreadyFinished
// unchanged for the handler to map. A streak-update failure is captured in the
// result (StreakErr) rather than failing the call — the session was saved.
func (s *UserAPIService) FinishSession(ctx context.Context, userID, sessionID int64) (*StopSessionResult, error) {
	finish, err := s.sessionSvc.FinishSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	out := &StopSessionResult{
		SessionMinutes:      finish.SessionMinutes,
		Progress:            finish.Progress,
		WeeklyTargetMinutes: s.sessionSvc.WeeklyTargetMinutes(),
	}
	streakUpd, streakErr := s.streaks.RecordCompletedSession(ctx, userID, finish.SessionMinutes, finish.EndedAt)
	if streakErr != nil {
		out.StreakErr = streakErr
	} else {
		out.Streak = streakUpd
	}
	return out, nil
}

// Sessions returns one page of the caller's own sessions, newest first, plus
// the total session count for pagination metadata. limit/offset are clamped
// defensively even though the request layer already bounds them.
func (s *UserAPIService) Sessions(ctx context.Context, userID int64, limit, offset int) ([]models.Session, int64, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > maxUserSessionsPage {
		limit = maxUserSessionsPage
	}
	if offset < 0 {
		offset = 0
	}

	items, err := s.sessions.ListByUserPaged(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}
	total, err := s.sessions.CountByUser(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count sessions: %w", err)
	}
	return items, total, nil
}

// Leaderboard returns the current week's Top-N plus the caller's own standing.
func (s *UserAPIService) Leaderboard(ctx context.Context, userID int64) (*LeaderboardSnapshot, error) {
	return s.leaderboard.Snapshot(ctx, userID)
}
