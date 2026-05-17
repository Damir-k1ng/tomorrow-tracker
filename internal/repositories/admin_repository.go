package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/utils"
)

// ErrSessionActive is returned when an admin tries to correct a session that
// is still running. Only finished sessions may be edited.
var ErrSessionActive = errors.New("session is active")

// querier is satisfied by both *pgxpool.Pool and pgx.Tx, so the audit-insert
// helper can run either standalone or inside a transaction.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// UserListParams drives the paginated admin users listing. SortColumn must
// already be a whitelisted, safe column name — the repository interpolates it.
type UserListParams struct {
	Search     string
	SortColumn string
	SortDesc   bool
	Limit      int
	Offset     int
}

// AuditListParams drives the paginated audit-log listing.
type AuditListParams struct {
	Action   *string // optional exact-match filter
	AdminID  *int64  // optional exact-match filter
	SortDesc bool
	Limit    int
	Offset   int
}

// SessionListParams drives the paginated admin sessions listing. Every filter
// is optional; Flagged narrows to sessions carrying anti-cheat evidence (a
// non-empty anti_cheat_flags array) — the moderation queue.
type SessionListParams struct {
	UserID   *int64 // optional: only this owner's sessions
	Valid    *bool  // optional: filter by is_valid
	Flagged  bool   // when true: only sessions with anti_cheat_flags != '[]'
	SortDesc bool
	Limit    int
	Offset   int
}

// SessionPatch carries the fields an admin may change on a finished session.
// DurationMinutes and IsValid are nil when absent from the request, so an
// admin can change one field without restating the others. AntiCheatFlags is
// applied only when SetFlags is true, and it must already be a canonical,
// whitelisted JSON array (validated upstream by the request layer). Reason is
// always mandatory.
type SessionPatch struct {
	DurationMinutes *int
	IsValid         *bool
	AntiCheatFlags  []byte
	SetFlags        bool
	Reason          string
}

// AuditEntry is one immutable admin_actions record to append. BeforeData /
// AfterData hold raw JSON (or nil).
type AuditEntry struct {
	AdminID      int64
	Action       string
	EntityType   *string
	EntityID     *int64
	TargetUserID *int64
	BeforeData   []byte
	AfterData    []byte
	Reason       *string
}

// AdminRepository is the data access layer for the Admin Panel API.
type AdminRepository interface {
	Stats(ctx context.Context) (*models.AdminStats, error)
	ListUsers(ctx context.Context, p UserListParams) (users []models.User, total int64, err error)
	ListAuditLogs(ctx context.Context, p AuditListParams) (logs []models.AuditLog, total int64, err error)
	// ListSessions returns a page of sessions (with owner identity) plus the
	// total row count, honouring the optional user / validity / flagged filters.
	ListSessions(ctx context.Context, p SessionListParams) (rows []models.AdminSessionRow, total int64, err error)
	// CorrectSessionWithAudit applies an admin correction to a finished
	// session, writes the immutable audit record, and — when the correction
	// changes the session's streak qualification — deterministically rebuilds
	// the owner's streak. All of this happens in ONE transaction: all-or-nothing.
	CorrectSessionWithAudit(ctx context.Context, adminID, sessionID int64, patch SessionPatch) (*models.SessionCorrection, error)
	// InsertAuditLog appends a standalone audit record (used by exports).
	InsertAuditLog(ctx context.Context, e AuditEntry) error
	// StreamUsers / StreamSessions iterate export rows one at a time so a CSV
	// can be streamed without buffering the whole result set in memory.
	StreamUsers(ctx context.Context, from, to time.Time, fn func(models.User) error) error
	StreamSessions(ctx context.Context, from, to time.Time, fn func(models.Session) error) error
}

type adminRepo struct {
	db  *pgxpool.Pool
	loc *time.Location // app timezone — streak day bucketing must be timezone-correct
}

// NewAdminRepository wires an AdminRepository backed by the given pool. loc is
// the application timezone; streak recomputation buckets sessions into local
// calendar days, so it must match the timezone used by the streak service.
func NewAdminRepository(db *pgxpool.Pool, loc *time.Location) AdminRepository {
	return &adminRepo{db: db, loc: loc}
}

