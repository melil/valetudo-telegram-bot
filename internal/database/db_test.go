package database

import (
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

	// 1. Первый запуск: база пустая, должен создаться админ
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

	// 2. Повторный вызов бутстрапа с другим ID не должен ничего менять, так как в базе уже есть записи
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

func TestDatabase_UserLifecycle(t *testing.T) {
	db := setupTestDB(t)

	// Пользователя нет
	allowed, err := db.IsAllowed(12345)
	if err != nil {
		t.Fatalf("IsAllowed error: %v", err)
	}
	if allowed {
		t.Errorf("expected user 12345 to not be allowed")
	}

	// Добавляем обычного пользователя
	err = db.AddUser(12345, "alice", RoleUser)
	if err != nil {
		t.Fatalf("AddUser error: %v", err)
	}

	allowed, err = db.IsAllowed(12345)
	if err != nil || !allowed {
		t.Errorf("expected user 12345 to be allowed")
	}

	isAdmin, err := db.IsAdmin(12345)
	if err != nil || isAdmin {
		t.Errorf("expected user 12345 not to be admin")
	}

	// Добавляем администратора
	err = db.AddUser(67890, "bob", RoleAdmin)
	if err != nil {
		t.Fatalf("AddUser admin error: %v", err)
	}

	isAdmin, err = db.IsAdmin(67890)
	if err != nil || !isAdmin {
		t.Errorf("expected user 67890 to be admin")
	}

	// Проверяем список админов
	admins, err := db.GetAdmins()
	if err != nil {
		t.Fatalf("GetAdmins error: %v", err)
	}
	if len(admins) != 1 || admins[0].ChatID != 67890 {
		t.Errorf("expected 1 admin with id 67890, got: %v", admins)
	}

	// Обновление роли пользователя до админа
	err = db.AddUser(12345, "alice", RoleAdmin)
	if err != nil {
		t.Fatalf("AddUser update role error: %v", err)
	}

	isAdmin, err = db.IsAdmin(12345)
	if err != nil || !isAdmin {
		t.Errorf("expected user 12345 to be promoted to admin")
	}

	admins, err = db.GetAdmins()
	if err != nil {
		t.Fatalf("GetAdmins error: %v", err)
	}
	if len(admins) != 2 {
		t.Errorf("expected 2 admins, got %d", len(admins))
	}

	// Удаление пользователя
	err = db.DeleteUser(12345)
	if err != nil {
		t.Fatalf("DeleteUser error: %v", err)
	}
	allowed, err = db.IsAllowed(12345)
	if err != nil || allowed {
		t.Errorf("expected user 12345 to be deleted")
	}
}
