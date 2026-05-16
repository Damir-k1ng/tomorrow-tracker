// Package middleware contains cross-cutting wrappers applied to every update.
package middleware

import (
	"context"
	"log/slog"
	"runtime/debug"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler is the signature every middleware wraps. Returning an error lets the
// outer pipeline log it once in a consistent shape.
type Handler func(ctx context.Context, update tgbotapi.Update) error

// Recover catches panics from downstream handlers so a single bad update can
// never take the whole bot down.
func Recover(log *slog.Logger, next Handler) Handler {
	return func(ctx context.Context, update tgbotapi.Update) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("handler panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()
		return next(ctx, update)
	}
}
