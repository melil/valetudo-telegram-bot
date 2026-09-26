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

	// 1. Проверяем, что в меню настроек админа появляется кнопка "sub_users"
	b.SetActiveChatID(1001) // Админ
	_, settingsMarkup := b.getSettingsMainMenu()
	hasUsersBtn := false
	for _, row := range settingsMarkup.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "sub_users" {
				hasUsersBtn = true
			}
		}
	}
	if !hasUsersBtn {
		t.Fatal("expected 'sub_users' button in admin settings menu")
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
