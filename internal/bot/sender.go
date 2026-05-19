package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Sender sends plain-text Telegram messages on behalf of broadcast-style
// callers. It implements services.MessageSender.
type Sender struct {
	api *tgbotapi.BotAPI
	log *slog.Logger
}

// NewSender wires a Sender over an authenticated Bot API client.
func NewSender(api *tgbotapi.BotAPI, log *slog.Logger) *Sender {
	return &Sender{api: api, log: log}
}

// SendMessage delivers text to chatID as a plain-text message. On a Telegram
// 429 ("retry after N") it waits the requested delay once and retries; any
// other error (notably 403 — the user has blocked the bot) is returned so the
// caller can count a failed recipient and move on.
func (s *Sender) SendMessage(ctx context.Context, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.api.Send(msg)
	if err == nil {
		return nil
	}

	var tgErr *tgbotapi.Error
	if errors.As(err, &tgErr) && tgErr.RetryAfter > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(tgErr.RetryAfter) * time.Second):
		}
		if _, retryErr := s.api.Send(msg); retryErr != nil {
			return fmt.Errorf("send after retry: %w", retryErr)
		}
		return nil
	}
	return fmt.Errorf("send message: %w", err)
}
