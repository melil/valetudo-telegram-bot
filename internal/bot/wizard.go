package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tgbot/internal/telegram"
)

type RoomInfo struct {
	ID   string
	Name string
}

type WizardSession struct {
	Mode          string
	SelectedRooms map[string]bool
	Rooms         []RoomInfo
	Iterations    int
	MessageID     int
}

func (b *Bot) getRooms() ([]RoomInfo, error) {
	segments, err := b.val.GetSegments()
	if err != nil {
		return nil, fmt.Errorf("ошибка связи с Valetudo API: %w", err)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("пылесос не вернул список комнат (проверьте разметку карты в Valetudo)")
	}

	var rooms []RoomInfo
	for _, s := range segments {
		name := strings.TrimSpace(s.Name)
		if alias, ok := b.cfg.RoomAliases[s.ID]; ok && alias != "" {
			name = alias
		} else if alias, ok := b.cfg.RoomAliases[s.Name]; ok && alias != "" {
			name = alias
		}
		if name == "" {
			name = "Комната " + s.ID
		}
		rooms = append(rooms, RoomInfo{ID: s.ID, Name: name})
	}

	sort.Slice(rooms, func(i, j int) bool {
		id1, err1 := strconv.Atoi(rooms[i].ID)
		id2, err2 := strconv.Atoi(rooms[j].ID)
		if err1 == nil && err2 == nil {
			return id1 < id2
		}
		return rooms[i].ID < rooms[j].ID
	})
	return rooms, nil
}

func (b *Bot) startCleaningWizard() {
	rooms, err := b.getRooms()
	if err != nil || len(rooms) == 0 {
		errMsg := "⚠️ Не удалось получить список комнат от пылесоса."
		if err != nil {
			errMsg += "\n" + err.Error()
		}
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, errMsg, b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		return
	}

	b.wizardMu.Lock()
	b.activeWizards[b.cfg.AllowedChatID] = &WizardSession{
		SelectedRooms: make(map[string]bool),
		Rooms:         rooms,
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

	var rows [][]telegram.InlineKeyboardButton
	allSelected := len(ws.Rooms) > 0
	hasSelected := false
	for _, r := range ws.Rooms {
		icon := "◻️"
		if ws.SelectedRooms[r.ID] {
			icon = "✅"
			hasSelected = true
		} else {
			allSelected = false
		}
		btn := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", icon, r.Name),
			CallbackData: "wiz_toggle_room:" + r.ID,
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}

	// Кнопка быстрого выбора всех / сброса
	var quickRow []telegram.InlineKeyboardButton
	if allSelected {
		quickRow = append(quickRow, telegram.InlineKeyboardButton{Text: "◻️ Снять все", CallbackData: "wiz_select_none"})
	} else {
		quickRow = append(quickRow, telegram.InlineKeyboardButton{Text: "☑️ Выбрать все", CallbackData: "wiz_select_all"})
	}
	rows = append(rows, quickRow)

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
	for _, r := range ws.Rooms {
		if ws.SelectedRooms[r.ID] {
			roomNames = append(roomNames, r.Name)
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
			{Text: "1️⃣ 1 проход (1x)", CallbackData: "wiz_iter:1"},
			{Text: "2️⃣ 2 прохода (2x)", CallbackData: "wiz_iter:2"},
		},
		{
			{Text: "3️⃣ 3 прохода (3x)", CallbackData: "wiz_iter:3"},
			{Text: "4️⃣ 4 прохода (4x)", CallbackData: "wiz_iter:4"},
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

	case data == "wiz_select_all":
		for _, r := range ws.Rooms {
			ws.SelectedRooms[r.ID] = true
		}
		text, markup := b.renderWizardStep2(ws)
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, markup)

	case data == "wiz_select_none":
		ws.SelectedRooms = make(map[string]bool)
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
		for _, r := range ws.Rooms {
			if ws.SelectedRooms[r.ID] {
				targetIDs = append(targetIDs, r.ID)
				targetNames = append(targetNames, r.Name)
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
