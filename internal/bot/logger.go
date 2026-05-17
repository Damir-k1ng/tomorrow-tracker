package bot

import (
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// slogBotLogger adapts the telegram-bot-api package logger onto the app's
// structured slog logger.
//
// By default the library writes to raw stderr through the stdlib logger. The
// most visible line is the transient
//
//	Conflict: terminated by other getUpdates request; make sure that only one
//	bot instance is running
//
// emitted while the old and new containers briefly overlap during a Railway
// rolling deploy — two instances poll getUpdates for a few seconds until the
// old one drains. It is self-healing (the library retries) and harmless, but
// as raw red stderr it looks like an incident. Routing it through slog turns
// it into a single structured `warn` line, consistent with every other log
// and filterable by operators.
type slogBotLogger struct{ log *slog.Logger }

// installBotLogger points the telegram-bot-api package logger at slog. It is
// called once at startup; SetLogger only fails on a nil logger, so the error
// is impossible here and intentionally discarded.
func installBotLogger(log *slog.Logger) {
	_ = tgbotapi.SetLogger(slogBotLogger{log: log})
}

func (l slogBotLogger) Println(v ...any) {
	l.log.Warn("telegram client", slog.String("detail", strings.TrimSpace(fmt.Sprintln(v...))))
}

func (l slogBotLogger) Printf(format string, v ...any) {
	l.log.Warn("telegram client", slog.String("detail", strings.TrimSpace(fmt.Sprintf(format, v...))))
}
