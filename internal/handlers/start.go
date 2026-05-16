package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/damirkabdulla/tomorrow-tracker/internal/menu"
)

const welcomeText = `🚀 Добро пожаловать в Tomorrow Tracker

Этот бот поможет тебе:
• отслеживать учебные часы
• контролировать прогресс до 30 часов
• смотреть расписание бассейна
• не выпадать из темпа обучения

Создано:
Damir Kabdulla
@King_traff

Удачи в бассейне 💪`

// Start handles the /start command. It registers the user and shows the menu.
func (h *Handlers) Start(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	kb := menu.Main()
	return h.reply(msg.Chat.ID, welcomeText, &kb)
}

// Unknown is used as a fallback when the bot receives text that is not a
// recognized button or command. Showing the menu again gently steers the user
// back into the supported flow.
func (h *Handlers) Unknown(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	kb := menu.Main()
	return h.reply(msg.Chat.ID, "Используй кнопки меню ниже 👇", &kb)
}
