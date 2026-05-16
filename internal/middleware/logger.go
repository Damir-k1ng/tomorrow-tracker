package middleware

import (
	"context"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Logger emits one structured log line per update with timing and outcome.
func Logger(log *slog.Logger, next Handler) Handler {
	return func(ctx context.Context, update tgbotapi.Update) error {
		start := time.Now()
		var (
			userID int64
			text   string
		)
		if update.Message != nil {
			text = update.Message.Text
			if update.Message.From != nil {
				userID = update.Message.From.ID
			}
		}

		err := next(ctx, update)

		attrs := []any{
			slog.Int64("update_id", int64(update.UpdateID)),
			slog.Int64("user_id", userID),
			slog.String("text", text),
			slog.Duration("took", time.Since(start)),
		}
		if err != nil {
			log.Error("update failed", append(attrs, slog.String("error", err.Error()))...)
		} else {
			log.Info("update handled", attrs...)
		}
		return err
	}
}
