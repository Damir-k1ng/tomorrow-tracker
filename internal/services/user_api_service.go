package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// maxUserSessionsPage is a defensive upper bound on a single page of the user
// sessions listing. The request layer already bounds ?limit; this guards the
// service against any other caller.
const maxUserSessionsPage = 100

// UserProfile is the composed read-model behind GET /api/v1/user/me: the
// caller's identity and streak (both carried on models.User) plus their
// lifetime study totals and the active session, if one is running.
type UserProfile struct {
	User          *models.User
	TotalMinutes  int
	TotalSessions int64
	ActiveSession *models.Session
}

// UserAPIService backs the read-only user-facing API (/api/v1/user/*). It
// composes the session repository and the leaderboard service. It holds no
// write paths — the Mini App user surface is strictly read-only in this phase.
type UserAPIService struct {
	sessions    repositories.SessionRepository
	leaderboard *LeaderboardService
}

// NewUserAPIService wires a UserAPIService.
func NewUserAPIService(sessions repositories.SessionRepository, leaderboard *LeaderboardService) *UserAPIService {
	return &UserAPIService{sessions: sessions, leaderboard: leaderboard}
}

// Profile composes the caller's overview. user is the already-authenticated
// caller resolved by the API middleware, so no extra identity lookup is needed.
func (s *UserAPIService) Profile(ctx context.Context, user *models.User) (*UserProfile, error) {
	totalMinutes, err := s.sessions.TotalCompletedMinutes(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("total minutes: %w", err)
	}
	totalSessions, err := s.sessions.CompletedSessionCount(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("total sessions: %w", err)
	}

	var active *models.Session
	if a, err := s.sessions.GetActive(ctx, user.ID); err == nil {
		active = a
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("active session: %w", err)
	}

	return &UserProfile{
		User:          user,
		TotalMinutes:  totalMinutes,
		TotalSessions: totalSessions,
		ActiveSession: active,
	}, nil
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
