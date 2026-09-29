package consumables

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tgbot/internal/i18n"
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
					"type": "filter",
					"subType": "main",
					"remaining": {"value": 5400, "unit": "minutes"}
				}
			]`))
		case "/capabilities/ConsumableMonitoringCapability/properties":
			w.Write([]byte(`{
				"availableConsumables": [
					{"type": "brush", "subType": "main", "unit": "minutes", "maxValue": 18000},
					{"type": "filter", "subType": "main", "unit": "minutes", "maxValue": 9000}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	val := valetudo.NewClient(ts.URL, time.Second)
	svc := NewService(val)

	items, err := svc.GetConsumablesDisplay(i18n.LocaleRU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// 14400 / 18000 = 80%
	if items[0].Percent != 80 {
		t.Errorf("expected 80%%, got %d%%", items[0].Percent)
	}
	if items[0].RemainingH != 240 || items[0].MaxH != 300 {
		t.Errorf("hours mismatch: remaining=%d max=%d", items[0].RemainingH, items[0].MaxH)
	}

	// 5400 / 9000 = 60%
	if items[1].Percent != 60 {
		t.Errorf("expected 60%%, got %d%%", items[1].Percent)
	}
}
