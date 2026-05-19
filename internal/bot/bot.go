package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/middleware"
)

// Connect authenticates with Telegram and returns the underlying API client.
// It is split from Bot construction so callers can build handlers around the
// API before wiring the final Bot.
func Connect(token string, log *slog.Logger) (*tgbotapi.BotAPI, error) {
	// Route the telegram-bot-api library's own logging through slog before any
	// API call, so transient deploy-overlap "Conflict" lines arrive as
	// structured warnings instead of raw red stderr.
	installBotLogger(log)

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("connect telegram: %w", err)
	}

	// Bound every Bot API call. The library's default client has NO timeout,
	// so a stalled connection to api.telegram.org can block the
	// single-threaded update loop indefinitely — production logs show sends
	// hanging 22s+ and TLS handshakes timing out. 50s sits safely above the
	// 30s getUpdates long-poll while turning an infinite hang into a fast,
	// retryable error.
	api.Client = &http.Client{Timeout: 50 * time.Second}

	log.Info("telegram authorized", slog.String("username", api.Self.UserName))
	return api, nil
}

// Bot owns the long-running update loop.
type Bot struct {
	api        *tgbotapi.BotAPI
	router     *Router
	log        *slog.Logger
	miniAppURL string // "" → no Mini App menu button
}

// New wires a Bot from an authenticated API client and a router. miniAppURL,
// when non-empty, is published as the chat menu button so users can launch
// the Mini App straight from the bot.
func New(api *tgbotapi.BotAPI, router *Router, log *slog.Logger, miniAppURL string) *Bot {
	return &Bot{api: api, router: router, log: log, miniAppURL: miniAppURL}
}

// Run blocks until ctx is cancelled, processing updates one at a time.
// Each update is wrapped in recovery + structured logging middleware so a
// single bad update can never crash the loop.
func (b *Bot) Run(ctx context.Context) {
	b.registerCommands()
	b.setMenuButton()

	pipeline := middleware.Recover(b.log, middleware.Logger(b.log, b.router.Dispatch))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	u.AllowedUpdates = []string{"message"}
	updates := b.api.GetUpdatesChan(u)

	b.log.Info("bot started, polling for updates")

	for {
		select {
		case <-ctx.Done():
			b.log.Info("shutdown signal received, stopping update loop")
			b.api.StopReceivingUpdates()
			// Drain remaining buffered updates so the goroutine inside the
			// Telegram client exits cleanly.
			for range updates {
			}
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			_ = pipeline(ctx, update)
		}
	}
}

// botCommands is the slash-command list registered with Telegram. The order
// here is the order shown in the in-chat "/" menu. Command names are latin
// ([a-z0-9_]); the descriptions are the Russian labels users see.
var botCommands = []tgbotapi.BotCommand{
	{Command: "study", Description: "▶️ Начать учебную сессию"},
	{Command: "stop", Description: "⏹ Завершить сессию"},
	{Command: "hours", Description: "⏱ Мои часы и прогресс"},
	{Command: "schedule", Description: "📅 Расписание бассейна"},
	{Command: "top", Description: "🏆 Топ-10 рейтинга"},
	{Command: "help", Description: "ℹ️ Помощь"},
	{Command: "start", Description: "🚀 Меню"},
}

// registerCommands publishes the slash-command list to Telegram so the in-chat
// "/" menu is always populated from code — no manual BotFather setup. A
// failure is non-fatal: the bot still runs, the menu just stays as it was.
func (b *Bot) registerCommands() {
	if _, err := b.api.Request(tgbotapi.NewSetMyCommands(botCommands...)); err != nil {
		b.log.Error("failed to register bot commands", slog.String("error", err.Error()))
		return
	}
	b.log.Info("bot commands registered", slog.Int("count", len(botCommands)))
}

// setMenuButton publishes the chat menu button (the control beside the message
// input) as a Web App launcher for the Mini App. It is a raw Bot API call:
// the pinned telegram-bot-api v5.5.1 predates Bot API 6.0 and has no Web App
// types at all. No-op when no Mini App URL is configured; a failure is
// non-fatal — the bot still runs, users just open the app another way.
func (b *Bot) setMenuButton() {
	if b.miniAppURL == "" {
		b.log.Info("mini app menu button skipped: no MINI_APP_URL configured")
		return
	}

	payload, err := json.Marshal(map[string]any{
		"menu_button": map[string]any{
			"type":    "web_app",
			"text":    "Открыть приложение",
			"web_app": map[string]any{"url": b.miniAppURL},
		},
	})
	if err != nil {
		b.log.Error("set menu button: marshal failed", slog.String("error", err.Error()))
		return
	}

	endpoint := "https://api.telegram.org/bot" + b.api.Token + "/setChatMenuButton"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		b.log.Error("set menu button: request failed", slog.String("error", err.Error()))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Trust the Bot API envelope, not the HTTP status: Telegram can return a
	// 200 with {"ok":false}. Decode and check `ok` either way.
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		b.log.Error("set menu button: decode response failed",
			slog.Int("status", resp.StatusCode), slog.String("error", err.Error()))
		return
	}
	if !result.OK {
		b.log.Error("set menu button: telegram rejected",
			slog.Int("status", resp.StatusCode), slog.String("description", result.Description))
		return
	}
	b.log.Info("mini app menu button set")
}
