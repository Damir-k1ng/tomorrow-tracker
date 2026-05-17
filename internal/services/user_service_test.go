package services

import (
	"context"
	"errors"
	"testing"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// errAlwaysFail is injected into the fake repo to prove a code path is NOT
// taken: if EnsureUser wrongly called UpdateRole, this error would surface.
var errAlwaysFail = errors.New("update must not be called")

func TestEnsureUser_PromotesConfiguredAdmin(t *testing.T) {
	const adminTG = int64(165146312)
	repo := &fakeUserRepo{}
	svc := NewUserService(repo, adminTG)

	u, err := svc.EnsureUser(context.Background(), adminTG, "king", "Damir")
	if err != nil {
		t.Fatalf("ensure user: %v", err)
	}
	if u.Role != models.RoleAdmin {
		t.Fatalf("expected admin role, got %q", u.Role)
	}
	if repo.user.Role != models.RoleAdmin {
		t.Fatalf("admin role not persisted, got %q", repo.user.Role)
	}
}

func TestEnsureUser_NonAdminStaysUser(t *testing.T) {
	const adminTG = int64(165146312)
	repo := &fakeUserRepo{}
	svc := NewUserService(repo, adminTG)

	u, err := svc.EnsureUser(context.Background(), 999, "someone", "Someone")
	if err != nil {
		t.Fatalf("ensure user: %v", err)
	}
	if u.Role != models.RoleUser {
		t.Fatalf("expected user role, got %q", u.Role)
	}
}

func TestEnsureUser_AdminPromotionIsIdempotent(t *testing.T) {
	const adminTG = int64(165146312)
	repo := &fakeUserRepo{
		user: &models.User{ID: 1, TelegramID: adminTG, Role: models.RoleAdmin},
	}
	svc := NewUserService(repo, adminTG)

	// Already admin — EnsureUser must not attempt another role write.
	repo.updateErr = errAlwaysFail
	u, err := svc.EnsureUser(context.Background(), adminTG, "king", "Damir")
	if err != nil {
		t.Fatalf("ensure user should not call UpdateRole when already admin: %v", err)
	}
	if u.Role != models.RoleAdmin {
		t.Fatalf("expected admin role, got %q", u.Role)
	}
}
