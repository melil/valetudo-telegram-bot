package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func TestClearHistoryAndStart(t *testing.T) {
	var mu sync.Mutex
	var deletedMsgIDs []int
	var sentMessages []telegram.SendMessagePayload
	var editedMessages []telegram.EditMessagePayload
	nextMsgID := 1000

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case strings.Contains(r.URL.Path, "deleteMessage"):
			var chatID int64
			var messageID int
			q := r.URL.Query()
			_ = json.Unmarshal([]byte(q.Get("chat_id")), &chatID)
			_ = json.Unmarshal([]byte(q.Get("message_id")), &messageID)
			deletedMsgIDs = append(deletedMsgIDs, messageID)
			w.Write([]byte(`{"ok":true,"result":true}`))

		case strings.Contains(r.URL.Path, "editMessageText"):
			var payload telegram.EditMessagePayload
			_ = json.NewDecoder(r.Body).Decode(&payload)
			editedMessages = append(editedMessages, payload)
			w.Write([]byte(`{"ok":true,"result":{"message_id":` + jsonNum(payload.MessageID) + `}}`))

		case strings.Contains(r.URL.Path, "sendMessage"):
			var payload telegram.SendMessagePayload
			_ = json.NewDecoder(r.Body).Decode(&payload)
			sentMessages = append(sentMessages, payload)
			nextMsgID++
			resp := map[string]any{
				"ok": true,
				"result": map[string]any{
					"message_id": nextMsgID,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "answerCallbackQuery"):
			w.Write([]byte(`{"ok":true,"result":true}`))
		}
	}))
	defer ts.Close()

	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", ts.URL, 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	// Simulating bot that has an existing dashboard message 555 from before history was cleared
	b.SetDashboardMsgID(555)

	// 1. User cleared history and types /start
	b.handleTextCommand(&telegram.Message{
		MessageID: 42,
		Chat: struct {
			ID int64 `json:"id"`
		}{ID: 12345},
		Text: "/start",
	})

	mu.Lock()
	if len(sentMessages) != 1 {
		t.Fatalf("expected 1 sendMessage call, got %d", len(sentMessages))
	}
	if len(editedMessages) != 0 {
		t.Fatalf("expected 0 editMessageText calls on /start, got %d", len(editedMessages))
	}
	// Verify that user message (42) and old dashboard message (555) were deleted
	hasDeletedUserMsg := false
	hasDeletedOldDash := false
	for _, id := range deletedMsgIDs {
		if id == 42 {
			hasDeletedUserMsg = true
		}
		if id == 555 {
			hasDeletedOldDash = true
		}
	}
	if !hasDeletedUserMsg {
		t.Errorf("expected user message 42 to be deleted, deleted IDs: %v", deletedMsgIDs)
	}
	if !hasDeletedOldDash {
		t.Errorf("expected old dashboard message 555 to be deleted, deleted IDs: %v", deletedMsgIDs)
	}

	newDashboardID := b.GetDashboardMsgID()
	if newDashboardID != 1001 {
		t.Errorf("expected dashboardMsgID to be 1001, got %d", newDashboardID)
	}
	mu.Unlock()

	// 2. User clicks an inline button on the newly created dashboard (e.g. menu_robot)
	b.handleCallback(&telegram.CallbackQuery{
		ID: "cb_test_1",
		Message: &telegram.Message{
			MessageID: newDashboardID,
		},
		Data: "menu_robot",
	})

	mu.Lock()
	if len(editedMessages) != 1 {
		t.Fatalf("expected 1 editMessageText call on inline button click, got %d", len(editedMessages))
	}
	if editedMessages[0].MessageID != 1001 {
		t.Errorf("expected editMessageText target to be 1001, got %d", editedMessages[0].MessageID)
	}
	mu.Unlock()

	// 3. User types /resources command
	b.handleTextCommand(&telegram.Message{
		MessageID: 43,
		Chat: struct {
			ID int64 `json:"id"`
		}{ID: 12345},
		Text: "/resources",
	})

	mu.Lock()
	if len(sentMessages) != 2 {
		t.Fatalf("expected 2 total sendMessage calls, got %d", len(sentMessages))
	}
	if b.GetDashboardMsgID() != 1002 {
		t.Errorf("expected dashboardMsgID to be updated to 1002, got %d", b.GetDashboardMsgID())
	}
	mu.Unlock()
}

func jsonNum(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
