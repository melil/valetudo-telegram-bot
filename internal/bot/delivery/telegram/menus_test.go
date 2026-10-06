package telegram

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"tgbot/internal/database"
	"tgbot/internal/i18n"
	"tgbot/internal/valetudo"
)

type mockAuditUserRepo struct {
	logs []database.AuditLog
}

func (m *mockAuditUserRepo) IsAllowed(chatID int64) (bool, error)                { return true, nil }
func (m *mockAuditUserRepo) IsAdmin(chatID int64) (bool, error)                  { return true, nil }
func (m *mockAuditUserRepo) GetUser(chatID int64) (*database.User, error)        { return nil, nil }
func (m *mockAuditUserRepo) GetAllUsers() ([]database.User, error)               { return nil, nil }
func (m *mockAuditUserRepo) GetAdmins() ([]database.User, error)                 { return nil, nil }
func (m *mockAuditUserRepo) GetSubscribedUsers(prefType string) ([]int64, error) { return nil, nil }
func (m *mockAuditUserRepo) AddUser(chatID int64, username string, role database.Role) error {
	return nil
}
func (m *mockAuditUserRepo) DeleteUser(chatID int64) error                                  { return nil }
func (m *mockAuditUserRepo) LogAction(chatID int64, username, action, details string) error { return nil }
func (m *mockAuditUserRepo) SetUserLocale(chatID int64, locale string) error                { return nil }
func (m *mockAuditUserRepo) SetUserNotificationPref(chatID int64, prefType string, enabled bool) error {
	return nil
}
func (m *mockAuditUserRepo) SetUserDashboardMsgID(chatID int64, msgID int) error  { return nil }
func (m *mockAuditUserRepo) GetAllDashboardMsgIDs() (map[int64]int, error)        { return nil, nil }
func (m *mockAuditUserRepo) GetRecentAuditLogs(limit int) ([]database.AuditLog, error) {
	if limit > len(m.logs) {
		limit = len(m.logs)
	}
	return m.logs[:limit], nil
}
func (m *mockAuditUserRepo) GetAuditLogsPaginated(offset, limit int) ([]database.AuditLog, int, error) {
	total := len(m.logs)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return m.logs[offset:end], total, nil
}
func (m *mockAuditUserRepo) GetAuditLogByID(id int64) (*database.AuditLog, error) {
	for _, l := range m.logs {
		if l.ID == id {
			return &l, nil
		}
	}
	return nil, nil
}
func (m *mockAuditUserRepo) BootstrapAdmin(defaultAdminChatID int64, username string) error { return nil }
func (m *mockAuditUserRepo) GetMetadata(key string) (string, error)                         { return "", nil }
func (m *mockAuditUserRepo) SetMetadata(key, value string) error                            { return nil }
func (m *mockAuditUserRepo) DeleteMetadata(key string) error                                { return nil }

func TestGetAuditLogMenu_PaginationAndExpansion(t *testing.T) {
	var mockLogs []database.AuditLog
	for i := 1; i <= 25; i++ {
		mockLogs = append(mockLogs, database.AuditLog{
			ID:        int64(i),
			ChatID:    1000 + int64(i),
			Username:  fmt.Sprintf("user%d", i),
			Action:    "start_cleaning",
			Details:   fmt.Sprintf("room %d", i),
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
		})
	}

	repo := &mockAuditUserRepo{logs: mockLogs}

	// 1. Page 1, unexpanded
	text, markup := GetAuditLogMenu(repo, 1, 0, i18n.LocaleRU)
	if !strings.Contains(text, "1/3") {
		t.Errorf("expected page 1/3 indicator, got text: %s", text)
	}
	if !strings.Contains(text, "1.") || !strings.Contains(text, "10.") {
		t.Errorf("expected items 1 and 10 on page 1, got text: %s", text)
	}
	if strings.Contains(text, "ID записи:") {
		t.Errorf("expected no expanded details on unexpanded view")
	}

	// Check pagination button callback
	var hasNextPage bool
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "audit:2:0" {
				hasNextPage = true
			}
		}
	}
	if !hasNextPage {
		t.Errorf("expected button to page 2 (audit:2:0)")
	}

	// 2. Expand item #1 (ID 1) on page 1
	textExp, markupExp := GetAuditLogMenu(repo, 1, 1, i18n.LocaleRU)
	if !strings.Contains(textExp, "ID:</b> #1") {
		t.Errorf("expected expanded ID #1, got text: %s", textExp)
	}
	if !strings.Contains(textExp, "room 1") {
		t.Errorf("expected details 'room 1' in expanded text, got: %s", textExp)
	}

	// Check collapse button
	var hasCollapseBtn bool
	var hasCardBtn bool
	for _, row := range markupExp.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "audit:1:0" && strings.Contains(btn.Text, "🔼 #1") {
				hasCollapseBtn = true
			}
			if btn.CallbackData == "audit_card:1:1" {
				hasCardBtn = true
			}
		}
	}
	if !hasCollapseBtn {
		t.Errorf("expected collapse button for item #1")
	}
	if !hasCardBtn {
		t.Errorf("expected card view button for item #1")
	}

	// 3. Card menu view
	cardText, cardMarkup := GetAuditLogCardMenu(repo, 1, 1, i18n.LocaleRU)
	if !strings.Contains(cardText, "room 1") || !strings.Contains(cardText, "Детали действия #1") {
		t.Errorf("expected card view details, got: %s", cardText)
	}
	if len(cardMarkup.InlineKeyboard) == 0 {
		t.Errorf("expected non-empty card markup")
	}
}

