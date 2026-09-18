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

func TestGetConsumablesDisplay(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/capabilities/ConsumableMonitoringCapability":
			w.Write([]byte(`[
				{
					"type": "brush",
					"subType": "main",
					"remaining": {"value": 14400, "unit": "minutes"}
				},
				{
					"type": "brush",
					"subType": "side_right",
					"remaining": {"value": 8400, "unit": "minutes"}
				},
				{
					"type": "filter",
					"subType": "main",
					"remaining": {"value": 5400, "unit": "minutes"}
				},
				{
					"type": "cleaning",
					"subType": "sensor",
					"remaining": {"value": 1680, "unit": "minutes"}
				}
			]`))
		case "/capabilities/ConsumableMonitoringCapability/properties":
			w.Write([]byte(`{
				"availableConsumables": [
					{"type": "brush", "subType": "main", "unit": "minutes", "maxValue": 18000},
					{"type": "brush", "subType": "side_right", "unit": "minutes", "maxValue": 12000},
					{"type": "filter", "subType": "main", "unit": "minutes", "maxValue": 9000},
					{"type": "cleaning", "subType": "sensor", "unit": "minutes", "maxValue": 1800}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cfg := &config.Config{RoomAliases: make(map[string]string)}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient(ts.URL, time.Second)
	b := New(cfg, tg, val)

	displays, err := b.getConsumablesDisplay()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(displays) != 4 {
		t.Fatalf("expected 4 displays, got %d", len(displays))
	}

	// 1. Main brush: 14400 min = 240h out of 300h (80%), 10d 0h
	mb := displays[0]
	if mb.RemainingH != 240 {
		t.Errorf("main brush remainingH: expected 240, got %d", mb.RemainingH)
	}
	if mb.MaxH != 300 {
		t.Errorf("main brush maxH: expected 300, got %d", mb.MaxH)
	}
	if mb.Percent != 80 {
		t.Errorf("main brush percent: expected 80, got %d", mb.Percent)
	}
	if mb.RemainingFormatted != "240 ч (10 д 0 ч)" {
		t.Errorf("main brush remainingFormatted: expected '240 ч (10 д 0 ч)', got '%s'", mb.RemainingFormatted)
	}

	// 2. Side brush: 8400 min = 140h out of 200h (70%), 5d 20h
	sb := displays[1]
	if sb.RemainingH != 140 {
		t.Errorf("side brush remainingH: expected 140, got %d", sb.RemainingH)
	}
	if sb.MaxH != 200 {
		t.Errorf("side brush maxH: expected 200, got %d", sb.MaxH)
	}
	if sb.Percent != 70 {
		t.Errorf("side brush percent: expected 70, got %d", sb.Percent)
	}
	if sb.RemainingFormatted != "140 ч (5 д 20 ч)" {
		t.Errorf("side brush remainingFormatted: expected '140 ч (5 д 20 ч)', got '%s'", sb.RemainingFormatted)
	}

	// 3. Filter: 5400 min = 90h out of 150h (60%), 3d 18h
	fl := displays[2]
	if fl.RemainingH != 90 {
		t.Errorf("filter remainingH: expected 90, got %d", fl.RemainingH)
	}
	if fl.MaxH != 150 {
		t.Errorf("filter maxH: expected 150, got %d", fl.MaxH)
	}
	if fl.Percent != 60 {
		t.Errorf("filter percent: expected 60, got %d", fl.Percent)
	}

	// 4. Sensors: 1680 min = 28h out of 30h (93%), 1d 4h
	sn := displays[3]
	if sn.RemainingH != 28 {
		t.Errorf("sensor remainingH: expected 28, got %d", sn.RemainingH)
	}
	if sn.MaxH != 30 {
		t.Errorf("sensor maxH: expected 30, got %d", sn.MaxH)
	}
	if sn.Percent != 93 {
		t.Errorf("sensor percent: expected 93, got %d", sn.Percent)
	}
}

func TestFormatRemainingTime(t *testing.T) {
	cfg := &config.Config{RoomAliases: make(map[string]string), DefaultLang: "ru"}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	cases := []struct {
		mins     int
		expected string
	}{
		{0, "исчерпан (0 мин)"},
		{-10, "исчерпан (0 мин)"},
		{45, "45 мин"},
		{60, "1 ч"},
		{90, "1 ч 30 мин"},
		{1680, "28 ч (1 д 4 ч)"},
		{14400, "240 ч (10 д 0 ч)"},
	}

	for _, c := range cases {
		got := b.formatRemainingTime(c.mins)
		if got != c.expected {
			t.Errorf("mins %d: expected '%s', got '%s'", c.mins, c.expected, got)
		}
	}

	// Test English formatting
	b.SetLang("en")
	if got := b.formatRemainingTime(14400); got != "240 h (10 d 0 h)" {
		t.Errorf("expected EN '240 h (10 d 0 h)', got '%s'", got)
	}
	if got := b.formatRemainingTime(0); got != "depleted (0 min)" {
		t.Errorf("expected EN 'depleted (0 min)', got '%s'", got)
	}
}
