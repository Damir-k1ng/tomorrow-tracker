package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// ErrActiveSessionExists is returned by Create when the database rejects a new
// session because the user already has an active one. It is the data-layer
// counterpart of the "one active session per user" partial unique index — the
// race-safe backstop for the service-level pre-check.
var ErrActiveSessionExists = errors.New("active session already exists")

// uniqueViolationCode is the PostgreSQL SQLSTATE for a unique-constraint
// violation (used to recognise a duplicate active-session insert).
const uniqueViolationCode = "23505"

// SessionRepository describes session persistence operations.
type SessionRepository interface {
	GetActive(ctx context.Context, userID int64) (*models.Session, error)
	Create(ctx context.Context, userID int64, startedAt time.Time) (*models.Session, error)
	Finish(ctx context.Context, sessionID int64, endedAt time.Time, durationMinutes int) error
	// FinishOwned closes a session only when it belongs to userID and is still
	// active. The ownership + is_active guards live in the SQL WHERE clause, so
	// the UPDATE is atomic: it can never finish another user's session and can
	// never double-finish one. ErrNotFound means no row matched (wrong owner,
	// unknown id, or already finished).
	FinishOwned(ctx context.Context, sessionID, userID int64, endedAt time.Time, durationMinutes int) error
	// ListOverlapping returns every session whose [started_at, ended_at|now)
	// interval overlaps [from, to). Active sessions have zero EndedAt.
	ListOverlapping(ctx context.Context, userID int64, from, to time.Time) ([]models.Session, error)
	// WeeklyTotals returns one row per user with their capped, clipped study
	// minutes inside [from, to). Each session contributes at most 12 hours.
	// Active sessions count up to `to` (the caller passes "now").
	// Rows with zero minutes are filtered out. Ordering matches leaderboard
	// rules: minutes DESC, then created_at ASC, then user id ASC.
	WeeklyTotals(ctx context.Context, from, to time.Time) ([]models.WeeklyTotal, error)
	// GetByID returns a single session by primary key.
	GetByID(ctx context.Context, sessionID int64) (*models.Session, error)
	// ListByUser returns a user's most recent sessions, newest first, bounded
	// by limit — never the full history.
	ListByUser(ctx context.Context, userID int64, limit int) ([]models.Session, error)
	// TotalCompletedMinutes sums duration_minutes across a user's finished
	// sessions.
	TotalCompletedMinutes(ctx context.Context, userID int64) (int, error)
	// ListByUserPaged returns a page of a user's sessions, newest first,
	// bounded by limit and offset — never the full history.
	ListByUserPaged(ctx context.Context, userID int64, limit, offset int) ([]models.Session, error)
	// CountByUser returns the total number of sessions a user has — used for
	// the pagination metadata of the user sessions listing.
	CountByUser(ctx context.Context, userID int64) (int64, error)
	// CompletedSessionCount returns the number of a user's finished
	// (non-active) sessions.
	CompletedSessionCount(ctx context.Context, userID int64) (int64, error)
}

// MaxSessionMinutes is the per-session anti-cheat cap. Anything longer is
// counted as exactly 12 hours toward leaderboards.
const MaxSessionMinutes = 12 * 60

// MinStreakSessionMinutes is the minimum length a finished session must reach
// to count toward a user's daily streak. It lives here, in the data layer,
// because both the streak service and the admin streak-recomputation query
// need a single shared source of truth.
const MinStreakSessionMinutes = 30

// sessionColumns is the canonical column list for a full session row. Keeping
// it in one place ensures every SELECT and scanSession stay in lockstep.
// anti_cheat_flags is cast to text so it scans through the shared helper.
const sessionColumns = `id, user_id, started_at, ended_at, duration_minutes,
        is_active, created_at, is_valid, anti_cheat_flags::text`

type sessionRepo struct {
	db *pgxpool.Pool
}

// NewSessionRepository wires a SessionRepository backed by the given pool.
func NewSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) GetActive(ctx context.Context, userID int64) (*models.Session, error) {
	const q = `
        SELECT ` + sessionColumns + `
        FROM sessions
        WHERE user_id = $1 AND is_active = TRUE
        ORDER BY started_at DESC
        LIMIT 1`
	s, err := scanSession(r.db.QueryRow(ctx, q, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan active session: %w", err)
	}
	return s, nil
}

func (r *sessionRepo) Create(ctx context.Context, userID int64, startedAt time.Time) (*models.Session, error) {
	const q = `
        INSERT INTO sessions (user_id, started_at, is_active)
        VALUES ($1, $2, TRUE)
        RETURNING id, created_at`
	s := &models.Session{
		UserID:    userID,
		StartedAt: startedAt,
		IsActive:  true,
	}
	if err := r.db.QueryRow(ctx, q, userID, startedAt).Scan(&s.ID, &s.CreatedAt); err != nil {
		// A unique violation here is the partial index uq_sessions_one_active
		// rejecting a second active session — surface it as a typed error so
		// the service maps it to a friendly "already active" response.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, ErrActiveSessionExists
		}
		return nil, fmt.Errorf("insert session: %w", err)
	}
	return s, nil
}

