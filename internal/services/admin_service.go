package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// recentSessionsLimit bounds the session history returned in user details —
// never the full, unbounded history.
const recentSessionsLimit = 20

// AdminService orchestrates the Admin Panel API: it composes repositories,
// applies business rules, and keeps the HTTP handlers thin. A session
// correction may change the session's streak qualification; when it does, the
// repository deterministically recomputes the affected user's streak inside
// the same transaction. The leaderboard stays query-based and is never
// recomputed here.
type AdminService struct {
	admin    repositories.AdminRepository
	sessions repositories.SessionRepository
	users    repositories.UserRepository
}

// NewAdminService wires an AdminService.
func NewAdminService(
	admin repositories.AdminRepository,
	sessions repositories.SessionRepository,
	users repositories.UserRepository,
) *AdminService {
	return &AdminService{admin: admin, sessions: sessions, users: users}
}

// Stats returns the operational dashboard counters.
func (s *AdminService) Stats(ctx context.Context) (*models.AdminStats, error) {
	return s.admin.Stats(ctx)
}

// ListUsers returns a page of users plus the total row count.
func (s *AdminService) ListUsers(ctx context.Context, p repositories.UserListParams) ([]models.User, int64, error) {
	return s.admin.ListUsers(ctx, p)
}

// ListAuditLogs returns a page of audit records plus the total row count.
func (s *AdminService) ListAuditLogs(ctx context.Context, p repositories.AuditListParams) ([]models.AuditLog, int64, error) {
	return s.admin.ListAuditLogs(ctx, p)
}

// UserDetails composes the per-user admin view: profile, streak (carried on
// the user record), total study minutes, the current active session if any,
// and the last 20 sessions.
func (s *AdminService) UserDetails(ctx context.Context, userID int64) (*models.UserDetails, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err // repositories.ErrNotFound bubbles up unchanged
	}

	total, err := s.sessions.TotalCompletedMinutes(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("total minutes: %w", err)
	}

	recent, err := s.sessions.ListByUser(ctx, userID, recentSessionsLimit)
	if err != nil {
		return nil, fmt.Errorf("recent sessions: %w", err)
	}

	var active *models.Session
	if a, err := s.sessions.GetActive(ctx, userID); err == nil {
		active = a
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("active session: %w", err)
	}

	return &models.UserDetails{
		User:           user,
		TotalMinutes:   total,
		ActiveSession:  active,
		RecentSessions: recent,
	}, nil
}

// CorrectSession applies an admin session correction. Field validation (ranges,
// the anti-cheat flag whitelist, the mandatory reason) is performed upstream by
// the request layer; the repository enforces the "finished sessions only" rule
// and writes the correction, the immutable audit record, and any streak
// recomputation atomically in a single transaction.
func (s *AdminService) CorrectSession(ctx context.Context, adminID, sessionID int64, patch repositories.SessionPatch) (*models.SessionCorrection, error) {
	return s.admin.CorrectSessionWithAudit(ctx, adminID, sessionID, patch)
}

// RecordExport appends the mandatory audit record for an export operation.
// Called before streaming, so an export is always logged.
func (s *AdminService) RecordExport(ctx context.Context, adminID int64, action string, from, to time.Time) error {
	after := []byte(fmt.Sprintf(`{"from":%q,"to":%q}`,
		from.Format(time.RFC3339), to.Format(time.RFC3339)))
	return s.admin.InsertAuditLog(ctx, repositories.AuditEntry{
		AdminID:   adminID,
		Action:    action,
		AfterData: after,
	})
}

// StreamUsers invokes fn for each user created in [from, to).
func (s *AdminService) StreamUsers(ctx context.Context, from, to time.Time, fn func(models.User) error) error {
	return s.admin.StreamUsers(ctx, from, to, fn)
}

// StreamSessions invokes fn for each session started in [from, to).
func (s *AdminService) StreamSessions(ctx context.Context, from, to time.Time, fn func(models.Session) error) error {
	return s.admin.StreamSessions(ctx, from, to, fn)
}
