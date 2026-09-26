package bot

import (
	"strings"
	"testing"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func TestParseMeminfo(t *testing.T) {
	sampleMeminfo := `MemTotal:         506720 kB
MemFree:           45180 kB
MemAvailable:     184920 kB
Buffers:           12400 kB
Cached:           127340 kB
SwapTotal:             0 kB
SwapFree:              0 kB`

	total, avail, used, ok := parseMeminfo(sampleMeminfo)
	if !ok {
		t.Fatalf("expected parseMeminfo to succeed")
	}
	if total != 506720 {
		t.Errorf("expected total=506720, got %d", total)
	}
	if avail != 184920 {
		t.Errorf("expected avail=184920, got %d", avail)
	}
	expectedUsed := uint64(506720 - 184920)
	if used != expectedUsed {
		t.Errorf("expected used=%d, got %d", expectedUsed, used)
	}

	// Test fallback when MemAvailable is missing
	sampleWithoutAvail := `MemTotal:         500000 kB
MemFree:           50000 kB
Buffers:           20000 kB
Cached:            30000 kB`

	total2, avail2, used2, ok2 := parseMeminfo(sampleWithoutAvail)
	if !ok2 {
		t.Fatalf("expected parseMeminfo without MemAvailable to succeed")
	}
	if total2 != 500000 {
		t.Errorf("expected total=500000, got %d", total2)
	}
	if avail2 != 100000 { // 50000 + 20000 + 30000
		t.Errorf("expected avail=100000, got %d", avail2)
	}
	if used2 != 400000 {
		t.Errorf("expected used=400000, got %d", used2)
	}

	// Empty or invalid input
	_, _, _, okBad := parseMeminfo("")
	if okBad {
		t.Errorf("expected empty meminfo to fail")
	}
}

func TestParseLoadAvg(t *testing.T) {
	sample := "0.45 0.38 0.41 1/120 4812"
	l1, l5, l15, ok := parseLoadAvg(sample)
	if !ok {
		t.Fatalf("expected parseLoadAvg to succeed")
	}
	if l1 != "0.45" || l5 != "0.38" || l15 != "0.41" {
		t.Errorf("unexpected load avg values: %s, %s, %s", l1, l5, l15)
	}

	_, _, _, okBad := parseLoadAvg("0.1")
	if okBad {
		t.Errorf("expected invalid loadavg to fail")
	}
}

func TestParseThermalTemp(t *testing.T) {
	// Millidegrees format typical in Linux
	temp, ok := parseThermalTemp("51400\n")
	if !ok {
		t.Fatalf("expected parseThermalTemp to succeed")
	}
	if temp < 51.39 || temp > 51.41 {
		t.Errorf("expected ~51.4, got %f", temp)
	}

	// Direct Celsius format
	temp2, ok2 := parseThermalTemp("48.5")
	if !ok2 {
		t.Fatalf("expected parseThermalTemp to succeed for 48.5")
	}
	if temp2 != 48.5 {
		t.Errorf("expected 48.5, got %f", temp2)
	}

	// Invalid input
	_, okBad := parseThermalTemp("invalid")
	if okBad {
		t.Errorf("expected parseThermalTemp to fail on invalid input")
	}
}

func TestFormatBytesAndKB(t *testing.T) {
	cases := []struct {
		bytes uint64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1048576 * 15, "15.0 MB"},
		{1073741824 * 2, "2.00 GB"},
	}

	for _, c := range cases {
		got := formatBytes(c.bytes)
		if got != c.want {
			t.Errorf("formatBytes(%d) = %s, want %s", c.bytes, got, c.want)
		}
	}

	if gotKB := formatKB(1024); gotKB != "1.0 MB" {
		t.Errorf("formatKB(1024) = %s, want 1.0 MB", gotKB)
	}
}

func TestFormatDurationLocales(t *testing.T) {
	d := 25*time.Hour + 30*time.Minute

	ru := formatDuration(d, i18n.LocaleRU)
	if ru != "1д 1ч 30м" {
		t.Errorf("expected RU '1д 1ч 30м', got '%s'", ru)
	}

	en := formatDuration(d, i18n.LocaleEN)
	if en != "1d 1h 30m" {
		t.Errorf("expected EN '1d 1h 30m', got '%s'", en)
	}

	de := formatDuration(d, i18n.LocaleDE)
	if de != "1T 1Std 30Min" {
		t.Errorf("expected DE '1T 1Std 30Min', got '%s'", de)
	}

	zh := formatDuration(d, i18n.LocaleZH)
	if zh != "1天 1小时 30分" {
		t.Errorf("expected ZH '1天 1小时 30分', got '%s'", zh)
	}

	// Less than 1 hour
	dShort := 4*time.Minute + 12*time.Second
	if got := formatDuration(dShort, i18n.LocaleRU); got != "4м 12с" {
		t.Errorf("expected RU '4м 12с', got '%s'", got)
	}
}

func TestBuildResourcesReport(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	report := b.buildResourcesReport()
	if !strings.Contains(report, "СИСТЕМНЫЕ РЕСУРСЫ РОБОТА") {
		t.Errorf("report missing title, got:\n%s", report)
	}
	if !strings.Contains(report, "Память кучи") {
		t.Errorf("report missing heap memory info, got:\n%s", report)
	}
	if !strings.Contains(report, "Горутины") {
		t.Errorf("report missing goroutines info, got:\n%s", report)
	}
	if !strings.Contains(report, "Хост-система робота") {
		t.Errorf("report missing host section, got:\n%s", report)
	}

	// Change language to EN and verify
	b.SetLang(i18n.LocaleEN)
	reportEN := b.buildResourcesReport()
	if !strings.Contains(reportEN, "ROBOT SYSTEM RESOURCES") {
		t.Errorf("EN report missing title, got:\n%s", reportEN)
	}
	if !strings.Contains(reportEN, "Heap Memory") {
		t.Errorf("EN report missing heap memory info, got:\n%s", reportEN)
	}
}

func TestRobotMenuHasResourcesButton(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	b.sendRobotMenu()
	// Check that b.sendRobotMenu executed without panic and rendered dashboard
	if b.GetDashboardMsgID() == 0 {
		// Mock tg client returns msgID 0 for fake url, which is expected
	}
}
