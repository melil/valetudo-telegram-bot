package session

import (
	"strings"
	"testing"

	"tgbot/internal/i18n"
)

func TestSessionTracking(t *testing.T) {
	svc := NewService(nil)

	if svc.IsActive() {
		t.Fatal("expected session to be inactive initially")
	}

	svc.StartSession([]string{"Гостиная", "Кухня"}, 98)
	if !svc.IsActive() {
		t.Fatal("expected session to be active")
	}

	svc.UpdateStats(10, 30, 15.5)
	svc.UpdateStats(24, 15, 32.8)
	svc.UpdateStats(20, 0, 30.0) // shouldn't decrease peak

	report := svc.FinishSession(75, i18n.LocaleRU)
	if report == nil {
		t.Fatal("expected report not to be nil")
	}
	if svc.IsActive() {
		t.Fatal("expected session to be inactive after finish")
	}

	if report.DurationMin != 24 || report.DurationSec != 15 {
		t.Errorf("expected 24 min 15 sec, got %d min %d sec", report.DurationMin, report.DurationSec)
	}
	if report.AreaM2 != 32.8 {
		t.Errorf("expected 32.8 m2, got %.1f", report.AreaM2)
	}
	if report.StartBattery != 98 || report.EndBattery != 75 || report.BatteryUsed != 23 {
		t.Errorf("battery mismatch: start=%d end=%d used=%d", report.StartBattery, report.EndBattery, report.BatteryUsed)
	}
	if !strings.Contains(report.Rooms, "Гостиная") || !strings.Contains(report.Rooms, "Кухня") {
		t.Errorf("expected rooms to contain Гостиная and Кухня, got %s", report.Rooms)
	}
}

func TestMultiPassSessionTracking(t *testing.T) {
	svc := NewService(nil)

	svc.StartSession([]string{"Kitchen", "SleepZone", "Hall", "Workspace"}, 100)
	if !svc.IsActive() {
		t.Fatal("expected session to be active")
	}

	// 1. Pass 1 (Dry cleaning): 56 min, 47.0 m²
	svc.UpdateStats(20, 0, 18.0)
	svc.UpdateStats(56, 0, 47.0)

	// 2. Robot returns to dock, switches to wet pass, resetting current stats
	// Pass 2 (Wet cleaning) starts:
	svc.UpdateStats(1, 0, 0.8)

	// Mid-cleaning dock visit to wash mops (27 min, 20.0 m²)
	svc.UpdateStats(27, 0, 20.0)

	// Wet cleaning completes (56 min, 40.0 m²)
	svc.UpdateStats(56, 0, 40.0)

	// 3. Final completion
	report := svc.FinishSession(66, i18n.LocaleRU)
	if report == nil {
		t.Fatal("expected report not to be nil")
	}

	// Total duration: 56m + 56m = 112m
	if report.DurationMin != 112 || report.DurationSec != 0 {
		t.Errorf("expected 112 min 0 sec, got %d min %d sec", report.DurationMin, report.DurationSec)
	}
	// Total area: 47.0 + 40.0 = 87.0 m²
	if report.AreaM2 != 87.0 {
		t.Errorf("expected 87.0 m2, got %.1f", report.AreaM2)
	}
	// Total battery: 100 -> 66 = 34%
	if report.StartBattery != 100 || report.EndBattery != 66 || report.BatteryUsed != 34 {
		t.Errorf("battery mismatch: start=%d end=%d used=%d", report.StartBattery, report.EndBattery, report.BatteryUsed)
	}
	// Zones must be preserved
	for _, zone := range []string{"Kitchen", "SleepZone", "Hall", "Workspace"} {
		if !strings.Contains(report.Rooms, zone) {
			t.Errorf("expected rooms to contain %s, got %s", zone, report.Rooms)
		}
	}
}
