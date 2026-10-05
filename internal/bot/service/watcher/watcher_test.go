package watcher

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"tgbot/internal/bot/domain"
	"tgbot/internal/bot/service/auth"
	"tgbot/internal/bot/service/session"
	"tgbot/internal/database"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

type mockRobotClient struct {
	mu         sync.Mutex
	attrs      []valetudo.GenericAttribute
	min        int
	sec        int
	areaM2     float64
	totalHours int
	totalCount int
	totalArea  float64
}

func (m *mockRobotClient) GetStatus() (valetudo.RobotStatus, error) {
	return valetudo.RobotStatus{Value: "unknown", Flag: "none"}, nil
}
func (m *mockRobotClient) GetAttributes() ([]valetudo.GenericAttribute, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attrs, nil
}
func (m *mockRobotClient) SetAttributes(attrs []valetudo.GenericAttribute) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attrs = attrs
}
func (m *mockRobotClient) GetCapabilities() ([]string, error)                        { return nil, nil }
func (m *mockRobotClient) TriggerAction(action string) error                         { return nil }
func (m *mockRobotClient) TriggerCapabilityAction(capability, action string) error   { return nil }
func (m *mockRobotClient) SetPreset(capability, value string) error                  { return nil }
func (m *mockRobotClient) GetSegments() ([]valetudo.MapSegment, error)               { return nil, nil }
func (m *mockRobotClient) CleanSegments(segmentIDs []string, iterations int) error  { return nil }
func (m *mockRobotClient) GetConsumables() ([]valetudo.ConsumableItem, error)        { return nil, nil }
func (m *mockRobotClient) GetConsumableProperties() (*valetudo.ConsumableProperties, error) {
	return nil, nil
}
func (m *mockRobotClient) ResetConsumable(cType, subType string) error { return nil }
func (m *mockRobotClient) GetMapReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("fake-png")), nil
}
func (m *mockRobotClient) SetOperationMode(mode string) error { return nil }
func (m *mockRobotClient) GetCurrentSessionStats() (int, int, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.min, m.sec, m.areaM2
}
func (m *mockRobotClient) GetTotalStats() (int, int, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.totalHours, m.totalCount, m.totalArea
}

type mockMessenger struct {
	mu           sync.Mutex
	sentMessages []string
	sentPhotos   []string
}

func (m *mockMessenger) GetUpdates(offset int) ([]telegram.Update, error) { return nil, nil }
func (m *mockMessenger) SendTextMessage(chatID int64, text string, disableNotification bool, markup any) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, text)
	return 1, nil
}
func (m *mockMessenger) SendPayload(payload telegram.SendMessagePayload) (int, error) {
	return 1, nil
}
func (m *mockMessenger) EditMessage(chatID int64, messageID int, text string, markup *telegram.InlineKeyboardMarkup) error {
	return nil
}
func (m *mockMessenger) DeleteMessage(chatID int64, messageID int) error { return nil }
func (m *mockMessenger) SendPhoto(chatID int64, photo io.Reader, caption string, disableNotification bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentPhotos = append(m.sentPhotos, caption)
	return nil
}
func (m *mockMessenger) AnswerCallbackQuery(callbackQueryID string) error { return nil }
func (m *mockMessenger) AnswerCallbackQueryAlert(callbackQueryID string, text string, showAlert bool) error {
	return nil
}

type mockUserRepo struct{}

func (m *mockUserRepo) IsAllowed(chatID int64) (bool, error)                { return true, nil }
func (m *mockUserRepo) IsAdmin(chatID int64) (bool, error)                  { return true, nil }
func (m *mockUserRepo) GetUser(chatID int64) (*database.User, error)        { return nil, nil }
func (m *mockUserRepo) GetAllUsers() ([]database.User, error)               { return nil, nil }
func (m *mockUserRepo) GetAdmins() ([]database.User, error)                 { return nil, nil }
func (m *mockUserRepo) GetSubscribedUsers(prefType string) ([]int64, error) { return []int64{123}, nil }
func (m *mockUserRepo) AddUser(chatID int64, username string, role database.Role) error {
	return nil
}
func (m *mockUserRepo) DeleteUser(chatID int64) error                                  { return nil }
func (m *mockUserRepo) LogAction(chatID int64, username, action, details string) error { return nil }
func (m *mockUserRepo) SetUserLocale(chatID int64, locale string) error                { return nil }
func (m *mockUserRepo) SetUserNotificationPref(chatID int64, prefType string, enabled bool) error {
	return nil
}
func (m *mockUserRepo) SetUserDashboardMsgID(chatID int64, msgID int) error  { return nil }
func (m *mockUserRepo) GetAllDashboardMsgIDs() (map[int64]int, error)        { return nil, nil }
func (m *mockUserRepo) GetRecentAuditLogs(limit int) ([]database.AuditLog, error) {
	return nil, nil
}
func (m *mockUserRepo) BootstrapAdmin(defaultAdminChatID int64, username string) error { return nil }
func (m *mockUserRepo) GetMetadata(key string) (string, error)                         { return "", nil }
func (m *mockUserRepo) SetMetadata(key, value string) error                            { return nil }
func (m *mockUserRepo) DeleteMetadata(key string) error                                { return nil }

