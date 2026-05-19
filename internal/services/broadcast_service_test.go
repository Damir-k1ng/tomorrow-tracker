package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/damirkabdulla/tomorrow-tracker/internal/models"
	"github.com/damirkabdulla/tomorrow-tracker/internal/repositories"
)

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeBroadcastRepo is an in-memory repositories.BroadcastRepository. Users are
// ids 1..N (telegram id == user id for simplicity); the mutex makes it safe
// for the concurrent worker.
type fakeBroadcastRepo struct {
	mu         sync.Mutex
	users      []int64
	broadcasts map[int64]*models.Broadcast
	nextID     int64
}

func newFakeBroadcastRepo(userCount int) *fakeBroadcastRepo {
	users := make([]int64, userCount)
	for i := range users {
		users[i] = int64(i + 1)
	}
	return &fakeBroadcastRepo{users: users, broadcasts: map[int64]*models.Broadcast{}}
}

func (r *fakeBroadcastRepo) Create(_ context.Context, message string, createdBy int64, total int) (*models.Broadcast, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	b := &models.Broadcast{
		ID: r.nextID, Message: message, Status: models.BroadcastRunning,
		TotalRecipients: total, CreatedBy: createdBy, CreatedAt: time.Now(),
	}
	r.broadcasts[b.ID] = b
	cp := *b
	return &cp, nil
}

func (r *fakeBroadcastRepo) Get(_ context.Context, id int64) (*models.Broadcast, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.broadcasts[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	cp := *b
	return &cp, nil
}

func (r *fakeBroadcastRepo) List(_ context.Context, _ int) ([]models.Broadcast, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]models.Broadcast, 0, len(r.broadcasts))
	for _, b := range r.broadcasts {
		out = append(out, *b)
	}
	return out, nil
}

