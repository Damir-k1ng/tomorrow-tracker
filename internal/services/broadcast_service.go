package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

// MaxBroadcastChars caps a broadcast message at Telegram's text-message limit.
const MaxBroadcastChars = 4096

// broadcastBatchSize is how many recipients are fetched from the DB per page.
const broadcastBatchSize = 200

// progressTimeout bounds the detached DB writes that persist progress — they
// must complete even while the application context is shutting down, so they
// run on their own short-lived context rather than the worker's.
const progressTimeout = 5 * time.Second

// Broadcast outcomes surfaced to the API layer.
var (
	// ErrBroadcastInProgress — only one broadcast runs at a time.
	ErrBroadcastInProgress = errors.New("broadcast already in progress")
	// ErrEmptyBroadcast — the message is blank.
	ErrEmptyBroadcast = errors.New("broadcast message is empty")
	// ErrBroadcastTooLong — the message exceeds MaxBroadcastChars.
	ErrBroadcastTooLong = errors.New("broadcast message is too long")
)

// MessageSender delivers one plain-text message to a Telegram chat. It is the
// seam between the broadcast worker and the Telegram bot client.
type MessageSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

// BroadcastService starts and runs admin broadcasts. Exactly one broadcast
// runs at a time. The worker is resumable: it processes recipients in
// ascending user-id order and persists a cursor, so a restart continues
// exactly where it stopped — no user is messaged twice and none is skipped.
type BroadcastService struct {
	appCtx       context.Context
	repo         repositories.BroadcastRepository
	sender       MessageSender
	log          *slog.Logger
	sendInterval time.Duration // throttle between sends

	mu      sync.Mutex // guards running
	running bool
}

// NewBroadcastService wires a BroadcastService. appCtx is the application
// context: workers run under it, so shutdown stops them cleanly and leaves the
// row resumable. sendInterval throttles sends (e.g. time.Second/15 keeps the
// broadcast well under Telegram's global rate limit, leaving headroom for the
// live bot).
func NewBroadcastService(
	appCtx context.Context,
	repo repositories.BroadcastRepository,
	sender MessageSender,
	sendInterval time.Duration,
	log *slog.Logger,
) *BroadcastService {
	return &BroadcastService{
		appCtx:       appCtx,
		repo:         repo,
		sender:       sender,
		log:          log,
		sendInterval: sendInterval,
	}
}

// Start validates text, snapshots the recipient count, creates the broadcast
// row and launches the worker. It returns ErrBroadcastInProgress when one is
// already running, and ErrEmptyBroadcast / ErrBroadcastTooLong on bad input.
func (s *BroadcastService) Start(adminID int64, text string) (*models.Broadcast, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyBroadcast
	}
	if len([]rune(text)) > MaxBroadcastChars {
		return nil, ErrBroadcastTooLong
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil, ErrBroadcastInProgress
	}

	total, err := s.repo.CountUsers(s.appCtx)
	if err != nil {
		return nil, fmt.Errorf("count recipients: %w", err)
	}
	b, err := s.repo.Create(s.appCtx, text, adminID, total)
	if err != nil {
		return nil, fmt.Errorf("create broadcast: %w", err)
	}

	s.running = true
	go s.run(*b)
	return b, nil
}

// Resume relaunches a broadcast left 'running' by a previous process — call it
// once at startup. It is a no-op when there is none.
func (s *BroadcastService) Resume() {
	b, err := s.repo.FindRunning(s.appCtx)
	if errors.Is(err, repositories.ErrNotFound) {
		return
	}
	if err != nil {
		s.log.Error("broadcast: resume lookup failed", slog.String("error", err.Error()))
		return
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.log.Info("broadcast: resuming interrupted broadcast",
		slog.Int64("id", b.ID), slog.Int64("cursor", b.LastProcessedUserID))
	go s.run(*b)
}

// Get returns one broadcast by id.
func (s *BroadcastService) Get(ctx context.Context, id int64) (*models.Broadcast, error) {
	return s.repo.Get(ctx, id)
}

// List returns recent broadcasts, newest first.
func (s *BroadcastService) List(ctx context.Context, limit int) ([]models.Broadcast, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.List(ctx, limit)
}

// run is the worker: it sends to every recipient after the cursor, throttled,
// persisting progress as it goes. It always clears the running flag on exit.
// On shutdown it persists and returns with the row still 'running', so Resume
// picks it up next start.
func (s *BroadcastService) run(b models.Broadcast) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	ticker := time.NewTicker(s.sendInterval)
	defer ticker.Stop()

	cursor := b.LastProcessedUserID
	sent, failed := b.SentCount, b.FailedCount

	for {
		recipients, err := s.repo.RecipientsAfter(s.appCtx, cursor, broadcastBatchSize)
		if err != nil {
			s.log.Error("broadcast: fetch recipients failed",
				slog.Int64("id", b.ID), slog.String("error", err.Error()))
			return // leaves status 'running' → resumed next start
		}
		if len(recipients) == 0 {
			break // every recipient processed
		}

		for _, rec := range recipients {
			select {
			case <-s.appCtx.Done():
				s.persist(b.ID, cursor, sent, failed)
				s.log.Info("broadcast: paused for shutdown",
					slog.Int64("id", b.ID), slog.Int("sent", sent), slog.Int("failed", failed))
				return
			case <-ticker.C:
			}

			if err := s.sender.SendMessage(s.appCtx, rec.TelegramID, b.Message); err != nil {
				failed++
			} else {
				sent++
			}
			cursor = rec.UserID
		}
		s.persist(b.ID, cursor, sent, failed)
	}

	ctx, cancel := context.WithTimeout(context.Background(), progressTimeout)
	defer cancel()
	if err := s.repo.Finish(ctx, b.ID, models.BroadcastDone, sent, failed); err != nil {
		s.log.Error("broadcast: finish failed",
			slog.Int64("id", b.ID), slog.String("error", err.Error()))
		return
	}
	s.log.Info("broadcast: completed",
		slog.Int64("id", b.ID), slog.Int("sent", sent), slog.Int("failed", failed))
}

// persist saves the cursor and counters on a detached context, so progress is
// recorded even when the application is shutting down. A failure is logged,
// not fatal — the next resume simply re-reads the last persisted cursor.
func (s *BroadcastService) persist(id, cursor int64, sent, failed int) {
	ctx, cancel := context.WithTimeout(context.Background(), progressTimeout)
	defer cancel()
	if err := s.repo.SaveProgress(ctx, id, cursor, sent, failed); err != nil {
		s.log.Error("broadcast: save progress failed",
			slog.Int64("id", id), slog.String("error", err.Error()))
	}
}
