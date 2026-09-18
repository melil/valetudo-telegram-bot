package bot

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func TestGetRoomsDynamic(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"id":"10","name":"Balcony"},
			{"id":"2","name":"Living Room"},
			{"id":"1","name":""}
		]`))
	}))
	defer ts.Close()

	cfg := &config.Config{
		RoomAliases: map[string]string{
			"1": "Кухня (кастом)",
		},
	}
	tg := telegram.NewClient("fake-token", "http://fake", 10)
	val := valetudo.NewClient(ts.URL, time.Second)
	b := New(cfg, tg, val)

	rooms, err := b.getRooms()
	if err != nil {
		t.Fatalf("getRooms failed: %v", err)
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

func TestGetRoomsValetudoOffline(t *testing.T) {
	cfg := &config.Config{
		RoomAliases: make(map[string]string),
	}
	tg := telegram.NewClient("fake-token", "http://fake", 10)
	// Point to unreachable port
	val := valetudo.NewClient("http://127.0.0.1:54321", 100*time.Millisecond)
	b := New(cfg, tg, val)

	rooms, err := b.getRooms()
	if err == nil {
		t.Fatalf("expected error when valetudo is offline, got rooms: %+v", rooms)
	}
}
