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
