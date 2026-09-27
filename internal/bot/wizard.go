package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
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
		return nil, fmt.Errorf("%s: %w", b.t("rooms.err_get", ""), err)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("%s", b.t("rooms.empty"))
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
			name = fmt.Sprintf(b.t("wizard.room_default"), s.ID)
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
	if !b.Caps().Has(valetudo.CapMapSegmentation) {
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
			},
		}
		_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
		return
	}

	rooms, err := b.getRooms()
	if err != nil || len(rooms) == 0 {
		errMsg := b.t("wizard.err_get_rooms")
		if err != nil {
			errMsg += "\n" + err.Error()
		}
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
			},
		}
		_ = b.renderDashboard(errMsg, markup)
		return
	}

	session := &WizardSession{
		SelectedRooms: make(map[string]bool),
		Rooms:         rooms,
		Iterations:    1,
		MessageID:     b.GetDashboardMsgID(),
	}

	b.wizardMu.Lock()
	b.activeWizards[b.GetActiveChatID()] = session
	b.wizardMu.Unlock()

	var text string
	var markup *telegram.InlineKeyboardMarkup

	if b.Caps().Has(valetudo.CapOperationModeControl) {
		text, markup = b.renderWizardStep1()
	} else {
		text, markup = b.renderWizardStep2(session)
	}

	_ = b.renderDashboard(text, markup)
}