func TestRobotAndSettingsMenusHierarchy(t *testing.T) {
	caps := valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
		string(valetudo.CapConsumableMonitoring),
		string(valetudo.CapFanSpeedControl),
		string(valetudo.CapWaterUsageControl),
	})

	// 1. Robot menu has settings split into robot settings & bot settings
	_, robotMarkup := GetRobotMenu(caps, "docked", "none", i18n.LocaleRU)
	robotCallbacks := make(map[string]bool)
	for _, row := range robotMarkup.InlineKeyboard {
		for _, btn := range row {
			robotCallbacks[btn.CallbackData] = true
		}
	}
	if !robotCallbacks["menu_robot_settings"] {
		t.Errorf("expected menu_robot_settings in robot menu")
	}
	if !robotCallbacks["menu_bot_settings"] {
		t.Errorf("expected menu_bot_settings in robot menu")
	}
	if !robotCallbacks["menu_main"] {
		t.Errorf("expected menu_main in robot menu")
	}

	// 2. Robot settings menu has physical/cleaning options
	_, robotSettingsMarkup := GetRobotSettingsMenu(caps, i18n.LocaleRU)
	robotSettingsCallbacks := make(map[string]bool)
	for _, row := range robotSettingsMarkup.InlineKeyboard {
		for _, btn := range row {
			robotSettingsCallbacks[btn.CallbackData] = true
		}
	}
	if !robotSettingsCallbacks["cmd_consumables"] {
		t.Errorf("expected cmd_consumables in robot settings")
	}
	if !robotSettingsCallbacks["cmd_telemetry"] {
		t.Errorf("expected cmd_telemetry in robot settings")
	}
	if !robotSettingsCallbacks["sub_fan"] {
		t.Errorf("expected sub_fan in robot settings")
	}
	if !robotSettingsCallbacks["sub_water"] {
		t.Errorf("expected sub_water in robot settings")
	}
	if !robotSettingsCallbacks["menu_robot"] {
		t.Errorf("expected menu_robot back button in robot settings")
	}

	// 3. Bot settings menu has system/bot options
	_, botSettingsMarkup := GetBotSettingsMenu(true, true, i18n.LocaleRU)
	botSettingsCallbacks := make(map[string]bool)
	for _, row := range botSettingsMarkup.InlineKeyboard {
		for _, btn := range row {
			botSettingsCallbacks[btn.CallbackData] = true
		}
	}
	if !botSettingsCallbacks["sub_lang"] {
		t.Errorf("expected sub_lang in bot settings")
	}
	if !botSettingsCallbacks["sub_notifications"] {
		t.Errorf("expected sub_notifications in bot settings")
	}
	if !botSettingsCallbacks["cmd_resources"] {
		t.Errorf("expected cmd_resources in bot settings")
	}
	if !botSettingsCallbacks["sub_users"] {
		t.Errorf("expected sub_users in bot settings")
	}
	if !botSettingsCallbacks["sub_audit"] {
		t.Errorf("expected sub_audit in bot settings")
	}
	if !botSettingsCallbacks["sub_updates"] {
		t.Errorf("expected sub_updates in bot settings")
	}
	if !botSettingsCallbacks["menu_robot"] {
		t.Errorf("expected menu_robot back button in bot settings")
	}
}

