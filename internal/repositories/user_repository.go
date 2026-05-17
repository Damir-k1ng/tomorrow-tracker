// Package repositories isolates all SQL access. Services depend on the
// interfaces defined here, so the underlying database can change without
// touching business logic.
package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// ErrNotFound is returned when a lookup yields no rows.
var ErrNotFound = errors.New("not found")

// UserRepository describes the user persistence operations the services need.
type UserRepository interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	Upsert(ctx context.Context, telegramID int64, username, firstName string) (*models.User, error)
	// UpdateStreak persists new streak counters. The whole update is a single
	// statement, so it is atomic with respect to concurrent reads.
	UpdateStreak(ctx context.Context, userID int64, current, best int, lastStudyAt time.Time) error
	// UpdateRole persists a new role for the user.
	UpdateRole(ctx context.Context, userID int64, role string) error
}

type userRepo struct {
	db *pgxpool.Pool
}

// NewUserRepository wires a UserRepository backed by the given pool.
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

const userSelectColumns = `id, telegram_id, username, first_name, role, created_at,
        current_streak, best_streak, last_study_at`

func (r *userRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+userSelectColumns+` FROM users WHERE telegram_id = $1`, telegramID)
	return scanUser(row)
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+userSelectColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

// Upsert creates the user if missing or refreshes username/first_name if changed.
// Telegram users may rename themselves between visits, so we keep the local copy
// fresh. Streak fields are intentionally untouched here — they are managed by
// the streak service. RETURNING gives us the full row in one round trip.
func (r *userRepo) Upsert(ctx context.Context, telegramID int64, username, firstName string) (*models.User, error) {
	const q = `
        INSERT INTO users (telegram_id, username, first_name)
        VALUES ($1, $2, $3)
        ON CONFLICT (telegram_id) DO UPDATE SET
            username   = EXCLUDED.username,
            first_name = EXCLUDED.first_name
        RETURNING ` + userSelectColumns
	row := r.db.QueryRow(ctx, q, telegramID, username, firstName)
	u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

func (r *userRepo) UpdateStreak(ctx context.Context, userID int64, current, best int, lastStudyAt time.Time) error {
	const q = `
        UPDATE users
        SET current_streak = $1, best_streak = $2, last_study_at = $3
        WHERE id = $4`
	tag, err := r.db.Exec(ctx, q, current, best, lastStudyAt, userID)
	if err != nil {
		return fmt.Errorf("update streak: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepo) UpdateRole(ctx context.Context, userID int64, role string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET role = $1 WHERE id = $2`, role, userID)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanUser handles both pgx.Row and pgx.Rows via the small scannable interface
// (defined in session_repository.go). A NULL last_study_at scans straight into
// the *time.Time field as nil — pgx handles nullable timestamptz natively.
func scanUser(s scannable) (*models.User, error) {
	u := &models.User{}
	if err := s.Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.Role, &u.CreatedAt,
		&u.CurrentStreak, &u.BestStreak, &u.LastStudyAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}