func (b *Bot) renderWizardStep1() (string, *telegram.InlineKeyboardMarkup) {
	text := b.t("wizard.step1_title")
	modes := []struct {
		ID   string
		Name string
	}{
		{"vacuum_and_mop", b.t("modes.wizard_vacuum_and_mop")},
		{"vacuum_then_mop", b.t("modes.wizard_vacuum_then_mop")},
		{"vacuum", b.t("modes.wizard_vacuum")},
		{"mop", b.t("modes.wizard_mop")},
	}

	var rows [][]telegram.InlineKeyboardButton
	for _, m := range modes {
		btn := telegram.InlineKeyboardButton{
			Text:         m.Name,
			CallbackData: "wiz_mode:" + m.ID,
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{{Text: b.t("wizard.btn_cancel"), CallbackData: "wiz_cancel"}})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) renderWizardStep2(ws *WizardSession) (string, *telegram.InlineKeyboardMarkup) {
	var text string
	if ws.Mode != "" {
		text = fmt.Sprintf(b.t("wizard.step2_title"), b.formatModeTitle(ws.Mode))
	} else {
		text = fmt.Sprintf(b.t("wizard.step2_title"), "—")
	}

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
		quickRow = append(quickRow, telegram.InlineKeyboardButton{Text: b.t("wizard.btn_select_none"), CallbackData: "wiz_select_none"})
	} else {
		quickRow = append(quickRow, telegram.InlineKeyboardButton{Text: b.t("wizard.btn_select_all"), CallbackData: "wiz_select_all"})
	}
	rows = append(rows, quickRow)

	var controlRow []telegram.InlineKeyboardButton
	if hasSelected {
		controlRow = append(controlRow, telegram.InlineKeyboardButton{Text: b.t("wizard.btn_next"), CallbackData: "wiz_to_step3"})
	}
	controlRow = append(controlRow, telegram.InlineKeyboardButton{Text: b.t("wizard.btn_cancel"), CallbackData: "wiz_cancel"})
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

	modeTitle := "—"
	if ws.Mode != "" {
		modeTitle = b.formatModeTitle(ws.Mode)
	}

	text := fmt.Sprintf(
		b.t("wizard.step3_title"),
		modeTitle, strings.Join(roomNames, ", "),
	)

	rows := [][]telegram.InlineKeyboardButton{
		{
			{Text: b.t("wizard.pass_1"), CallbackData: "wiz_iter:1"},
			{Text: b.t("wizard.pass_2"), CallbackData: "wiz_iter:2"},
		},
		{
			{Text: b.t("wizard.pass_3"), CallbackData: "wiz_iter:3"},
			{Text: b.t("wizard.pass_4"), CallbackData: "wiz_iter:4"},
		},
		{
			{Text: b.t("wizard.btn_back_to_rooms"), CallbackData: "wiz_back_to_step2"},
			{Text: b.t("wizard.btn_cancel"), CallbackData: "wiz_cancel"},
		},
	}

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) handleWizardCallback(cb *telegram.CallbackQuery) bool {
	data := cb.Data
	if !strings.HasPrefix(data, "wiz_") {
		return false
	}

	if data == "wiz_start" {
		b.startCleaningWizard()
		return true
	}

	b.wizardMu.Lock()
	ws, exists := b.activeWizards[cb.From.ID]
	b.wizardMu.Unlock()

	if !exists || ws == nil {
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: b.t("main_menu.start_cleaning"), CallbackData: "wiz_start"},
					{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"},
				},
			},
		}
		_ = b.renderDashboard(b.t("wizard.session_expired"), markup)
		return true
	}

	switch {
	case data == "wiz_cancel":
		b.wizardMu.Lock()
		delete(b.activeWizards, cb.From.ID)
		b.wizardMu.Unlock()
		b.sendMainDashboard()

	case strings.HasPrefix(data, "wiz_mode:"):
		mode := strings.TrimPrefix(data, "wiz_mode:")
		ws.Mode = mode
		text, markup := b.renderWizardStep2(ws)
		_ = b.renderDashboard(text, markup)

	case strings.HasPrefix(data, "wiz_toggle_room:"):
		roomID := strings.TrimPrefix(data, "wiz_toggle_room:")
		ws.SelectedRooms[roomID] = !ws.SelectedRooms[roomID]
		text, markup := b.renderWizardStep2(ws)
		_ = b.renderDashboard(text, markup)

	case data == "wiz_select_all":
		for _, r := range ws.Rooms {
			ws.SelectedRooms[r.ID] = true
		}
		text, markup := b.renderWizardStep2(ws)
		_ = b.renderDashboard(text, markup)

	case data == "wiz_select_none":
		ws.SelectedRooms = make(map[string]bool)
		text, markup := b.renderWizardStep2(ws)
		_ = b.renderDashboard(text, markup)

	case data == "wiz_to_step3":
		text, markup := b.renderWizardStep3(ws)
		_ = b.renderDashboard(text, markup)

	case data == "wiz_back_to_step2":
		text, markup := b.renderWizardStep2(ws)
		_ = b.renderDashboard(text, markup)

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
		delete(b.activeWizards, cb.From.ID)
		b.wizardMu.Unlock()

		if ws.Mode != "" && b.Caps().Has(valetudo.CapOperationModeControl) {
			if err := b.val.SetOperationMode(ws.Mode); err != nil {
				markup := &telegram.InlineKeyboardMarkup{
					InlineKeyboard: [][]telegram.InlineKeyboardButton{
						{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
					},
				}
				_ = b.renderDashboard(fmt.Sprintf(b.t("wizard.err_mode"), err.Error()), markup)
				return true
			}
		}

		if err := b.val.CleanSegments(targetIDs, ws.Iterations); err != nil {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(fmt.Sprintf(b.t("wizard.err_start_segments"), err.Error()), markup)
			return true
		}

		b.SetRobotStatus("cleaning", "none")
		b.StartSession(targetNames, b.getBatteryLevel())
		b.LogAction(cb.From.ID, "wizard_clean", strings.Join(targetNames, ", "))

		modeTitle := "—"
		if ws.Mode != "" {
			modeTitle = b.formatModeTitle(ws.Mode)
		}

		successMsg := fmt.Sprintf(
			b.t("wizard.started_title"),
			modeTitle, strings.Join(targetNames, ", "), ws.Iterations,
		)
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"},
					{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"},
				},
			},
		}
		_ = b.renderDashboard(successMsg, markup)
	}

	return true
}
