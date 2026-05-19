package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
)

// BroadcastRecipient is one target of a broadcast: the internal user id (which
// doubles as the resume cursor) paired with the Telegram chat id to send to.
type BroadcastRecipient struct {
	UserID     int64
	TelegramID int64
}

// BroadcastRepository is the data-access layer for admin broadcasts.
type BroadcastRepository interface {
	// Create inserts a new running broadcast and returns the full row.
	Create(ctx context.Context, message string, createdBy int64, total int) (*models.Broadcast, error)
	// Get returns one broadcast by id, or ErrNotFound.
	Get(ctx context.Context, id int64) (*models.Broadcast, error)
	// List returns recent broadcasts, newest first.
	List(ctx context.Context, limit int) ([]models.Broadcast, error)
	// FindRunning returns the single in-progress broadcast, or ErrNotFound.
	FindRunning(ctx context.Context) (*models.Broadcast, error)
	// SaveProgress persists the resume cursor and counters mid-run.
	SaveProgress(ctx context.Context, id, lastUserID int64, sent, failed int) error
	// Finish marks the broadcast terminal and stamps finished_at.
	Finish(ctx context.Context, id int64, status string, sent, failed int) error
	// CountUsers is the recipient-total snapshot taken when a broadcast starts.
	CountUsers(ctx context.Context) (int, error)
	// RecipientsAfter returns up to limit recipients with users.id > afterUserID,
	// ascending — the next page the worker sends.
	RecipientsAfter(ctx context.Context, afterUserID int64, limit int) ([]BroadcastRecipient, error)
}

type broadcastRepo struct {
	db *pgxpool.Pool
}

// NewBroadcastRepository wires a BroadcastRepository backed by the given pool.
func NewBroadcastRepository(db *pgxpool.Pool) BroadcastRepository {
	return &broadcastRepo{db: db}
}

const broadcastColumns = `id, message, status, total_recipients, sent_count,
        failed_count, last_processed_user_id, created_by, created_at, finished_at`

func (r *broadcastRepo) Create(ctx context.Context, message string, createdBy int64, total int) (*models.Broadcast, error) {
	const q = `
        INSERT INTO broadcasts (message, status, total_recipients, created_by)
        VALUES ($1, $2, $3, $4)
        RETURNING ` + broadcastColumns
	row := r.db.QueryRow(ctx, q, message, models.BroadcastRunning, total, createdBy)
	b, err := scanBroadcast(row)
	if err != nil {
		return nil, fmt.Errorf("create broadcast: %w", err)
	}
	return b, nil
}

func (r *broadcastRepo) Get(ctx context.Context, id int64) (*models.Broadcast, error) {
	row := r.db.QueryRow(ctx, `SELECT `+broadcastColumns+` FROM broadcasts WHERE id = $1`, id)
	return scanBroadcast(row)
}

func (r *broadcastRepo) List(ctx context.Context, limit int) ([]models.Broadcast, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+broadcastColumns+` FROM broadcasts ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list broadcasts: %w", err)
	}
	defer rows.Close()

	var out []models.Broadcast
	for rows.Next() {
		b, err := scanBroadcast(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate broadcasts: %w", err)
	}
	return out, nil
}

func (r *broadcastRepo) FindRunning(ctx context.Context) (*models.Broadcast, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+broadcastColumns+` FROM broadcasts WHERE status = $1 ORDER BY id LIMIT 1`,
		models.BroadcastRunning)
	return scanBroadcast(row)
}

func (r *broadcastRepo) SaveProgress(ctx context.Context, id, lastUserID int64, sent, failed int) error {
	const q = `
        UPDATE broadcasts
        SET last_processed_user_id = $1, sent_count = $2, failed_count = $3
        WHERE id = $4`
	tag, err := r.db.Exec(ctx, q, lastUserID, sent, failed, id)
	if err != nil {
		return fmt.Errorf("save broadcast progress: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *broadcastRepo) Finish(ctx context.Context, id int64, status string, sent, failed int) error {
	const q = `
        UPDATE broadcasts
        SET status = $1, sent_count = $2, failed_count = $3, finished_at = now()
        WHERE id = $4`
	tag, err := r.db.Exec(ctx, q, status, sent, failed, id)
	if err != nil {
		return fmt.Errorf("finish broadcast: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *broadcastRepo) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func (r *broadcastRepo) RecipientsAfter(ctx context.Context, afterUserID int64, limit int) ([]BroadcastRecipient, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, telegram_id FROM users WHERE id > $1 ORDER BY id ASC LIMIT $2`,
		afterUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recipients: %w", err)
	}
	defer rows.Close()

	var out []BroadcastRecipient
	for rows.Next() {
		var rec BroadcastRecipient
		if err := rows.Scan(&rec.UserID, &rec.TelegramID); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipients: %w", err)
	}
	return out, nil
}

// scanBroadcast reads one broadcasts row. A NULL finished_at scans into the
// *time.Time field as nil.
func scanBroadcast(s scannable) (*models.Broadcast, error) {
	b := &models.Broadcast{}
	if err := s.Scan(
		&b.ID, &b.Message, &b.Status, &b.TotalRecipients, &b.SentCount,
		&b.FailedCount, &b.LastProcessedUserID, &b.CreatedBy, &b.CreatedAt, &b.FinishedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan broadcast: %w", err)
	}
	return b, nil
}
