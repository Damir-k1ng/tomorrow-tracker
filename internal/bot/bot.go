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
