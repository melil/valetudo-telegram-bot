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
	// /start sends 2 messages: welcome reply keyboard (1001) and main dashboard (1002)
	if len(sentMessages) != 2 {
		t.Fatalf("expected 2 sendMessage calls on /start (welcome + dashboard), got %d", len(sentMessages))
	}
	if len(editedMessages) != 0 {
		t.Fatalf("expected 0 editMessageText calls on /start, got %d", len(editedMessages))
	}
	// Verify that user message (42) was NOT deleted (critical to avoid client start loop),
	// but old dashboard message (555) was deleted.
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
	if hasDeletedUserMsg {
		t.Errorf("expected user /start message 42 NOT to be deleted, but it was deleted")
	}
	if !hasDeletedOldDash {
		t.Errorf("expected old dashboard message 555 to be deleted, deleted IDs: %v", deletedMsgIDs)
	}

	newDashboardID := b.GetDashboardMsgID()
	if newDashboardID != 1002 {
		t.Errorf("expected dashboardMsgID to be 1002, got %d", newDashboardID)
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
	if editedMessages[0].MessageID != 1002 {
		t.Errorf("expected editMessageText target to be 1002, got %d", editedMessages[0].MessageID)
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
	if len(sentMessages) != 3 {
		t.Fatalf("expected 3 total sendMessage calls, got %d", len(sentMessages))
	}
	if b.GetDashboardMsgID() != 1003 {
		t.Errorf("expected dashboardMsgID to be updated to 1003, got %d", b.GetDashboardMsgID())
	}
	// Non-start text command (43) should be deleted
	hasDeletedResourcesMsg := false
	for _, id := range deletedMsgIDs {
		if id == 43 {
			hasDeletedResourcesMsg = true
		}
	}
	if !hasDeletedResourcesMsg {
		t.Errorf("expected user /resources message 43 to be deleted, deleted IDs: %v", deletedMsgIDs)
	}
	mu.Unlock()
}

func jsonNum(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestRenderDashboardWithPhoto_Lifecycle(t *testing.T) {
	var mu sync.Mutex
	var sendPhotoCount int
	var editMediaCount int
	var editMessageTextCount int
	var deleteCount int
	var sendMsgCount int
	nextID := 2000

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		switch {
		case strings.Contains(r.URL.Path, "sendPhoto"):
			sendPhotoCount++
			nextID++
			resp := map[string]any{
				"ok":     true,
				"result": map[string]any{"message_id": nextID},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "editMessageMedia"):
			editMediaCount++
			w.Write([]byte(`{"ok":true,"result":{"message_id":` + jsonNum(nextID) + `}}`))

		case strings.Contains(r.URL.Path, "editMessageText"):
			editMessageTextCount++
			// Simulate failure when previous message was a photo message
			w.Write([]byte(`{"ok":false,"description":"Bad Request: there is no text in the message to edit"}`))

		case strings.Contains(r.URL.Path, "sendMessage"):
			sendMsgCount++
			nextID++
			resp := map[string]any{
				"ok":     true,
				"result": map[string]any{"message_id": nextID},
			}
			_ = json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "deleteMessage"):
			deleteCount++
			w.Write([]byte(`{"ok":true,"result":true}`))

		default:
			w.Write([]byte(`{"ok":true}`))
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

	// 1. Initial photo dashboard render
	photo1 := strings.NewReader("fake-png-1")
	err := b.RenderDashboardWithPhoto(12345, photo1, "Caption 1", nil)
	if err != nil {
		t.Fatalf("unexpected error rendering photo dashboard: %v", err)
	}
	if sendPhotoCount != 1 {
		t.Errorf("expected 1 sendPhoto call, got %d", sendPhotoCount)
	}
	if b.GetDashboardMsgID(12345) != 2001 {
		t.Errorf("expected dashboardMsgID to be 2001, got %d", b.GetDashboardMsgID(12345))
	}

	// 2. Edit existing photo dashboard with new photo
	photo2 := strings.NewReader("fake-png-2")
	err = b.RenderDashboardWithPhoto(12345, photo2, "Caption 2", nil)
	if err != nil {
		t.Fatalf("unexpected error editing photo dashboard: %v", err)
	}
	if editMediaCount != 1 {
		t.Errorf("expected 1 editMessageMedia call, got %d", editMediaCount)
	}

	// 3. Transition to text dashboard (e.g. Settings menu)
	err = b.RenderDashboard(12345, "Text settings menu", nil)
	if err != nil {
		t.Fatalf("unexpected error transitioning to text dashboard: %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("expected 1 deleteMessage call when transitioning from photo to text, got %d", deleteCount)
	}
	if sendMsgCount != 1 {
		t.Errorf("expected 1 sendMessage call, got %d", sendMsgCount)
	}
	if b.GetDashboardMsgID(12345) != 2002 {
		t.Errorf("expected dashboardMsgID to be 2002, got %d", b.GetDashboardMsgID(12345))
	}
}

