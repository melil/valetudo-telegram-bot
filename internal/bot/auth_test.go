package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/database"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func setupTestBotWithDB(t *testing.T) (*Bot, *database.DB, *[]map[string]any, *[]map[string]any, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var sentMessages []map[string]any
	var editedMessages []map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case strings.Contains(r.URL.Path, "deleteMessage"):
			w.Write([]byte(`{"ok":true,"result":true}`))

		case strings.Contains(r.URL.Path, "sendMessage"):
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			sentMessages = append(sentMessages, payload)

			resp := map[string]any{
				"ok": true,
				"result": map[string]any{
					"message_id": 999,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "editMessageText"):
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			editedMessages = append(editedMessages, payload)
			w.Write([]byte(`{"ok":true,"result":true}`))

		case strings.Contains(r.URL.Path, "answerCallbackQuery"):
			w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	t.Cleanup(ts.Close)

	tempDir := t.TempDir()
	db, err := database.Open(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("database.Open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	// Bootstrap an admin
	const adminChatID = int64(1001)
	if err := db.BootstrapAdmin(adminChatID, "admin_boss"); err != nil {
		t.Fatalf("BootstrapAdmin failed: %v", err)
	}

	cfg := &config.Config{
		AllowedChatID: adminChatID,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", ts.URL, 10)
	val := valetudo.NewClient("http://fake", time.Second)

	b := New(cfg, tg, val, db)
	return b, db, &sentMessages, &editedMessages, &mu
}

func TestBot_UnauthorizedAccessRequest(t *testing.T) {
	b, _, sentMessages, _, mu := setupTestBotWithDB(t)

	unauthorizedUserChatID := int64(2002)

	// Пользователь шлёт /start
	msg := &telegram.Message{
		MessageID: 10,
		From: &telegram.User{
			ID:        unauthorizedUserChatID,
			FirstName: "Иван",
			LastName:  "Петров",
			Username:  "ivan_p",
		},
		Chat: struct {
			ID int64 `json:"id"`
		}{ID: unauthorizedUserChatID},
		Text: "/start",
	}

	// Вызываем обработку через Run-логику
	if b.isUserAllowed(msg.Chat.ID) {
		t.Fatal("user should not be allowed yet")
	}
	b.handleUnauthorizedAccess(msg)

	mu.Lock()
	defer mu.Unlock()

	// Должно быть отправлено 2 сообщения:
	// 1 - отказ самому пользователю с его Chat ID
	// 2 - уведомление админу с кнопками Разрешить/Отклонить
	if len(*sentMessages) != 2 {
		t.Fatalf("expected 2 sent messages, got %d", len(*sentMessages))
	}

	userMsg := (*sentMessages)[0]
	if int64(userMsg["chat_id"].(float64)) != unauthorizedUserChatID {
		t.Errorf("expected first message to user %d, got %v", unauthorizedUserChatID, userMsg["chat_id"])
	}
	if !strings.Contains(userMsg["text"].(string), "⛔ <b>У вас нет доступа.</b>") {
		t.Errorf("unexpected user message text: %s", userMsg["text"])
	}

	adminMsg := (*sentMessages)[1]
	if int64(adminMsg["chat_id"].(float64)) != 1001 {
		t.Errorf("expected second message to admin 1001, got %v", adminMsg["chat_id"])
	}
	if !strings.Contains(adminMsg["text"].(string), "Запрос доступа к боту") {
		t.Errorf("unexpected admin message text: %s", adminMsg["text"])
	}
	if !strings.Contains(adminMsg["text"].(string), "@ivan_p") {
		t.Errorf("admin message should contain username @ivan_p")
	}
}

func TestBot_AdminApproveAndRejectCallback(t *testing.T) {
	b, db, sentMessages, editedMessages, mu := setupTestBotWithDB(t)

	targetChatID := int64(3003)

	// 1. Не-админ пытается нажать кнопку подтверждения
	cbFromNonAdmin := &telegram.CallbackQuery{
		ID: "cb_1",
		From: telegram.User{
			ID: 9999, // не админ
		},
		Data: "auth_approve:3003:newbie",
	}
	handled := b.handleAuthCallback(cbFromNonAdmin)
	if !handled {
		t.Fatal("expected handleAuthCallback to return true for auth_ callback")
	}

	allowed, _ := db.IsAllowed(targetChatID)
	if allowed {
		t.Fatal("user should NOT be allowed when approved by non-admin")
	}

	// 2. Настоящий админ (1001) нажимает "Разрешить"
	cbFromAdmin := &telegram.CallbackQuery{
		ID: "cb_2",
		From: telegram.User{
			ID: 1001,
		},
		Message: &telegram.Message{
			MessageID: 55,
		},
		Data: "auth_approve:3003:newbie",
	}
	handled = b.handleAuthCallback(cbFromAdmin)
	if !handled {
		t.Fatal("expected handleAuthCallback to return true for admin approve")
	}

	// Проверяем, что пользователь теперь в базе
	allowed, _ = db.IsAllowed(targetChatID)
	if !allowed {
		t.Fatal("user 3003 should now be allowed in DB")
	}
	userObj, err := db.GetUser(targetChatID)
	if err != nil || userObj.Role != database.RoleUser {
		t.Fatalf("expected role 'user', got: %v", userObj)
	}

	mu.Lock()
	// Проверяем, что админское сообщение обновилось (кнопки убраны, статус Доступ разрешен)
	if len(*editedMessages) != 1 {
		t.Fatalf("expected 1 edited message for admin, got %d", len(*editedMessages))
	}
	if !strings.Contains((*editedMessages)[0]["text"].(string), "Доступ разрешен") {
		t.Errorf("expected edited message text to contain 'Доступ разрешен', got: %v", (*editedMessages)[0]["text"])
	}

	// Проверяем, что пользователю 3003 отправлено приветственное сообщение
	foundWelcome := false
	for _, m := range *sentMessages {
		if int64(m["chat_id"].(float64)) == targetChatID && strings.Contains(m["text"].(string), "Доступ предоставлен") {
			foundWelcome = true
			break
		}
	}
	if !foundWelcome {
		t.Error("expected welcome message sent to user 3003")
	}
	mu.Unlock()

	// 3. Тест отклонения доступа (auth_reject)
	rejectedChatID := int64(4004)
	cbReject := &telegram.CallbackQuery{
		ID: "cb_3",
		From: telegram.User{
			ID: 1001,
		},
		Message: &telegram.Message{
			MessageID: 56,
		},
		Data: "auth_reject:4004",
	}
	handled = b.handleAuthCallback(cbReject)
	if !handled {
		t.Fatal("expected handleAuthCallback to return true for admin reject")
	}

	allowed, _ = db.IsAllowed(rejectedChatID)
	if allowed {
		t.Fatal("user 4004 should NOT be allowed after reject")
	}
}

func TestBot_UsersMenuAndDeletion(t *testing.T) {
	b, db, sentMessages, _, mu := setupTestBotWithDB(t)

	// Добавляем обычного пользователя bob
	const bobChatID = int64(7777)
	if err := db.AddUser(bobChatID, "bob_the_cleaner", database.RoleUser); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	// 1. Проверяем, что в меню настроек бота админа появляется кнопка "sub_users"
	b.SetActiveChatID(1001) // Админ
	_, settingsMarkup := b.getBotSettingsMenu()
	hasUsersBtn := false
	for _, row := range settingsMarkup.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "sub_users" {
				hasUsersBtn = true
			}
		}
	}
	if !hasUsersBtn {
		t.Fatal("expected 'sub_users' button in admin bot settings menu")
	}

	// 2. Открываем меню пользователей
	text, usersMarkup := b.getUsersMenu(1001)
	if !strings.Contains(text, "Управление пользователями") {
		t.Fatalf("unexpected users menu text: %s", text)
	}

	// Проверяем, что для bob есть кнопка удаления "user_del:7777", а для админа 1001 - нет
	hasBobDeleteBtn := false
	hasAdminDeleteBtn := false
	for _, row := range usersMarkup.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "user_del:7777" {
				hasBobDeleteBtn = true
			}
			if btn.CallbackData == "user_del:1001" {
				hasAdminDeleteBtn = true
			}
		}
	}
	if !hasBobDeleteBtn {
		t.Fatal("expected delete button for user 7777")
	}
	if hasAdminDeleteBtn {
		t.Fatal("admin should NOT have delete button for self")
	}

	// 3. Админ удаляет пользователя bob (нажатие кнопки user_del:7777)
	cbDel := &telegram.CallbackQuery{
		ID: "cb_del_bob",
		From: telegram.User{
			ID: 1001, // админ
		},
		Data: "user_del:7777",
	}
	b.handleCallback(cbDel)

	// Проверяем, что bob удален из БД
	bobAllowed, _ := db.IsAllowed(bobChatID)
	if bobAllowed {
		t.Fatal("user 7777 should be deleted from DB")
	}

	// Проверяем, что bob получил уведомление об отзыве доступа
	mu.Lock()
	foundRevokeMsg := false
	for _, m := range *sentMessages {
		if int64(m["chat_id"].(float64)) == bobChatID && strings.Contains(m["text"].(string), "доступ к управлению роботом был отозван") {
			foundRevokeMsg = true
			break
		}
	}
	mu.Unlock()
	if !foundRevokeMsg {
		t.Fatal("expected revoke notification sent to bob")
	}

	// 4. Попытка удалить самого себя блокируется
	cbDelSelf := &telegram.CallbackQuery{
		ID: "cb_del_self",
		From: telegram.User{
			ID: 1001,
		},
		Data: "user_del:1001",
	}
	b.handleCallback(cbDelSelf)

	adminAllowed, _ := db.IsAllowed(1001)
	if !adminAllowed {
		t.Fatal("admin should not be able to delete self")
	}
}

