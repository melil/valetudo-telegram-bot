package bot

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"tgbot/internal/config"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func TestSessionTrackingAndReport(t *testing.T) {
	cfg := &config.Config{DefaultLang: "ru"}
	b := New(cfg, telegram.NewClient("token", "", 30), valetudo.NewClient("http://localhost:8080/api/v2/robot", time.Second))
	b.SetCaps(valetudo.NewCapabilitySet([]string{string(valetudo.CapBasicControl)}))

	// 1. Session start
	b.StartSession([]string{"Гостиная", "Кухня"}, 98)
	if !b.IsSessionActive() {
		t.Fatalf("expected session to be active")
	}

	// 2. Metrics update during cleaning
	b.UpdateSessionStats(10, 30, 15.5)
	b.UpdateSessionStats(24, 15, 32.8)
	// Peak should be 24m 15s and 32.8 m2
	b.UpdateSessionStats(20, 0, 30.0) // smaller values shouldn't decrease peak

	// 3. Finish session
	report := b.FinishSession(75)
	if report == nil {
		t.Fatalf("expected report not to be nil")
	}
	if b.IsSessionActive() {
		t.Fatalf("expected session to be inactive after finish")
	}

	if report.DurationMin != 24 || report.DurationSec != 15 {
		t.Errorf("expected 24 min 15 sec, got %d min %d sec", report.DurationMin, report.DurationSec)
	}
	if report.AreaM2 != 32.8 {
		t.Errorf("expected 32.8 m2, got %.1f", report.AreaM2)
	}
	if report.StartBattery != 98 || report.EndBattery != 75 || report.BatteryUsed != 23 {
		t.Errorf("battery stats mismatch: start=%d end=%d used=%d", report.StartBattery, report.EndBattery, report.BatteryUsed)
	}
	if !strings.Contains(report.Rooms, "Гостиная") || !strings.Contains(report.Rooms, "Кухня") {
		t.Errorf("expected rooms to contain Гостиная and Кухня, got %s", report.Rooms)
	}

	// 4. Test Alert formatting and 200 rune Telegram limit
	alertText := b.formatReportAlert(report)
	runeCount := utf8.RuneCountInString(alertText)
	if runeCount > 200 {
		t.Errorf("alert text exceeds Telegram limit of 200 runes: got %d", runeCount)
	}
	if !strings.Contains(alertText, "24 мин 15 сек") {
		t.Errorf("alert text missing time: %s", alertText)
	}
	if !strings.Contains(alertText, "32.8 м²") {
		t.Errorf("alert text missing area: %s", alertText)
	}

	// 5. Test Caption formatting
	caption := b.formatReportCaption(report)
	if !strings.Contains(caption, "<b>Уборка завершена!</b>") {
		t.Errorf("caption missing title: %s", caption)
	}

	// 6. Test Dashboard display with lastReport
	dashText, dashMarkup := b.getMainDashboard()
	if !strings.Contains(dashText, "Последняя уборка") {
		t.Errorf("expected dashboard to contain 'Последняя уборка', got: %s", dashText)
	}

	foundBtn := false
	for _, row := range dashMarkup.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "view_last_report" {
				foundBtn = true
				break
			}
		}
	}
	if !foundBtn {
		t.Errorf("expected 'view_last_report' button in dashboard")
	}
}

func TestSessionFallbackElapsed(t *testing.T) {
	cfg := &config.Config{DefaultLang: "ru"}
	b := New(cfg, telegram.NewClient("token", "", 30), valetudo.NewClient("http://localhost:8080/api/v2/robot", time.Second))

	b.StartSession(nil, 100)
	// Simulate start time 5 minutes ago without peak stats update
	b.sessionMu.Lock()
	b.session.StartTime = time.Now().Add(-5*time.Minute - 10*time.Second)
	b.sessionMu.Unlock()

	report := b.FinishSession(90)
	if report.DurationMin < 5 {
		t.Errorf("expected at least 5 minutes from elapsed fallback, got %d", report.DurationMin)
	}
	if report.Rooms != "Вся квартира" {
		t.Errorf("expected 'Вся квартира' for empty rooms, got '%s'", report.Rooms)
	}
}

func TestFormatReportCaption_NilReport(t *testing.T) {
	cfg := &config.Config{DefaultLang: "ru"}
	b := New(cfg, telegram.NewClient("token", "", 30), valetudo.NewClient("http://localhost:8080/api/v2/robot", time.Second))

	caption := b.formatReportCaption(nil)
	if !strings.Contains(caption, "Уборка завершена!") {
		t.Errorf("expected caption to contain title, got: %s", caption)
	}
}

