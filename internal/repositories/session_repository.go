package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// SessionRepository describes session persistence operations.
type SessionRepository interface {
	GetActive(ctx context.Context, userID int64) (*models.Session, error)
	Create(ctx context.Context, userID int64, startedAt time.Time) (*models.Session, error)
	Finish(ctx context.Context, sessionID int64, endedAt time.Time, durationMinutes int) error
	// ListOverlapping returns every session whose [started_at, ended_at|now)
	// interval overlaps [from, to). Active sessions have zero EndedAt.
	ListOverlapping(ctx context.Context, userID int64, from, to time.Time) ([]models.Session, error)
	// WeeklyTotals returns one row per user with their capped, clipped study
	// minutes inside [from, to). Each session contributes at most 12 hours.
	// Active sessions count up to `to` (the caller passes "now").
	// Rows with zero minutes are filtered out. Ordering matches leaderboard
	// rules: minutes DESC, then user.created_at ASC.
	WeeklyTotals(ctx context.Context, from, to time.Time) ([]models.WeeklyTotal, error)
}

// MaxSessionMinutes is the per-session anti-cheat cap. Anything longer is
// counted as exactly 12 hours toward leaderboards.
const MaxSessionMinutes = 12 * 60

// isoTS serializes a time.Time as RFC3339Nano UTC. The modernc.org/sqlite
// driver's default time.Time binding produces Go's String() format
// ("2026-05-11 02:00:00 +0500 +05"), which SQLite's date functions
// (julianday, strftime, etc.) cannot parse. Binding as an ISO-8601 string
// instead keeps stored values portable and unlocks SQL date math.
// This is also the format PostgreSQL accepts when we eventually migrate.
func isoTS(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

type sessionRepo struct {
	db *sql.DB
}

// NewSessionRepository wires a SessionRepository backed by the given DB.
func NewSessionRepository(db *sql.DB) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) GetActive(ctx context.Context, userID int64) (*models.Session, error) {
	const q = `
        SELECT id, user_id, started_at, ended_at, duration_minutes, is_active, created_at
        FROM sessions
        WHERE user_id = ? AND is_active = 1
        ORDER BY started_at DESC
        LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, userID)

	s, err := scanSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan active session: %w", err)
	}
	return s, nil
}

func (r *sessionRepo) Create(ctx context.Context, userID int64, startedAt time.Time) (*models.Session, error) {
	const insert = `
        INSERT INTO sessions (user_id, started_at, is_active)
        VALUES (?, ?, 1)`
	res, err := r.db.ExecContext(ctx, insert, userID, isoTS(startedAt))
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("session id: %w", err)
	}
	return &models.Session{
		ID:        id,
		UserID:    userID,
		StartedAt: startedAt,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (r *sessionRepo) Finish(ctx context.Context, sessionID int64, endedAt time.Time, durationMinutes int) error {
	const update = `
        UPDATE sessions
        SET ended_at = ?, duration_minutes = ?, is_active = 0
        WHERE id = ? AND is_active = 1`
	res, err := r.db.ExecContext(ctx, update, isoTS(endedAt), durationMinutes, sessionID)
	if err != nil {
		return fmt.Errorf("finish session: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		// Session was already closed or never existed — services can decide
		// how to surface this to the user.
		return ErrNotFound
	}
	return nil
}

func (r *sessionRepo) ListOverlapping(ctx context.Context, userID int64, from, to time.Time) ([]models.Session, error) {
	const q = `
        SELECT id, user_id, started_at, ended_at, duration_minutes, is_active, created_at
        FROM sessions
        WHERE user_id = ?
          AND started_at < ?
          AND (ended_at IS NULL OR ended_at > ?)
        ORDER BY started_at ASC`
	rows, err := r.db.QueryContext(ctx, q, userID, isoTS(to), isoTS(from))
	if err != nil {
		return nil, fmt.Errorf("list overlapping: %w", err)
	}
	defer rows.Close()

	var out []models.Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		out = append(out, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// WeeklyTotals computes per-user weekly minutes in a single aggregation.
//
// Per session, the contribution is:
//   minutes = clamp(0, 720, (min(to, ended_at|to) - max(from, started_at)) * 1440)
//
// julianday() returns the date as a fractional day number; multiplying by 1440
// converts to minutes. The MAX/MIN scalar forms clip the session window into
// the requested [from, to) range, MAX(0, ...) discards negative durations
// (invalid timestamps), and MIN(720, ...) enforces the 12h-per-session cap.
//
// Ordering: minutes DESC, then created_at ASC, then user id ASC — fully
// deterministic so users do not see their rank flicker when totals are tied.
//
// Sessions are filtered to only those overlapping [from, to) so the index on
// started_at is leveraged.
func (r *sessionRepo) WeeklyTotals(ctx context.Context, from, to time.Time) ([]models.WeeklyTotal, error) {
	const q = `
        SELECT
            u.id,
            u.first_name,
            u.username,
            u.created_at,
            COALESCE(SUM(
                MIN(720, MAX(0, CAST(
                    (julianday(MIN(?, COALESCE(s.ended_at, ?)))
                     - julianday(MAX(?, s.started_at))) * 1440
                AS INTEGER)))
            ), 0) AS minutes
        FROM users u
        JOIN sessions s ON s.user_id = u.id
        WHERE s.started_at < ?
          AND (s.ended_at IS NULL OR s.ended_at > ?)
        GROUP BY u.id
        HAVING minutes > 0
        ORDER BY minutes DESC, u.created_at ASC, u.id ASC`

	fromS := isoTS(from)
	toS := isoTS(to)

	rows, err := r.db.QueryContext(ctx, q,
		toS, toS, // MIN(to, COALESCE(ended_at, to))
		fromS,       // MAX(from, started_at)
		toS, fromS,  // WHERE bounds
	)
	if err != nil {
		return nil, fmt.Errorf("weekly totals: %w", err)
	}
	defer rows.Close()

	var out []models.WeeklyTotal
	for rows.Next() {
		var t models.WeeklyTotal
		if err := rows.Scan(&t.UserID, &t.FirstName, &t.Username, &t.CreatedAt, &t.Minutes); err != nil {
			return nil, fmt.Errorf("scan weekly total: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// scannable abstracts over *sql.Row and *sql.Rows so scanSession works for both.
type scannable interface {
	Scan(dest ...any) error
}

func scanSession(s scannable) (*models.Session, error) {
	out := &models.Session{}
	var endedAt sql.NullTime
	var isActiveInt int
	if err := s.Scan(&out.ID, &out.UserID, &out.StartedAt, &endedAt, &out.DurationMinutes, &isActiveInt, &out.CreatedAt); err != nil {
		return nil, err
	}
	if endedAt.Valid {
		out.EndedAt = endedAt.Time
	}
	out.IsActive = isActiveInt != 0
	return out, nil
}
