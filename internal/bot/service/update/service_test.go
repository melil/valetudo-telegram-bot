package update

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tgbot/internal/database"
	"tgbot/internal/telegram"
	"tgbot/internal/version"
)

type mockUserRepo struct {
	meta map[string]string
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{meta: make(map[string]string)}
}

func (m *mockUserRepo) IsAllowed(chatID int64) (bool, error) { return true, nil }
func (m *mockUserRepo) IsAdmin(chatID int64) (bool, error)   { return true, nil }
func (m *mockUserRepo) GetUser(chatID int64) (*database.User, error) {
	return &database.User{ChatID: chatID, Role: database.RoleAdmin, Locale: "ru"}, nil
}
func (m *mockUserRepo) GetAllUsers() ([]database.User, error) { return nil, nil }
func (m *mockUserRepo) GetAdmins() ([]database.User, error) {
	return []database.User{{ChatID: 12345, Role: database.RoleAdmin, Locale: "ru"}}, nil
}
func (m *mockUserRepo) GetSubscribedUsers(prefType string) ([]int64, error) { return nil, nil }
func (m *mockUserRepo) AddUser(chatID int64, username string, role database.Role) error {
	return nil
}
func (m *mockUserRepo) DeleteUser(chatID int64) error { return nil }
func (m *mockUserRepo) LogAction(chatID int64, username, action, details string) error {
	return nil
}
func (m *mockUserRepo) SetUserLocale(chatID int64, locale string) error { return nil }
func (m *mockUserRepo) SetUserNotificationPref(chatID int64, prefType string, enabled bool) error {
	return nil
}
func (m *mockUserRepo) SetUserDashboardMsgID(chatID int64, msgID int) error { return nil }
func (m *mockUserRepo) GetAllDashboardMsgIDs() (map[int64]int, error)       { return nil, nil }
func (m *mockUserRepo) GetRecentAuditLogs(limit int) ([]database.AuditLog, error) {
	return nil, nil
}
func (m *mockUserRepo) BootstrapAdmin(defaultAdminChatID int64, username string) error {
	return nil
}
func (m *mockUserRepo) GetMetadata(key string) (string, error) {
	return m.meta[key], nil
}
func (m *mockUserRepo) SetMetadata(key, value string) error {
	m.meta[key] = value
	return nil
}
func (m *mockUserRepo) DeleteMetadata(key string) error {
	delete(m.meta, key)
	return nil
}

type mockMessenger struct {
	lastSentText string
}

func (m *mockMessenger) GetUpdates(offset int) ([]telegram.Update, error) { return nil, nil }
func (m *mockMessenger) SendTextMessage(chatID int64, text string, disableNotification bool, markup any) (int, error) {
	m.lastSentText = text
	return 1, nil
}
func (m *mockMessenger) SendPayload(payload telegram.SendMessagePayload) (int, error) {
	m.lastSentText = payload.Text
	return 1, nil
}
func (m *mockMessenger) EditMessage(chatID int64, messageID int, text string, markup *telegram.InlineKeyboardMarkup) error {
	return nil
}
func (m *mockMessenger) DeleteMessage(chatID int64, messageID int) error { return nil }
func (m *mockMessenger) SendPhoto(chatID int64, photo io.Reader, caption string, disableNotification bool) error {
	return nil
}
func (m *mockMessenger) AnswerCallbackQuery(callbackQueryID string) error { return nil }
func (m *mockMessenger) AnswerCallbackQueryAlert(callbackQueryID string, text string, showAlert bool) error {
	return nil
}

func TestUpdateService_CheckForUpdate(t *testing.T) {
	// Set current version
	version.Version = "1.0.0"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/releases/latest" {
			resp := ghReleaseResponse{
				TagName:     "1.1.0",
				Name:        "Release 1.1.0",
				Body:        "- Fixed bug\n- New feature",
				HTMLURL:     "https://github.com/test/repo/releases/tag/1.1.0",
				PublishedAt: time.Now(),
				Assets: []ghReleaseAsset{
					{
						Name:               "tgbot",
						Size:               1024,
						BrowserDownloadURL: "http://example.com/tgbot",
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	svc := NewService(Config{Repo: "test/repo"}, nil, nil, nil, nil)
	svc.apiBaseURL = server.URL

	rel, hasUpdate, err := svc.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}
	if !hasUpdate {
		t.Errorf("expected hasUpdate to be true for version 1.1.0 vs 1.0.0")
	}
	if rel.Version != "1.1.0" {
		t.Errorf("expected version 1.1.0, got %s", rel.Version)
	}
	if rel.AssetURL != "http://example.com/tgbot" {
		t.Errorf("expected AssetURL http://example.com/tgbot, got %s", rel.AssetURL)
	}
}

func TestUpdateService_DismissAndNotify(t *testing.T) {
	db := newMockUserRepo()
	tg := &mockMessenger{}
	svc := NewService(Config{Repo: "test/repo", AllowedChatID: 12345}, db, tg, nil, nil)

	// Test Dismiss
	if err := svc.DismissVersion("1.2.0"); err != nil {
		t.Fatalf("DismissVersion failed: %v", err)
	}
	if !svc.IsVersionDismissed("1.2.0") {
		t.Errorf("expected 1.2.0 to be dismissed")
	}
	if svc.IsVersionDismissed("1.2.1") {
		t.Errorf("expected 1.2.1 to not be dismissed")
	}

	// Test PostUpdate notification
	version.Version = "1.2.0"
	_ = db.SetMetadata("pending_update_version", "1.2.0")
	_ = db.SetMetadata("pending_update_chat_id", "12345")

	svc.CheckAndNotifyPostUpdate(context.Background())

	if tg.lastSentText == "" {
		t.Errorf("expected message to be sent after post-update")
	}

	// Ensure markers were cleared
	pending, _ := db.GetMetadata("pending_update_version")
	if pending != "" {
		t.Errorf("expected pending_update_version to be cleared, got %s", pending)
	}
}

func TestUpdateService_ApplyUpdate(t *testing.T) {
	tempDir := t.TempDir()
	fakeExe := filepath.Join(tempDir, "tgbot")
	if err := os.WriteFile(fakeExe, []byte("old_binary"), 0755); err != nil {
		t.Fatalf("failed to create fake exe: %v", err)
	}

	// HTTP server providing binary
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("new_binary_content_12345"))
	}))
	defer server.Close()

	restarted := false
	db := newMockUserRepo()
	svc := NewService(Config{Repo: "test/repo"}, db, nil, nil, nil)
	svc.exePathFunc = func() (string, error) { return fakeExe, nil }
	svc.restartFunc = func() { restarted = true }

	rel := &ReleaseInfo{
		Version:  "1.0.1",
		AssetURL: server.URL,
	}

	err := svc.ApplyUpdate(context.Background(), rel, 12345)
	if err != nil {
		t.Fatalf("ApplyUpdate failed: %v", err)
	}

	content, err := os.ReadFile(fakeExe)
	if err != nil {
		t.Fatalf("failed to read fakeExe after update: %v", err)
	}
	if string(content) != "new_binary_content_12345" {
		t.Errorf("expected updated content 'new_binary_content_12345', got %q", string(content))
	}

	pending, _ := db.GetMetadata("pending_update_version")
	if pending != "1.0.1" {
		t.Errorf("expected pending_update_version == '1.0.1', got %q", pending)
	}

	// Wait for restart trigger
	time.Sleep(1600 * time.Millisecond)
	if !restarted {
		t.Errorf("expected restartFunc to be called")
	}
}