func TestBot_AuthLocalization(t *testing.T) {
	b, _, sentMessages, _, mu := setupTestBotWithDB(t)

	// Переключаем язык бота на английский
	b.SetLang("en")

	unauthorizedUserChatID := int64(8888)
	msg := &telegram.Message{
		MessageID: 15,
		From: &telegram.User{
			ID:        unauthorizedUserChatID,
			FirstName: "John",
			Username:  "john_doe",
		},
		Chat: struct {
			ID int64 `json:"id"`
		}{ID: unauthorizedUserChatID},
		Text: "/start",
	}

	b.handleUnauthorizedAccess(msg)

	mu.Lock()
	defer mu.Unlock()

	if len(*sentMessages) != 2 {
		t.Fatalf("expected 2 sent messages, got %d", len(*sentMessages))
	}

	userMsg := (*sentMessages)[0]
	if !strings.Contains(userMsg["text"].(string), "Access Denied") {
		t.Errorf("expected English text in userMsg, got: %s", userMsg["text"])
	}

	adminMsg := (*sentMessages)[1]
	if !strings.Contains(adminMsg["text"].(string), "Bot Access Request") {
		t.Errorf("expected English text in adminMsg, got: %s", adminMsg["text"])
	}
}

func TestBot_PerUserLocale(t *testing.T) {
	b, db, _, _, _ := setupTestBotWithDB(t)

	userChatID := int64(2002)
	if err := db.AddUser(userChatID, "ivan", database.RoleUser); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}

	// Admin changes language to "de"
	b.SetLang("de")

	// User 2002 changes language to "zh" via callback
	cb := &telegram.CallbackQuery{
		ID:   "cb_lang_zh",
		From: telegram.User{ID: userChatID, Username: "ivan"},
		Data: "set_lang:zh",
	}
	b.handleCallback(cb)

	// Verify DB values
	uAdmin, _ := db.GetUser(1001)
	if uAdmin.Locale != "de" {
		t.Errorf("expected admin locale 'de', got '%s'", uAdmin.Locale)
	}
	uUser, _ := db.GetUser(userChatID)
	if uUser.Locale != "zh" {
		t.Errorf("expected user locale 'zh', got '%s'", uUser.Locale)
	}

	// Verify GetUserLang returns independent locales
	if b.GetUserLang(1001) != "de" {
		t.Errorf("expected b.GetUserLang(admin) 'de', got '%s'", b.GetUserLang(1001))
	}
	if b.GetUserLang(userChatID) != "zh" {
		t.Errorf("expected b.GetUserLang(user) 'zh', got '%s'", b.GetUserLang(userChatID))
	}

	// Verify tUser output for both users
	adminBack := b.tUser(1001, "settings_menu.btn_back")
	userBack := b.tUser(userChatID, "settings_menu.btn_back")

	if !strings.Contains(adminBack, "Zurück") {
		t.Errorf("expected German text for admin, got: %s", adminBack)
	}
	if !strings.Contains(userBack, "返回") {
		t.Errorf("expected Chinese text for user, got: %s", userBack)
	}
}

