package bot

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// telegramSecretHeader is the header Telegram sends on every webhook request,
// echoing the secret_token passed to setWebhook. Verifying it is what makes a
// public webhook endpoint trustworthy — the secret never travels in the URL,
// so it never lands in an access log.
const telegramSecretHeader = "X-Telegram-Bot-Api-Secret-Token" //nolint:gosec // header name, not a credential

// maxWebhookBodyBytes caps a webhook request body before it is decoded. A
// Telegram update is small; this rejects oversized junk cheaply.
const maxWebhookBodyBytes = 1 << 20 // 1 MiB

// WebhookHandler returns the http.Handler that receives Telegram updates in
// webhook mode. It verifies the secret-token header, decodes the update, hands
// it to the bounded worker pool, and ALWAYS responds 200 immediately: the
// update is processed asynchronously, so Telegram's delivery is never coupled
// to handler latency (a slow send must not make Telegram retry the update).
//
// ctx is the application context — the background handler runs under it, so an
// in-flight update observes shutdown.
func (b *Bot) WebhookHandler(ctx context.Context, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Constant-time comparison so a mismatch leaks no timing signal.
		got := r.Header.Get(telegramSecretHeader)
		if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
			b.log.Warn("webhook: secret token mismatch", slog.String("remote", r.RemoteAddr))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var update tgbotapi.Update
		if err := json.NewDecoder(io.LimitReader(r.Body, maxWebhookBodyBytes)).Decode(&update); err != nil {
			b.log.Warn("webhook: bad update payload", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Acknowledge first, process after — Telegram only needs the 200.
		w.WriteHeader(http.StatusOK)
		b.dispatch(ctx, update)
	})
}

// RegisterWebhook tells Telegram to deliver updates to webhookURL via POST,
// authenticated by secret (echoed back in the secret-token header).
//
// It is a raw Bot API call for the same reason setMenuButton is: the pinned
// telegram-bot-api v5.5.1 predates the secret_token parameter (Bot API 6.1),
// so its typed WebhookConfig cannot express it.
func (b *Bot) RegisterWebhook(webhookURL, secret string) error {
	payload, err := json.Marshal(map[string]any{
		"url":             webhookURL,
		"secret_token":    secret,
		"allowed_updates": []string{"message"},
		"max_connections": 40,
	})
	if err != nil {
		return fmt.Errorf("marshal setWebhook: %w", err)
	}

	endpoint := "https://api.telegram.org/bot" + b.api.Token + "/setWebhook"
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("setWebhook request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Trust the Bot API envelope, not the HTTP status: Telegram can return a
	// 200 with {"ok":false}.
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("setWebhook decode response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("setWebhook rejected by telegram: %s", result.Description)
	}
	return nil
}

// RunWebhook is the webhook-mode counterpart of Run. It publishes the bot
// commands and menu button, registers the webhook with Telegram, then blocks
// until ctx is cancelled. Updates arrive via the handler from WebhookHandler,
// which the HTTP server serves — there is no polling loop here.
//
// It must be called AFTER the HTTP server is listening, so Telegram's first
// POST has somewhere to land.
func (b *Bot) RunWebhook(ctx context.Context, webhookURL, secret string) error {
	b.registerCommands()
	b.setMenuButton()

	if err := b.RegisterWebhook(webhookURL, secret); err != nil {
		return err
	}
	b.log.Info("webhook registered, awaiting updates",
		slog.String("url", webhookURL),
		slog.Int("max_concurrent", maxConcurrentUpdates))

	<-ctx.Done()
	b.log.Info("shutdown signal received, stopping webhook mode")
	b.wg.Wait()
	return nil
}
