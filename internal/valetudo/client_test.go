package valetudo

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetSegments(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/capabilities/MapSegmentationCapability" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"__class":"ValetudoMapSegment","id":"1","name":"Kitchen"},
			{"__class":"ValetudoMapSegment","id":"2","name":"SleepZone"},
			{"__class":"ValetudoMapSegment","id":"3","name":"Hall"},
			{"__class":"ValetudoMapSegment","id":"4","name":"Workspace"}
		]`))
	}))
	defer ts.Close()

	c := NewClient(ts.URL, 2*time.Second)
	segments, err := c.GetSegments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(segments))
	}

	expected := []struct {
		id   string
		name string
	}{
		{"1", "Kitchen"},
		{"2", "SleepZone"},
		{"3", "Hall"},
		{"4", "Workspace"},
	}

	for i, exp := range expected {
		if segments[i].ID != exp.id || segments[i].Name != exp.name {
			t.Errorf("segment[%d]: expected %+v, got %+v", i, exp, segments[i])
		}
	}
}

func TestGetConsumablesAndProperties(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/capabilities/ConsumableMonitoringCapability":
			w.Write([]byte(`[
				{"type":"brush","subType":"main","remaining":{"value":14400,"unit":"minutes"}},
				{"type":"filter","subType":"main","remaining":{"value":5400,"unit":"minutes"}}
			]`))
		case "/capabilities/ConsumableMonitoringCapability/properties":
			w.Write([]byte(`{
				"availableConsumables": [
					{"type":"brush","subType":"main","unit":"minutes","maxValue":18000},
					{"type":"filter","subType":"main","unit":"minutes","maxValue":9000}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL, 2*time.Second)
	items, err := c.GetConsumables()
	if err != nil {
		t.Fatalf("unexpected GetConsumables error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Remaining.Value != 14400 {
		t.Errorf("expected remaining value 14400, got %d", items[0].Remaining.Value)
	}

	props, err := c.GetConsumableProperties()
	if err != nil {
		t.Fatalf("unexpected GetConsumableProperties error: %v", err)
	}
	if len(props.AvailableConsumables) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(props.AvailableConsumables))
	}
	if props.AvailableConsumables[0].MaxValue != 18000 {
		t.Errorf("expected max value 18000, got %d", props.AvailableConsumables[0].MaxValue)
	}
}
