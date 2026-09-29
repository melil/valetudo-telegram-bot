package cleaning

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tgbot/internal/i18n"
	"tgbot/internal/valetudo"
)

func TestGetRoomsDynamic(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]string{
			{"id": "10", "name": "Balcony"},
			{"id": "2", "name": "Living Room"},
			{"id": "1", "name": ""},
		})
	}))
	defer ts.Close()

	val := valetudo.NewClient(ts.URL, time.Second)
	roomAliases := map[string]string{
		"1": "Кухня (кастом)",
	}
	svc := NewWizardService(val, roomAliases)

	rooms, err := svc.GetRooms(i18n.LocaleRU)
	if err != nil {
		t.Fatalf("GetRooms failed: %v", err)
	}

	if len(rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(rooms))
	}

	// Should be sorted numerically: 1, 2, 10
	if rooms[0].ID != "1" || rooms[0].Name != "Кухня (кастом)" {
		t.Errorf("expected room 0 to be ID 1 with alias, got %+v", rooms[0])
	}
	if rooms[1].ID != "2" || rooms[1].Name != "Living Room" {
		t.Errorf("expected room 1 to be ID 2 Living Room, got %+v", rooms[1])
	}
	if rooms[2].ID != "10" || rooms[2].Name != "Balcony" {
		t.Errorf("expected room 2 to be ID 10 Balcony, got %+v", rooms[2])
	}
}

func TestWizardStateTransitions(t *testing.T) {
	svc := NewWizardService(nil, nil)

	const chatID = int64(42)
	sess := svc.StartSession(chatID, 100, nil)
	if sess == nil {
		t.Fatal("expected session to be created")
	}

	if !svc.SetMode(chatID, "vacuum") {
		t.Error("SetMode failed")
	}

	retrieved, ok := svc.GetSession(chatID)
	if !ok || retrieved.Mode != "vacuum" {
		t.Errorf("expected mode vacuum, got %v", retrieved)
	}

	svc.CancelSession(chatID)
	_, okAfter := svc.GetSession(chatID)
	if okAfter {
		t.Error("expected session to be removed after cancel")
	}
}
