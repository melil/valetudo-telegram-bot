package telegram

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"tgbot/internal/database"
	"tgbot/internal/i18n"
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