func (r *fakeBroadcastRepo) FindRunning(_ context.Context) (*models.Broadcast, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.broadcasts {
		if b.Status == models.BroadcastRunning {
			cp := *b
			return &cp, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (r *fakeBroadcastRepo) SaveProgress(_ context.Context, id, lastUserID int64, sent, failed int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.broadcasts[id]
	if !ok {
		return repositories.ErrNotFound
	}
	b.LastProcessedUserID, b.SentCount, b.FailedCount = lastUserID, sent, failed
	return nil
}

func (r *fakeBroadcastRepo) Finish(_ context.Context, id int64, status string, sent, failed int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.broadcasts[id]
	if !ok {
		return repositories.ErrNotFound
	}
	now := time.Now()
	b.Status, b.SentCount, b.FailedCount, b.FinishedAt = status, sent, failed, &now
	return nil
}

func (r *fakeBroadcastRepo) CountUsers(_ context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users), nil
}

func (r *fakeBroadcastRepo) RecipientsAfter(_ context.Context, after int64, limit int) ([]repositories.BroadcastRecipient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []repositories.BroadcastRecipient
	for _, id := range r.users {
		if id > after {
			out = append(out, repositories.BroadcastRecipient{UserID: id, TelegramID: id})
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

// fakeSender records delivered chat ids; failOn marks recipients that error;
// gate, when non-nil, blocks every send until the channel is closed.
type fakeSender struct {
	mu     sync.Mutex
	sent   []int64
	failOn map[int64]bool
	gate   chan struct{}
}

func (f *fakeSender) SendMessage(ctx context.Context, chatID int64, _ string) error {
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOn[chatID] {
		return errors.New("user blocked the bot")
	}
	f.sent = append(f.sent, chatID)
	return nil
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

// waitDone polls until the broadcast reaches 'done' or the deadline trips.
func waitDone(t *testing.T, svc *BroadcastService, id int64) *models.Broadcast {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		b, err := svc.Get(context.Background(), id)
		if err == nil && b.Status == models.BroadcastDone {
			return b
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("broadcast %d did not finish in time", id)
	return nil
}

func TestBroadcast_SendsToAllRecipients(t *testing.T) {
	repo := newFakeBroadcastRepo(10)
	sender := &fakeSender{failOn: map[int64]bool{}}
	svc := NewBroadcastService(context.Background(), repo, sender, time.Millisecond, discardLog())

	b, err := svc.Start(1, "hello everyone")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	done := waitDone(t, svc, b.ID)

	if done.SentCount != 10 || done.FailedCount != 0 {
		t.Fatalf("sent=%d failed=%d, want 10/0", done.SentCount, done.FailedCount)
	}
	if sender.count() != 10 {
		t.Fatalf("sender delivered %d, want 10", sender.count())
	}
}

func TestBroadcast_CountsFailedRecipients(t *testing.T) {
	repo := newFakeBroadcastRepo(10)
	sender := &fakeSender{failOn: map[int64]bool{3: true, 7: true}}
	svc := NewBroadcastService(context.Background(), repo, sender, time.Millisecond, discardLog())

	b, err := svc.Start(1, "hi")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	done := waitDone(t, svc, b.ID)

	if done.SentCount != 8 || done.FailedCount != 2 {
		t.Fatalf("sent=%d failed=%d, want 8/2 (a blocked user is failed, not fatal)",
			done.SentCount, done.FailedCount)
	}
}

func TestBroadcast_SingleFlight(t *testing.T) {
	repo := newFakeBroadcastRepo(5)
	gate := make(chan struct{})
	sender := &fakeSender{failOn: map[int64]bool{}, gate: gate}
	svc := NewBroadcastService(context.Background(), repo, sender, time.Millisecond, discardLog())

	b1, err := svc.Start(1, "first")
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}
	// The worker is parked in SendMessage on the gate, so the broadcast is
	// still running — a second Start must be rejected.
	if _, err := svc.Start(1, "second"); !errors.Is(err, ErrBroadcastInProgress) {
		t.Fatalf("second Start err = %v, want ErrBroadcastInProgress", err)
	}
	close(gate)
	waitDone(t, svc, b1.ID)
}

func TestBroadcast_ResumesFromCursor(t *testing.T) {
	repo := newFakeBroadcastRepo(10)
	// A broadcast left 'running' by a previous process: 6 of 10 already sent,
	// cursor at user 6.
	repo.broadcasts[1] = &models.Broadcast{
		ID: 1, Message: "resumed", Status: models.BroadcastRunning,
		TotalRecipients: 10, SentCount: 6, LastProcessedUserID: 6,
		CreatedBy: 1, CreatedAt: time.Now(),
	}
	repo.nextID = 1
	sender := &fakeSender{failOn: map[int64]bool{}}
	svc := NewBroadcastService(context.Background(), repo, sender, time.Millisecond, discardLog())

	svc.Resume()
	done := waitDone(t, svc, 1)

	if done.SentCount != 10 {
		t.Fatalf("sent=%d, want 10 (6 prior + 4 resumed)", done.SentCount)
	}
	if sender.count() != 4 {
		t.Fatalf("sender delivered %d, want 4 — only users 7..10, no re-sends", sender.count())
	}
	for _, id := range sender.sent {
		if id <= 6 {
			t.Fatalf("re-sent to user %d at or before the cursor", id)
		}
	}
}

func TestBroadcast_ValidatesInput(t *testing.T) {
	svc := NewBroadcastService(context.Background(), newFakeBroadcastRepo(1), &fakeSender{}, time.Millisecond, discardLog())

	if _, err := svc.Start(1, "   "); !errors.Is(err, ErrEmptyBroadcast) {
		t.Fatalf("blank text err = %v, want ErrEmptyBroadcast", err)
	}
	if _, err := svc.Start(1, strings.Repeat("x", MaxBroadcastChars+1)); !errors.Is(err, ErrBroadcastTooLong) {
		t.Fatalf("oversized text err = %v, want ErrBroadcastTooLong", err)
	}
}
