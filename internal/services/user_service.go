// Package services contains the business logic that the bot handlers invoke.
// Services depend on repository interfaces, not concrete database drivers.
package services

import (
	"context"
	"fmt"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// UserService manages user registration / lookup.
type UserService struct {
	repo repositories.UserRepository
}

// NewUserService wires a UserService with the given repository.
func NewUserService(repo repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// EnsureUser registers a new user or refreshes the username/first_name copy.
// It is safe to call on every interaction; users are identified by Telegram ID.
func (s *UserService) EnsureUser(ctx context.Context, telegramID int64, username, firstName string) (*models.User, error) {
	u, err := s.repo.Upsert(ctx, telegramID, username, firstName)
	if err != nil {
		return nil, fmt.Errorf("ensure user: %w", err)
	}
	return u, nil
}
