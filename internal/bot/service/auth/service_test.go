package auth

import (
	"testing"
)

type mockUserRepo struct {
	allowed bool
	admin   bool
}

func (m *mockUserRepo) IsAllowed(chatID int64) (bool, error) {
	return m.allowed, nil
}

func (m *mockUserRepo) IsAdmin(chatID int64) (bool, error) {
	return m.admin, nil
}

func (m *mockUserRepo) GetUser(chatID int64) (*databaseUserStub, error) {
	return nil, nil
}

func TestAuthService_IsAllowed(t *testing.T) {
	svc := NewService(12345, nil, nil)

	if !svc.IsUserAllowed(12345) {
		t.Errorf("expected allowedChatID to be allowed")
	}
	if svc.IsUserAllowed(99999) {
		t.Errorf("expected unknown chat ID to be disallowed")
	}
}

func TestAuthService_IsAdmin(t *testing.T) {
	svc := NewService(12345, nil, nil)

	if !svc.IsUserAdmin(12345) {
		t.Errorf("expected allowedChatID to be admin")
	}
	if svc.IsUserAdmin(99999) {
		t.Errorf("expected unknown chat ID to not be admin")
	}
}

type databaseUserStub struct{}
