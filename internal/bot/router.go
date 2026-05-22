package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/handlers"
	"github.com/damirkabdulla/tomorrow-tracker/internal/menu"
)

// Router maps incoming Telegram updates to handler functions. Keeping the
// dispatch table flat (one switch) makes the supported surface obvious.
type Router struct {
	h *handlers.Handlers
}

// NewRouter wires a Router around the handler bundle.
func NewRouter(h *handlers.Handlers) *Router {
	return &Router{h: h}
}

// Dispatch is called by the middleware pipeline for every update.
func (r *Router) Dispatch(ctx context.Context, update tgbotapi.Update) error {
	msg := update.Message
	if msg == nil || msg.From == nil {
		// Edited messages, channel posts, callbacks: not used in MVP v1.
		return nil
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			return r.h.Start(ctx, msg)
		case "help":
			return r.h.Help(ctx, msg)
		case "study":
			return r.h.StartSession(ctx, msg)
		case "stop":
			return r.h.EndSession(ctx, msg)
		case "hours":
			return r.h.MyHours(ctx, msg)
		case "schedule":
			return r.h.Schedule(ctx, msg)
		case "top":
			return r.h.Leaderboard(ctx, msg)
		case "ask":
			return r.h.Ask(ctx, msg)
		case "clear":
			return r.h.ClearAI(ctx, msg)
		default:
			return r.h.Unknown(ctx, msg)
		}
	}

	switch msg.Text {
	case menu.BtnStart:
		return r.h.StartSession(ctx, msg)
	case menu.BtnEnd:
		return r.h.EndSession(ctx, msg)
	case menu.BtnMyHours:
		return r.h.MyHours(ctx, msg)
	case menu.BtnSchedule:
		return r.h.Schedule(ctx, msg)
	case menu.BtnLeaderboard:
		return r.h.Leaderboard(ctx, msg)
	case menu.BtnAIMentor:
		return r.h.EnterAIMode(ctx, msg)
	case menu.BtnExitAI:
		return r.h.ExitAIMode(ctx, msg)
	case menu.BtnClearAI:
		return r.h.ClearAI(ctx, msg)
	}

	// In free-form AI chat mode, any non-button text is forwarded to the
	// model. Outside AI mode this branch is dead, so the user falls through
	// to Unknown which steers them back to the menu.
	if r.h.InAIMode(msg.From.ID) {
		return r.h.AskFree(ctx, msg)
	}
	return r.h.Unknown(ctx, msg)
}
