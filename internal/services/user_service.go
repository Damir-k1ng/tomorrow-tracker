// Package services contains the business logic that the bot handlers invoke.
// Services depend on repository interfaces, not concrete database drivers.
package services

import (
	"context"
	"fmt"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// UserService manages user registration / lookup and role assignment.
type UserService struct {
	repo            repositories.UserRepository
	adminTelegramID int64
}

// NewUserService wires a UserService. adminTelegramID identifies the single
// account that is auto-promoted to the admin role; pass 0 to disable
// auto-promotion entirely.
func NewUserService(repo repositories.UserRepository, adminTelegramID int64) *UserService {
	return &UserService{repo: repo, adminTelegramID: adminTelegramID}
}

// EnsureUser registers a new user or refreshes the username/first_name copy.
// It is safe to call on every interaction; users are identified by Telegram ID.
//
// If the user's Telegram ID matches the configured admin, they are promoted to
// the admin role. The promotion is idempotent — it only writes when the role
// is not already 'admin' — so calling this on every bot/API request is cheap.
func (s *UserService) EnsureUser(ctx context.Context, telegramID int64, username, firstName string) (*models.User, error) {
	u, err := s.repo.Upsert(ctx, telegramID, username, firstName)
	if err != nil {
		return nil, fmt.Errorf("ensure user: %w", err)
	}

	if s.adminTelegramID != 0 && u.TelegramID == s.adminTelegramID && u.Role != models.RoleAdmin {
		if err := s.repo.UpdateRole(ctx, u.ID, models.RoleAdmin); err != nil {
			return nil, fmt.Errorf("promote admin: %w", err)
		}
		u.Role = models.RoleAdmin
	}

	return u, nil
}