// Stats returns operational counters in a single round trip.
func (r *adminRepo) Stats(ctx context.Context) (*models.AdminStats, error) {
	const q = `
        SELECT
            (SELECT count(*) FROM users),
            (SELECT count(*) FROM sessions WHERE is_active),
            (SELECT COALESCE(SUM(duration_minutes), 0) FROM sessions WHERE NOT is_active),
            (SELECT count(*) FROM sessions WHERE NOT is_active),
            (SELECT COALESCE(MAX(best_streak), 0) FROM users),
            (SELECT COALESCE(AVG(current_streak), 0)::float8 FROM users),
            (SELECT count(*) FROM users WHERE created_at >= now() - interval '7 days')`

	var (
		s          models.AdminStats
		sumMinutes int64
		avgStreak  float64
	)
	err := r.db.QueryRow(ctx, q).Scan(
		&s.TotalUsers, &s.ActiveSessions, &sumMinutes, &s.CompletedSessions,
		&s.BestStreak, &avgStreak, &s.NewUsers7d,
	)
	if err != nil {
		return nil, fmt.Errorf("admin stats: %w", err)
	}
	s.TotalStudyHours = round1(float64(sumMinutes) / 60.0)
	s.AverageStreak = round1(avgStreak)
	return &s, nil
}

// round1 rounds to one decimal place.
func round1(x float64) float64 { return math.Round(x*10) / 10 }

func (r *adminRepo) ListUsers(ctx context.Context, p UserListParams) ([]models.User, int64, error) {
	q := fmt.Sprintf(`
        SELECT id, telegram_id, username, first_name, role, created_at,
               current_streak, best_streak, last_study_at,
               COUNT(*) OVER() AS total
        FROM users
        WHERE ($1 = '' OR username ILIKE $2 OR first_name ILIKE $2)
        ORDER BY %s, id DESC
        LIMIT $3 OFFSET $4`, orderClause(p.SortColumn, p.SortDesc))

	pattern := "%" + p.Search + "%"
	rows, err := r.db.Query(ctx, q, p.Search, pattern, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var (
		out   []models.User
		total int64
	)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.Role, &u.CreatedAt,
			&u.CurrentStreak, &u.BestStreak, &u.LastStudyAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}
	return out, total, nil
}