func TestBot_NotificationPreferences(t *testing.T) {
	b, db, _, _, _ := setupTestBotWithDB(t)

	adminChatID := int64(1001)
	userChatID := int64(2002)
	_ = db.AddUser(userChatID, "ivan", database.RoleUser)

	// Initial check: both subscribed to errors
	subErrors := b.getNotifyChatIDs("errors")
	if len(subErrors) != 2 {
		t.Errorf("expected 2 subscribers for errors, got %d", len(subErrors))
	}

	// User 2002 toggles errors notification OFF
	cbToggle := &telegram.CallbackQuery{
		ID:   "cb_toggle_err",
		From: telegram.User{ID: userChatID, Username: "ivan"},
		Data: "toggle_notify:errors",
	}
	b.handleCallback(cbToggle)

	// Check DB
	u, _ := db.GetUser(userChatID)
	if u.NotifyErrors {
		t.Errorf("expected user NotifyErrors to be false")
	}
	if !u.NotifyReports {
		t.Errorf("expected user NotifyReports to remain true")
	}

	// Check getNotifyChatIDs
	subErrors = b.getNotifyChatIDs("errors")
	if len(subErrors) != 1 || subErrors[0] != adminChatID {
		t.Errorf("expected only admin in error notifications, got %v", subErrors)
	}
	subReports := b.getNotifyChatIDs("reports")
	if len(subReports) != 2 {
		t.Errorf("expected both users in report notifications, got %v", subReports)
	}
}

