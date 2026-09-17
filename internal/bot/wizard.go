package bot

import (
	"fmt"
	"strconv"
	"strings"

	"tgbot/internal/telegram"
)

type WizardSession struct {
	Mode          string
	SelectedRooms map[string]bool
	Iterations    int
	MessageID     int
}

func (b *Bot) startCleaningWizard() {
	b.wizardMu.Lock()
	b.activeWizards[b.cfg.AllowedChatID] = &WizardSession{
		SelectedRooms: make(map[string]bool),
		Iterations:    1,
	}
	b.wizardMu.Unlock()

	text, markup := b.renderWizardStep1()
	msgID, err := b.tg.SendPayload(telegram.SendMessagePayload{
		ChatID:              b.cfg.AllowedChatID,
		Text:                text,
		ParseMode:           "HTML",
		ReplyMarkup:         markup,
		DisableNotification: b.cfg.IsDNDActive(),
	})
	if err == nil && msgID != 0 {
		b.wizardMu.Lock()
		if ws, ok := b.activeWizards[b.cfg.AllowedChatID]; ok {
			ws.MessageID = msgID
		}
		b.wizardMu.Unlock()
	}
}

func (b *Bot) renderWizardStep1() (string, *telegram.InlineKeyboardMarkup) {
	text := "🪄 <b>Шаг 1 из 3: Выберите тип уборки</b>\nКак будем убирать выбранные зоны?"
	modes := []struct {
		ID   string
		Name string
	}{
		{"vacuum_and_mop", "🌪 Сухая + Влажная (одновременно)"},
		{"vacuum_then_mop", "🔄 Сначала сухая, затем влажная"},
		{"vacuum", "💨 Только сухая (пылесос)"},
		{"mop", "💧 Только влажная (швабры)"},
	}

	var rows [][]telegram.InlineKeyboardButton
	for _, m := range modes {
		btn := telegram.InlineKeyboardButton{
			Text:         m.Name,
			CallbackData: "wiz_mode:" + m.ID,
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{{Text: "❌ Отмена", CallbackData: "wiz_cancel"}})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) renderWizardStep2(ws *WizardSession) (string, *telegram.InlineKeyboardMarkup) {
	text := fmt.Sprintf("🪄 <b>Шаг 2 из 3: Выберите комнаты</b>\nРежим: <code>%s</code>\n\n<i>Отметьте одну или несколько комнат и нажмите «Далее»:</i>", formatModeTitle(ws.Mode))

	order := []string{"1", "2", "3", "4"}
	var rows [][]telegram.InlineKeyboardButton

	hasSelected := false
	for _, id := range order {
		name := b.cfg.RoomAliases[id]
		icon := "◻️"
		if ws.SelectedRooms[id] {
			icon = "✅"
			hasSelected = true
		}
		btn := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", icon, name),
			CallbackData: "wiz_toggle_room:" + id,
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}

	var controlRow []telegram.InlineKeyboardButton
	if hasSelected {
		controlRow = append(controlRow, telegram.InlineKeyboardButton{Text: "Далее ➡️", CallbackData: "wiz_to_step3"})
	}
	controlRow = append(controlRow, telegram.InlineKeyboardButton{Text: "❌ Отмена", CallbackData: "wiz_cancel"})
	rows = append(rows, controlRow)

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) renderWizardStep3(ws *WizardSession) (string, *telegram.InlineKeyboardMarkup) {
	var roomNames []string
	for id, ok := range ws.SelectedRooms {
		if ok {
			roomNames = append(roomNames, b.cfg.RoomAliases[id])
		}
	}

	text := fmt.Sprintf(
		"🪄 <b>Шаг 3 из 3: Количество проходов</b>\n\n"+
			"• <b>Режим:</b> <code>%s</code>\n"+
			"• <b>Зоны:</b> %s\n\n"+
			"Сколько раз повторить уборку выбранных зон?",
		formatModeTitle(ws.Mode), strings.Join(roomNames, ", "),
	)

	rows := [][]telegram.InlineKeyboardButton{
		{
			{Text: "1️⃣ Один проход (1x)", CallbackData: "wiz_iter:1"},
			{Text: "2️⃣ Двойной проход (2x)", CallbackData: "wiz_iter:2"},
		},
		{
			{Text: "⬅️ Назад к комнатам", CallbackData: "wiz_back_to_step2"},
			{Text: "❌ Отмена", CallbackData: "wiz_cancel"},
		},
	}

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) handleWizardCallback(cb *telegram.CallbackQuery) bool {
	data := cb.Data
	if !strings.HasPrefix(data, "wiz_") {
		return false
	}

	b.wizardMu.Lock()
	ws, exists := b.activeWizards[b.cfg.AllowedChatID]
	b.wizardMu.Unlock()

	if !exists || ws == nil {
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "⚠️ Сессия настройки устарела. Нажмите кнопку <b>🪄 Старт уборки</b> заново.", nil)
		return true
	}

	switch {
	case data == "wiz_cancel":
		b.wizardMu.Lock()
		delete(b.activeWizards, b.cfg.AllowedChatID)
		b.wizardMu.Unlock()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "❌ Настройка уборки отменена.", nil)

	case strings.HasPrefix(data, "wiz_mode:"):
		mode := strings.TrimPrefix(data, "wiz_mode:")
		ws.Mode = mode
		text, markup := b.renderWizardStep2(ws)
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, markup)

	case strings.HasPrefix(data, "wiz_toggle_room:"):
		roomID := strings.TrimPrefix(data, "wiz_toggle_room:")
		ws.SelectedRooms[roomID] = !ws.SelectedRooms[roomID]
		text, markup := b.renderWizardStep2(ws)
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, markup)

	case data == "wiz_to_step3":
		text, markup := b.renderWizardStep3(ws)
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, markup)

	case data == "wiz_back_to_step2":
		text, markup := b.renderWizardStep2(ws)
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, markup)

	case strings.HasPrefix(data, "wiz_iter:"):
		iterStr := strings.TrimPrefix(data, "wiz_iter:")
		iterations, _ := strconv.Atoi(iterStr)
		if iterations <= 0 {
			iterations = 1
		}
		ws.Iterations = iterations

		var targetIDs []string
		var targetNames []string
		for id, active := range ws.SelectedRooms {
			if active {
				targetIDs = append(targetIDs, id)
				targetNames = append(targetNames, b.cfg.RoomAliases[id])
			}
		}

		b.wizardMu.Lock()
		delete(b.activeWizards, b.cfg.AllowedChatID)
		b.wizardMu.Unlock()

		if err := b.val.SetOperationMode(ws.Mode); err != nil {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "❌ Ошибка установки режима: "+err.Error(), nil)
			return true
		}

		if err := b.val.CleanSegments(targetIDs, ws.Iterations); err != nil {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "❌ Ошибка старта сегментов: "+err.Error(), nil)
			return true
		}

		successMsg := fmt.Sprintf(
			"🚀 <b>Уборка запущена!</b>\n\n"+
				"• <b>Режим:</b> <code>%s</code>\n"+
				"• <b>Комнаты:</b> %s\n"+
				"• <b>Проходов:</b> %d",
			formatModeTitle(ws.Mode), strings.Join(targetNames, ", "), ws.Iterations,
		)
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "🤖 Открыть меню робота", CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, successMsg, markup)
	}

	return true
}