func TestWatcher_MidCleanDockingDoesNotPrematurelyFinishSession(t *testing.T) {
	mockVal := &mockRobotClient{}
	mockTg := &mockMessenger{}
	authSvc := auth.NewService(123, &mockUserRepo{}, mockTg)
	sessionSvc := session.NewService(mockVal)

	svc := NewService(
		mockVal,
		mockTg,
		authSvc,
		sessionSvc,
		Config{
			Interval: 10 * time.Millisecond,
			FormatCaption: func(report *domain.CleaningReport, chatID int64) string {
				return "Report: " + report.Rooms
			},
			TranslateUser: func(chatID int64, key string, args ...any) string {
				return key
			},
		},
	)

	// Start with robot docked at baseline
	mockVal.SetAttributes([]valetudo.GenericAttribute{
		{Class: "StatusStateAttribute", Value: "docked", Flag: "none"},
		{Class: "BatteryStateAttribute", Level: 100},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.Start(ctx)
	time.Sleep(30 * time.Millisecond) // initial tick (firstRun)

	// 1. Wizard starts session for specific rooms
	sessionSvc.StartSession([]string{"Kitchen", "Hall"}, 100)
	if !sessionSvc.IsActive() {
		t.Fatal("expected session to be active")
	}

	// 2. Robot begins cleaning
	mockVal.SetAttributes([]valetudo.GenericAttribute{
		{Class: "StatusStateAttribute", Value: "cleaning", Flag: "segment"},
		{Class: "BatteryStateAttribute", Level: 95},
	})
	mockVal.mu.Lock()
	mockVal.min, mockVal.sec, mockVal.areaM2 = 25, 0, 20.0
	mockVal.mu.Unlock()
	time.Sleep(35 * time.Millisecond)

	// 3. Robot docks mid-clean to wash mop (status: docked, flag: resumable, dock: cleaning)
	mockVal.SetAttributes([]valetudo.GenericAttribute{
		{Class: "StatusStateAttribute", Value: "docked", Flag: "resumable"},
		{Class: "DockStatusStateAttribute", Value: "cleaning"},
		{Class: "BatteryStateAttribute", Level: 85},
	})
	time.Sleep(45 * time.Millisecond)

	// Check: session must still be active, and NO report photo/text sent!
	if !sessionSvc.IsActive() {
		t.Fatal("expected session to remain active while robot is mid-clean docked")
	}
	mockTg.mu.Lock()
	if len(mockTg.sentPhotos) > 0 || len(mockTg.sentMessages) > 0 {
		t.Fatalf("expected no reports sent during mid-clean dock, got photos=%d msgs=%d", len(mockTg.sentPhotos), len(mockTg.sentMessages))
	}
	mockTg.mu.Unlock()

	// 4. Robot leaves dock to continue cleaning
	mockVal.SetAttributes([]valetudo.GenericAttribute{
		{Class: "StatusStateAttribute", Value: "cleaning", Flag: "segment"},
		{Class: "BatteryStateAttribute", Level: 83},
	})
	mockVal.mu.Lock()
	mockVal.min, mockVal.sec, mockVal.areaM2 = 45, 0, 42.0
	mockVal.mu.Unlock()
	time.Sleep(35 * time.Millisecond)

	// 5. Robot finishes cleaning completely and docks (status: docked, flag: none, dock: drying)
	mockVal.SetAttributes([]valetudo.GenericAttribute{
		{Class: "StatusStateAttribute", Value: "docked", Flag: "none"},
		{Class: "DockStatusStateAttribute", Value: "drying"},
		{Class: "BatteryStateAttribute", Level: 70},
	})
	time.Sleep(50 * time.Millisecond)

	// Now session should be finished!
	if sessionSvc.IsActive() {
		t.Fatal("expected session to be finished after final dock")
	}

	// Report must have been sent with original rooms
	mockTg.mu.Lock()
	if len(mockTg.sentPhotos) != 1 {
		t.Fatalf("expected exactly 1 photo report, got %d", len(mockTg.sentPhotos))
	}
	caption := mockTg.sentPhotos[0]
	mockTg.mu.Unlock()

	if !strings.Contains(caption, "Kitchen") || !strings.Contains(caption, "Hall") {
		t.Errorf("expected final report to contain original rooms, got %s", caption)
	}
}