func (r *sessionRepo) FinishOwned(ctx context.Context, sessionID, userID int64, endedAt time.Time, durationMinutes int) error {
	const q = `
        UPDATE sessions
        SET ended_at = $1, duration_minutes = $2, is_active = FALSE
        WHERE id = $3 AND user_id = $4 AND is_active = TRUE`
	tag, err := r.db.Exec(ctx, q, endedAt, durationMinutes, sessionID, userID)
	if err != nil {
		return fmt.Errorf("finish owned session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// No row matched: unknown id, not owned by userID, or already closed.
		// The service decides how to surface each case to the user.
		return ErrNotFound
	}
	return nil
}

func (r *sessionRepo) Finish(ctx context.Context, sessionID int64, endedAt time.Time, durationMinutes int) error {
	const q = `
        UPDATE sessions
        SET ended_at = $1, duration_minutes = $2, is_active = FALSE
        WHERE id = $3 AND is_active = TRUE`
	tag, err := r.db.Exec(ctx, q, endedAt, durationMinutes, sessionID)
	if err != nil {
		return fmt.Errorf("finish session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Session was already closed or never existed — services decide how
		// to surface this to the user.
		return ErrNotFound
	}
	return nil
}

func (r *sessionRepo) ListOverlapping(ctx context.Context, userID int64, from, to time.Time) ([]models.Session, error) {
	const q = `
        SELECT ` + sessionColumns + `
        FROM sessions
        WHERE user_id = $1
          AND started_at < $2
          AND (ended_at IS NULL OR ended_at > $3)
        ORDER BY started_at ASC`
	rows, err := r.db.Query(ctx, q, userID, to, from)
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
//
//	minutes = clamp(0, 720, floor( seconds( min(to, ended_at|to)
//	                                       - max(from, started_at) ) / 60 ))
//
// EXTRACT(EPOCH FROM interval) yields the interval length in seconds; dividing
// by 60 gives minutes. GREATEST/LEAST clip the session window into [from, to),
// GREATEST(0, ...) discards negative durations (invalid timestamps), and
// LEAST(720, ...) enforces the 12h-per-session anti-cheat cap.
//
// Ordering: minutes DESC, then created_at ASC, then user id ASC — fully
// deterministic so users do not see their rank flicker when totals are tied.
//
// $1 = to (window end / "now"), $2 = from (window start). GROUP BY users.id is
// valid because id is the primary key, so the other u.* columns are
// functionally dependent and may be selected without aggregation.
func (r *sessionRepo) WeeklyTotals(ctx context.Context, from, to time.Time) ([]models.WeeklyTotal, error) {
	const q = `
        WITH totals AS (
            SELECT
                u.id         AS user_id,
                u.first_name AS first_name,
                u.username   AS username,
                u.created_at AS created_at,
                COALESCE(SUM(
                    LEAST(720, GREATEST(0,
                        FLOOR(
                            EXTRACT(EPOCH FROM (
                                LEAST($1::timestamptz, COALESCE(s.ended_at, $1::timestamptz))
                                - GREATEST($2::timestamptz, s.started_at)
                            )) / 60
                        )::bigint
                    ))
                ), 0)::int AS minutes
            FROM users u
            JOIN sessions s ON s.user_id = u.id
            WHERE s.started_at < $1::timestamptz
              AND (s.ended_at IS NULL OR s.ended_at > $2::timestamptz)
            GROUP BY u.id
        )
        SELECT user_id, first_name, username, created_at, minutes
        FROM totals
        WHERE minutes > 0
        ORDER BY minutes DESC, created_at ASC, user_id ASC`

	rows, err := r.db.Query(ctx, q, to, from)
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

func (r *sessionRepo) GetByID(ctx context.Context, sessionID int64) (*models.Session, error) {
	const q = `
        SELECT ` + sessionColumns + `
        FROM sessions
        WHERE id = $1`
	s, err := scanSession(r.db.QueryRow(ctx, q, sessionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}
	return s, nil
}

func (r *sessionRepo) ListByUser(ctx context.Context, userID int64, limit int) ([]models.Session, error) {
	const q = `
        SELECT ` + sessionColumns + `
        FROM sessions
        WHERE user_id = $1
        ORDER BY started_at DESC, id DESC
        LIMIT $2`
	rows, err := r.db.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list by user: %w", err)
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

func (r *sessionRepo) TotalCompletedMinutes(ctx context.Context, userID int64) (int, error) {
	const q = `
        SELECT COALESCE(SUM(duration_minutes), 0)
        FROM sessions
        WHERE user_id = $1 AND is_active = FALSE`
	var total int
	if err := r.db.QueryRow(ctx, q, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("total completed minutes: %w", err)
	}
	return total, nil
}

func (r *sessionRepo) ListByUserPaged(ctx context.Context, userID int64, limit, offset int) ([]models.Session, error) {
	const q = `
        SELECT ` + sessionColumns + `
        FROM sessions
        WHERE user_id = $1
        ORDER BY started_at DESC, id DESC
        LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list by user paged: %w", err)
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

func (r *sessionRepo) CountByUser(ctx context.Context, userID int64) (int64, error) {
	const q = `SELECT COUNT(*) FROM sessions WHERE user_id = $1`
	var total int64
	if err := r.db.QueryRow(ctx, q, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("count by user: %w", err)
	}
	return total, nil
}

func (r *sessionRepo) CompletedSessionCount(ctx context.Context, userID int64) (int64, error) {
	const q = `SELECT COUNT(*) FROM sessions WHERE user_id = $1 AND is_active = FALSE`
	var total int64
	if err := r.db.QueryRow(ctx, q, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("completed session count: %w", err)
	}
	return total, nil
}

// scannable abstracts over pgx.Row and pgx.Rows so scan helpers work for both.
type scannable interface {
	Scan(dest ...any) error
}

func scanSession(s scannable) (*models.Session, error) {
	out := &models.Session{}
	var endedAt *time.Time
	var flags string // anti_cheat_flags::text — NOT NULL in the schema
	if err := s.Scan(&out.ID, &out.UserID, &out.StartedAt, &endedAt,
		&out.DurationMinutes, &out.IsActive, &out.CreatedAt,
		&out.IsValid, &flags); err != nil {
		return nil, err
	}
	if endedAt != nil {
		out.EndedAt = *endedAt
	}
	out.AntiCheatFlags = json.RawMessage(flags)
	return out, nil
}