func (r *adminRepo) ListAuditLogs(ctx context.Context, p AuditListParams) ([]models.AuditLog, int64, error) {
	dir := "DESC"
	if !p.SortDesc {
		dir = "ASC"
	}
	q := fmt.Sprintf(`
        SELECT id, admin_id, action, entity_type, entity_id, target_user_id,
               before_data::text, after_data::text, reason, created_at,
               COUNT(*) OVER() AS total
        FROM admin_actions
        WHERE ($1::text IS NULL OR action = $1)
          AND ($2::bigint IS NULL OR admin_id = $2)
        ORDER BY created_at %s, id %s
        LIMIT $3 OFFSET $4`, dir, dir)

	rows, err := r.db.Query(ctx, q, p.Action, p.AdminID, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var (
		out   []models.AuditLog
		total int64
	)
	for rows.Next() {
		var (
			l          models.AuditLog
			beforeJSON *string
			afterJSON  *string
		)
		if err := rows.Scan(
			&l.ID, &l.AdminID, &l.Action, &l.EntityType, &l.EntityID, &l.TargetUserID,
			&beforeJSON, &afterJSON, &l.Reason, &l.CreatedAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		if beforeJSON != nil {
			l.BeforeData = json.RawMessage(*beforeJSON)
		}
		if afterJSON != nil {
			l.AfterData = json.RawMessage(*afterJSON)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}
	return out, total, nil
}

func (r *adminRepo) ListSessions(ctx context.Context, p SessionListParams) ([]models.AdminSessionRow, int64, error) {
	dir := "DESC"
	if !p.SortDesc {
		dir = "ASC"
	}
	// Filters are all parameter-driven (no interpolation). $2 = Flagged: when
	// false the term collapses to TRUE; when true it requires a non-empty
	// anti_cheat_flags array. COUNT(*) OVER() yields the unpaginated total.
	q := fmt.Sprintf(`
        SELECT s.id, s.user_id, s.started_at, s.ended_at, s.duration_minutes,
               s.is_active, s.created_at, s.is_valid, s.anti_cheat_flags::text,
               u.first_name, u.username,
               COUNT(*) OVER() AS total
        FROM sessions s
        JOIN users u ON u.id = s.user_id
        WHERE ($1::bigint IS NULL OR s.user_id = $1)
          AND ($2::bool IS NULL OR s.is_valid = $2)
          AND (NOT $3::bool OR s.anti_cheat_flags <> '[]'::jsonb)
        ORDER BY s.started_at %s, s.id %s
        LIMIT $4 OFFSET $5`, dir, dir)

	rows, err := r.db.Query(ctx, q, p.UserID, p.Valid, p.Flagged, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var (
		out   []models.AdminSessionRow
		total int64
	)
	for rows.Next() {
		var (
			row     models.AdminSessionRow
			endedAt *time.Time
			flags   string
		)
		if err := rows.Scan(
			&row.Session.ID, &row.Session.UserID, &row.Session.StartedAt, &endedAt,
			&row.Session.DurationMinutes, &row.Session.IsActive, &row.Session.CreatedAt,
			&row.Session.IsValid, &flags, &row.OwnerFirstName, &row.OwnerUsername, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan session row: %w", err)
		}
		if endedAt != nil {
			row.Session.EndedAt = *endedAt
		}
		row.Session.AntiCheatFlags = json.RawMessage(flags)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows: %w", err)
	}
	return out, total, nil
}

func (r *adminRepo) CorrectSessionWithAudit(ctx context.Context, adminID, sessionID int64, patch SessionPatch) (*models.SessionCorrection, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	// Rollback is a no-op once Commit has succeeded; this guarantees cleanup
	// on every early-return error path.
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the session row so a concurrent finish/edit cannot race the correction.
	var (
		ownerID     int64
		oldDuration int
		isActive    bool
		oldIsValid  bool
		oldFlags    string
	)
	err = tx.QueryRow(ctx,
		`SELECT user_id, duration_minutes, is_active, is_valid, anti_cheat_flags::text
		   FROM sessions WHERE id = $1 FOR UPDATE`,
		sessionID).Scan(&ownerID, &oldDuration, &isActive, &oldIsValid, &oldFlags)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock session: %w", err)
	}
	if isActive {
		return nil, ErrSessionActive
	}

	// Resolve new field values — an absent field keeps its old value.
	newDuration := oldDuration
	if patch.DurationMinutes != nil {
		newDuration = *patch.DurationMinutes
	}
	newIsValid := oldIsValid
	if patch.IsValid != nil {
		newIsValid = *patch.IsValid
	}
	newFlags := oldFlags
	if patch.SetFlags {
		newFlags = string(patch.AntiCheatFlags)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE sessions
		    SET duration_minutes = $1, is_valid = $2, anti_cheat_flags = $3::jsonb
		  WHERE id = $4`,
		newDuration, newIsValid, newFlags, sessionID); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// The audit insert runs on the SAME tx — if anything below fails, the
	// session correction is rolled back. All-or-nothing.
	before, _ := json.Marshal(sessionAuditSnapshot(oldDuration, oldIsValid, oldFlags))
	after, _ := json.Marshal(sessionAuditSnapshot(newDuration, newIsValid, newFlags))
	entityType := "session"
	if err := insertAuditLog(ctx, tx, AuditEntry{
		AdminID:      adminID,
		Action:       "PATCH_SESSION",
		EntityType:   &entityType,
		EntityID:     &sessionID,
		TargetUserID: &ownerID,
		BeforeData:   before,
		AfterData:    after,
		Reason:       &patch.Reason,
	}); err != nil {
		return nil, fmt.Errorf("insert audit: %w", err)
	}

	corr := &models.SessionCorrection{
		SessionID:   sessionID,
		OwnerID:     ownerID,
		OldDuration: oldDuration,
		NewDuration: newDuration,
		OldIsValid:  oldIsValid,
		NewIsValid:  newIsValid,
	}

	// A session counts toward the streak only while (is_valid AND duration >=
	// the minimum). When that boolean flips, the owner's materialized streak
	// must be rebuilt — in THIS transaction, so the session edit, the audit
	// record, and the streak move atomically together.
	oldQualifies := oldIsValid && oldDuration >= MinStreakSessionMinutes
	newQualifies := newIsValid && newDuration >= MinStreakSessionMinutes
	if oldQualifies != newQualifies {
		cur, best, err := r.recomputeStreakTx(ctx, tx, ownerID)
		if err != nil {
			return nil, err
		}
		corr.StreakRecomputed = true
		corr.CurrentStreak = cur
		corr.BestStreak = best
	} else if err := tx.QueryRow(ctx,
		`SELECT current_streak, best_streak FROM users WHERE id = $1`,
		ownerID).Scan(&corr.CurrentStreak, &corr.BestStreak); err != nil {
		return nil, fmt.Errorf("read streak: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return corr, nil
}

// recomputeStreakTx deterministically rebuilds one user's materialized streak
// from their qualifying sessions and persists it — all inside the given tx.
// The user row is locked FOR UPDATE first, so a concurrent streak write (e.g.
// the bot finishing a session) cannot interleave with the recomputation.
//
// best_streak is recomputed from scratch, not max'd with the stored value: the
// sessions table is the single source of truth, so invalidating a session can
// legitimately lower best_streak. That is what makes the result deterministic.
func (r *adminRepo) recomputeStreakTx(ctx context.Context, tx pgx.Tx, userID int64) (current, best int, err error) {
	var locked int64
	if err = tx.QueryRow(ctx,
		`SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&locked); err != nil {
		return 0, 0, fmt.Errorf("lock user: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT ended_at FROM sessions
		  WHERE user_id = $1 AND NOT is_active AND is_valid
		    AND duration_minutes >= $2 AND ended_at IS NOT NULL`,
		userID, MinStreakSessionMinutes)
	if err != nil {
		return 0, 0, fmt.Errorf("load qualifying sessions: %w", err)
	}
	var endedAts []time.Time
	for rows.Next() {
		var t time.Time
		if err = rows.Scan(&t); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan ended_at: %w", err)
		}
		endedAts = append(endedAts, t)
	}
	rows.Close() // close before issuing the UPDATE on the same tx
	if err = rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("rows: %w", err)
	}

	cur, bst, last, hasAny := utils.StreakFromSessions(endedAts, r.loc)
	if hasAny {
		if _, err = tx.Exec(ctx,
			`UPDATE users SET current_streak = $1, best_streak = $2, last_study_at = $3
			  WHERE id = $4`, cur, bst, last, userID); err != nil {
			return 0, 0, fmt.Errorf("persist streak: %w", err)
		}
	} else if _, err = tx.Exec(ctx,
		`UPDATE users SET current_streak = 0, best_streak = 0, last_study_at = NULL
		  WHERE id = $1`, userID); err != nil {
		return 0, 0, fmt.Errorf("clear streak: %w", err)
	}
	return cur, bst, nil
}

// sessionAuditSnapshot is the structured before/after payload stored in a
// session-correction audit record. anti_cheat_flags is embedded as raw JSON so
// the array is preserved verbatim.
func sessionAuditSnapshot(duration int, isValid bool, flags string) map[string]any {
	return map[string]any{
		"duration_minutes": duration,
		"is_valid":         isValid,
		"anti_cheat_flags": json.RawMessage(flags),
	}
}

func (r *adminRepo) InsertAuditLog(ctx context.Context, e AuditEntry) error {
	if err := insertAuditLog(ctx, r.db, e); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *adminRepo) StreamUsers(ctx context.Context, from, to time.Time, fn func(models.User) error) error {
	const q = `
        SELECT id, telegram_id, username, first_name, role, created_at,
               current_streak, best_streak, last_study_at
        FROM users
        WHERE created_at >= $1 AND created_at < $2
        ORDER BY id ASC`
	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return fmt.Errorf("stream users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.Role, &u.CreatedAt,
			&u.CurrentStreak, &u.BestStreak, &u.LastStudyAt,
		); err != nil {
			return fmt.Errorf("scan user: %w", err)
		}
		if err := fn(u); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (r *adminRepo) StreamSessions(ctx context.Context, from, to time.Time, fn func(models.Session) error) error {
	const q = `
        SELECT ` + sessionColumns + `
        FROM sessions
        WHERE started_at >= $1 AND started_at < $2
        ORDER BY id ASC`
	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return fmt.Errorf("stream sessions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return fmt.Errorf("scan session: %w", err)
		}
		if err := fn(*s); err != nil {
			return err
		}
	}
	return rows.Err()
}

// insertAuditLog appends one admin_actions row using the given querier, so it
// works both standalone (pool) and inside a transaction (tx).
func insertAuditLog(ctx context.Context, q querier, e AuditEntry) error {
	const sql = `
        INSERT INTO admin_actions
            (admin_id, action, entity_type, entity_id, target_user_id,
             before_data, after_data, reason)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := q.Exec(ctx, sql,
		e.AdminID, e.Action, e.EntityType, e.EntityID, e.TargetUserID,
		nullableJSON(e.BeforeData), nullableJSON(e.AfterData), e.Reason)
	return err
}

// nullableJSON passes a JSON byte slice to a jsonb column, or SQL NULL when
// empty — pgx needs an explicit nil to write NULL.
func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// orderClause builds a safe "ORDER BY <col> <dir>" fragment. col is a
// pre-whitelisted literal, so no injection is possible.
func orderClause(col string, desc bool) string {
	if desc {
		return col + " DESC"
	}
	return col + " ASC"
}
