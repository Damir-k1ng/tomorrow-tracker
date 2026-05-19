package bot

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const testWebhookSecret = "test-secret-123"

// newWebhookTestBot builds a Bot with only the fields WebhookHandler needs: a
// discard logger, a worker-pool slot channel, and a counting pipeline that
// records how many updates actually reached processing.
func newWebhookTestBot(processed *atomic.Int32) *Bot {
	return &Bot{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		sem: make(chan struct{}, maxConcurrentUpdates),
		pipeline: func(_ context.Context, _ tgbotapi.Update) error {
			processed.Add(1)
			return nil
		},
	}
}

func TestWebhookHandler_ValidUpdate(t *testing.T) {
	var processed atomic.Int32
	b := newWebhookTestBot(&processed)
	h := b.WebhookHandler(context.Background(), testWebhookSecret)

	body := `{"update_id":1,"message":{"message_id":1,"text":"/start"}}`
	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(body))
	req.Header.Set(telegramSecretHeader, testWebhookSecret)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	// dispatch is asynchronous — wait for the worker goroutine to finish.
	b.wg.Wait()
	if got := processed.Load(); got != 1 {
		t.Fatalf("pipeline ran %d times, want 1", got)
	}
}

func TestWebhookHandler_WrongSecretRejected(t *testing.T) {
	var processed atomic.Int32
	b := newWebhookTestBot(&processed)
	h := b.WebhookHandler(context.Background(), testWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(`{"update_id":1}`))
	req.Header.Set(telegramSecretHeader, "wrong-secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	b.wg.Wait()
	if got := processed.Load(); got != 0 {
		t.Fatalf("pipeline ran %d times on a wrong-secret request, want 0", got)
	}
}

func TestWebhookHandler_MissingSecretRejected(t *testing.T) {
	var processed atomic.Int32
	b := newWebhookTestBot(&processed)
	h := b.WebhookHandler(context.Background(), testWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(`{"update_id":1}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a missing secret header", rec.Code)
	}
}

func TestWebhookHandler_RejectsGET(t *testing.T) {
	var processed atomic.Int32
	b := newWebhookTestBot(&processed)
	h := b.WebhookHandler(context.Background(), testWebhookSecret)

	req := httptest.NewRequest(http.MethodGet, "/telegram/webhook", nil)
	req.Header.Set(telegramSecretHeader, testWebhookSecret)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestWebhookHandler_BadJSONRejected(t *testing.T) {
	var processed atomic.Int32
	b := newWebhookTestBot(&processed)
	h := b.WebhookHandler(context.Background(), testWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(`{not json`))
	req.Header.Set(telegramSecretHeader, testWebhookSecret)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	b.wg.Wait()
	if got := processed.Load(); got != 0 {
		t.Fatalf("pipeline ran %d times on malformed JSON, want 0", got)
	}
}