func TestBot_DashboardPersistence(t *testing.T) {
	b, db, _, _, _ := setupTestBotWithDB(t)

	adminChatID := int64(1001)
	userChatID := int64(2002)
	_ = db.AddUser(userChatID, "ivan", database.RoleUser)

	// Set dashboard msg IDs for both users
	b.SetDashboardMsgIDForChat(adminChatID, 1234)
	b.SetDashboardMsgIDForChat(userChatID, 5678)

	// Verify in DB
	uAdmin, _ := db.GetUser(adminChatID)
	if uAdmin.DashboardMsgID != 1234 {
		t.Errorf("expected admin DashboardMsgID 1234, got %d", uAdmin.DashboardMsgID)
	}
	uUser, _ := db.GetUser(userChatID)
	if uUser.DashboardMsgID != 5678 {
		t.Errorf("expected user DashboardMsgID 5678, got %d", uUser.DashboardMsgID)
	}

	// Create a new Bot instance sharing the same DB
	cfg := &config.Config{
		AllowedChatID: adminChatID,
		DefaultLang:   "ru",
	}
	b2 := New(cfg, nil, nil, db)

	// Verify dashboard IDs were restored in b2
	if b2.GetDashboardMsgID(adminChatID) != 1234 {
		t.Errorf("expected b2 admin dashboard msg ID 1234, got %d", b2.GetDashboardMsgID(adminChatID))
	}
	if b2.GetDashboardMsgID(userChatID) != 5678 {
		t.Errorf("expected b2 user dashboard msg ID 5678, got %d", b2.GetDashboardMsgID(userChatID))
	}
}

func TestBot_AuditLog(t *testing.T) {
	b, db, _, _, _ := setupTestBotWithDB(t)

	adminChatID := int64(1001)
	b.caps = valetudo.NewCapabilitySet([]string{string(valetudo.CapLocate)})

	// Execute several actions
	b.handleCallback(&telegram.CallbackQuery{
		ID:   "cb_locate",
		From: telegram.User{ID: adminChatID, Username: "admin_boss"},
		Data: "cmd_locate",
	})

	b.handleCallback(&telegram.CallbackQuery{
		ID:   "cb_lang",
		From: telegram.User{ID: adminChatID, Username: "admin_boss"},
		Data: "set_lang:en",
	})

	// Check audit log in DB
	logs, err := db.GetRecentAuditLogs(10)
	if err != nil {
		t.Fatalf("GetRecentAuditLogs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 audit logs, got %d", len(logs))
	}

	// Verify audit menu generation
	menuText, markup := b.getAuditLogMenu()
	if !strings.Contains(menuText, "Recent Action Log") && !strings.Contains(menuText, "Журнал последних действий") {
		t.Errorf("expected audit menu title in menuText, got: %s", menuText)
	}
	if markup == nil || len(markup.InlineKeyboard) == 0 {
		t.Errorf("expected non-empty audit menu markup")
	}
}