func TestMainDashboard_QuickCleanAndWizard(t *testing.T) {
	caps := valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
		string(valetudo.CapMapSegmentation),
	})

	_, markup := GetMainDashboard(caps, "docked", "none", nil, nil, i18n.LocaleRU)
	if len(markup.InlineKeyboard) < 1 {
		t.Fatalf("expected rows in dashboard markup")
	}

	// First row should have both wizard and quick clean
	row0 := markup.InlineKeyboard[0]
	if len(row0) != 2 {
		t.Fatalf("expected 2 buttons in row 0, got %d", len(row0))
	}
	if row0[0].CallbackData != "wiz_start" || row0[0].Text != i18n.T(i18n.LocaleRU, "main_menu.start_cleaning") {
		t.Errorf("expected wiz_start button in row 0, got %+v", row0[0])
	}
	if row0[1].CallbackData != "cmd_start" || row0[1].Text != i18n.T(i18n.LocaleRU, "main_menu.quick_clean") {
		t.Errorf("expected cmd_start button in row 0, got %+v", row0[1])
	}
}

func TestBuildHelpText(t *testing.T) {
	userHelp := BuildHelpText(false, i18n.LocaleRU)
	if !strings.Contains(userHelp, "/help") || !strings.Contains(userHelp, "/wizard") || !strings.Contains(userHelp, "/clean") {
		t.Errorf("expected user help to contain basic commands, got:\n%s", userHelp)
	}
	if strings.Contains(userHelp, "/users") || strings.Contains(userHelp, "/audit") || strings.Contains(userHelp, "/update") {
		t.Errorf("non-admin help should not contain admin commands, got:\n%s", userHelp)
	}

	adminHelp := BuildHelpText(true, i18n.LocaleRU)
	if !strings.Contains(adminHelp, "/users") || !strings.Contains(adminHelp, "/audit") || !strings.Contains(adminHelp, "/update") {
		t.Errorf("admin help should contain admin commands, got:\n%s", adminHelp)
	}
}

func TestWashTempAndDryTimeMenus(t *testing.T) {
	// 1. Wash Temp Menu
	text, markup := GetWashTempMenu([]string{"cold", "hot"}, i18n.LocaleRU)
	if !strings.Contains(text, "Температура воды") {
		t.Errorf("expected wash temp title in text, got: %s", text)
	}
	var washCallbacks []string
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			washCallbacks = append(washCallbacks, btn.CallbackData)
		}
	}
	if len(washCallbacks) < 3 || washCallbacks[0] != "set_wash_temp:cold" || washCallbacks[1] != "set_wash_temp:hot" || washCallbacks[2] != "menu_station" {
		t.Errorf("unexpected wash temp callbacks: %+v", washCallbacks)
	}

	// 2. Dry Time Menu
	text, markup = GetDryTimeMenu([]string{"2h", "3h", "4h"}, i18n.LocaleRU)
	if !strings.Contains(text, "Время сушки") {
		t.Errorf("expected dry time title in text, got: %s", text)
	}
	var dryCallbacks []string
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			dryCallbacks = append(dryCallbacks, btn.CallbackData)
		}
	}
	if len(dryCallbacks) < 4 || dryCallbacks[0] != "set_dry_time:2h" || dryCallbacks[1] != "set_dry_time:3h" || dryCallbacks[2] != "set_dry_time:4h" || dryCallbacks[3] != "menu_station" {
		t.Errorf("unexpected dry time callbacks: %+v", dryCallbacks)
	}
}

func TestStationMenu_WashTempAndDryTimeButtons(t *testing.T) {
	caps := valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
		string(valetudo.CapMopDockMopWashTemperatureControl),
		string(valetudo.CapMopDockMopDryingTimeControl),
	})

	_, markup := GetStationMenu(caps, "docked", i18n.LocaleRU)
	var callbacks []string
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			callbacks = append(callbacks, btn.CallbackData)
		}
	}

	hasWashTemp := false
	hasDryTime := false
	for _, cb := range callbacks {
		if cb == "menu_wash_temp" {
			hasWashTemp = true
		}
		if cb == "menu_dry_time" {
			hasDryTime = true
		}
	}

	if !hasWashTemp {
		t.Errorf("expected menu_wash_temp button in station menu")
	}
	if !hasDryTime {
		t.Errorf("expected menu_dry_time button in station menu")
	}
}


