package database

import (
	"fmt"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestDatabase_BootstrapAdmin(t *testing.T) {
	db := setupTestDB(t)

	err := db.BootstrapAdmin(111222, "admin_user")
	if err != nil {
		t.Fatalf("BootstrapAdmin failed: %v", err)
	}

	u, err := db.GetUser(111222)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if u.Role != RoleAdmin {
		t.Errorf("expected role %s, got %s", RoleAdmin, u.Role)
	}
	if u.Username != "admin_user" {
		t.Errorf("expected username admin_user, got %s", u.Username)
	}
	if u.Locale != "ru" {
		t.Errorf("expected default locale ru, got %s", u.Locale)
	}

	err = db.BootstrapAdmin(999999, "other_admin")
	if err != nil {
		t.Fatalf("BootstrapAdmin 2 failed: %v", err)
	}

	allowed, err := db.IsAllowed(999999)
	if err != nil {
		t.Fatalf("IsAllowed failed: %v", err)
	}
	if allowed {
		t.Errorf("expected 999999 not to be created during second bootstrap")
	}
}

func TestDatabase_UserLifecycleAndPreferences(t *testing.T) {
	db := setupTestDB(t)

	err := db.AddUser(12345, "alice", RoleUser)
	if err != nil {
		t.Fatalf("AddUser error: %v", err)
	}

	// 1. Проверяем сохранение языка (Feature 1)
	err = db.SetUserLocale(12345, "en")
	if err != nil {
		t.Fatalf("SetUserLocale failed: %v", err)
	}
	u, err := db.GetUser(12345)
	if err != nil || u.Locale != "en" {
		t.Fatalf("expected locale 'en', got: %v", u)
	}

	// 2. Проверяем сохранение message_id дашборда (Feature 6)
	err = db.SetUserDashboardMsgID(12345, 888)
	if err != nil {
		t.Fatalf("SetUserDashboardMsgID failed: %v", err)
	}
	msgMap, err := db.GetAllDashboardMsgIDs()
	if err != nil || msgMap[12345] != 888 {
		t.Fatalf("expected msgMap[12345] == 888, got: %v", msgMap)
	}

	// 3. Проверяем настройки уведомлений (Feature 2)
	err = db.SetUserNotificationPref(12345, "errors", false)
	if err != nil {
		t.Fatalf("SetUserNotificationPref errors failed: %v", err)
	}
	u, err = db.GetUser(12345)
	if err != nil || u.NotifyErrors != false {
		t.Fatalf("expected NotifyErrors to be false, got: %v", u)
	}

	subErrors, err := db.GetSubscribedUsers("errors")
	if err != nil {
		t.Fatalf("GetSubscribedUsers errors failed: %v", err)
	}
	for _, id := range subErrors {
		if id == 12345 {
			t.Fatalf("user 12345 should not be subscribed to errors")
		}
	}

	subReports, err := db.GetSubscribedUsers("reports")
	if err != nil || len(subReports) != 1 || subReports[0] != 12345 {
		t.Fatalf("user 12345 should be subscribed to reports: %v", subReports)
	}
}

func TestDatabase_AuditLogRingBuffer(t *testing.T) {
	db := setupTestDB(t)

	// Добавляем 105 записей действий (Feature 7)
	for i := 1; i <= 105; i++ {
		err := db.LogAction(1001, "admin", "test_action", fmt.Sprintf("event #%d", i))
		if err != nil {
			t.Fatalf("LogAction #%d failed: %v", i, err)
		}
	}

	// Проверяем, что в базе осталось ровно 100 записей
	var count int
	err := db.db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&count)
	if err != nil {
		t.Fatalf("count audit_logs failed: %v", err)
	}
	if count != 100 {
		t.Fatalf("expected ring buffer of 100 items, got %d", count)
	}

	// Получаем последние 5 записей
	logs, err := db.GetRecentAuditLogs(5)
	if err != nil || len(logs) != 5 {
		t.Fatalf("expected 5 recent logs, got %v", logs)
	}
	// Самая последняя запись должна быть event #105
	if logs[0].Details != "event #105" {
		t.Errorf("expected first log to be event #105, got %s", logs[0].Details)
	}
}
