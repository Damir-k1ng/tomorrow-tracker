package handlers

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// scheduleText is rendered as plain text with emoji bullets — same look as
// Telegram's native messages and immune to Markdown escaping bugs.
const scheduleText = `📅 Расписание бассейна

ПН
🟦 Quest
🛡 Raid Defence ×3

ВТ
🟦 Quest ×4

СР
🟦 Quest ×4

ЧТ
🟦 Quest ×4

ПТ
📍 Checkpoint ×4

СБ
⚔️ Raid ×3

ВС
⚔️ Raid ×3`

// Schedule handles the "📅 Расписание" button.
func (h *Handlers) Schedule(ctx context.Context, msg *tgbotapi.Message) error {
	if _, err := h.ensureUser(ctx, msg.From); err != nil {
		return err
	}
	return h.reply(msg.Chat.ID, scheduleText, nil)
}
