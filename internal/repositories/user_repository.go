// Package repositories isolates all SQL access. Services depend on the
// interfaces defined here, so swapping SQLite for PostgreSQL only requires
// providing an alternate implementation.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// ErrNotFound is returned when a lookup yields no rows.
var ErrNotFound = errors.New("not found")

// UserRepository describes the user persistence operations the services need.
type UserRepository interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	Upsert(ctx context.Context, telegramID int64, username, firstName string) (*models.User, error)
	// UpdateStreak persists new streak counters. lastStudyAt is stored as
	// RFC3339 text so it round-trips with SQLite's date functions and is
	// portable to PostgreSQL. The whole update is one statement, so it's
	// atomic with respect to concurrent reads.
	UpdateStreak(ctx context.Context, userID int64, current, best int, lastStudyAt time.Time) error
}

type userRepo struct {
	db *sql.DB
}

// NewUserRepository wires a UserRepository backed by the given DB.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepo{db: db}
}

const userSelectColumns = `id, telegram_id, username, first_name, created_at,
        current_streak, best_streak, last_study_at`

func (r *userRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+userSelectColumns+` FROM users WHERE telegram_id = ?`, telegramID)
	return scanUser(row)
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+userSelectColumns+` FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// Upsert creates the user if missing or refreshes username/first_name if changed.
// Telegram users may rename themselves between visits, so we keep the local copy
// fresh. Streak fields are intentionally untouched here — they are managed by
// the streak service.
func (r *userRepo) Upsert(ctx context.Context, telegramID int64, username, firstName string) (*models.User, error) {
	const insert = `
        INSERT INTO users (telegram_id, username, first_name)
        VALUES (?, ?, ?)
        ON CONFLICT(telegram_id) DO UPDATE SET
            username   = excluded.username,
            first_name = excluded.first_name`
	if _, err := r.db.ExecContext(ctx, insert, telegramID, username, firstName); err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return r.GetByTelegramID(ctx, telegramID)
}

func (r *userRepo) UpdateStreak(ctx context.Context, userID int64, current, best int, lastStudyAt time.Time) error {
	const q = `
        UPDATE users
        SET current_streak = ?, best_streak = ?, last_study_at = ?
        WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, current, best, lastStudyAt.UTC().Format(time.RFC3339Nano), userID)
	if err != nil {
		return fmt.Errorf("update streak: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

// scanUser handles both *sql.Row and *sql.Rows via the small scannable
// interface (defined in session_repository.go). last_study_at is stored as
// RFC3339 text and parsed back to *time.Time; NULL stays nil.
func scanUser(s scannable) (*models.User, error) {
	u := &models.User{}
	var lastStudy sql.NullString
	if err := s.Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.CreatedAt,
		&u.CurrentStreak, &u.BestStreak, &lastStudy,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	if lastStudy.Valid && lastStudy.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastStudy.String); err == nil {
			u.LastStudyAt = &t
		} else if t, err := time.Parse(time.RFC3339, lastStudy.String); err == nil {
			u.LastStudyAt = &t
		}
	}
	return u, nil
}
