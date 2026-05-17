package bot

import (
	"context"
	"fmt"
	"log/slog"

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
	log.Info("telegram authorized", slog.String("username", api.Self.UserName))
	return api, nil
}

// Bot owns the long-running update loop.
type Bot struct {
	api    *tgbotapi.BotAPI
	router *Router
	log    *slog.Logger
}

// New wires a Bot from an authenticated API client and a router.
func New(api *tgbotapi.BotAPI, router *Router, log *slog.Logger) *Bot {
	return &Bot{api: api, router: router, log: log}
}

// Run blocks until ctx is cancelled, processing updates one at a time.
// Each update is wrapped in recovery + structured logging middleware so a
// single bad update can never crash the loop.
func (b *Bot) Run(ctx context.Context) {
	b.registerCommands()

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
