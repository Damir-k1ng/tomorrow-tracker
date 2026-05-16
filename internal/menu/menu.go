// Package menu owns the user-facing button labels and reply keyboard.
// Both the router (for matching incoming text) and handlers (for replying
// with the keyboard) depend on it, which keeps them decoupled from each other.
package menu

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// Button labels are also the routing keys, so they're declared once and
// referenced everywhere a handler needs to match.
const (
	BtnSchedule    = "📅 Расписание"
	BtnMyHours     = "⏱ Мои часы"
	BtnStart       = "▶️ Начать сессию"
	BtnEnd         = "⏹ Завершить сессию"
	BtnLeaderboard = "🏆 Топ-10"
)

// Main builds the persistent reply keyboard shown after /start.
func Main() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnStart),
			tgbotapi.NewKeyboardButton(BtnEnd),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnMyHours),
			tgbotapi.NewKeyboardButton(BtnSchedule),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnLeaderboard),
		),
	)
	kb.ResizeKeyboard = true
	return kb
}
