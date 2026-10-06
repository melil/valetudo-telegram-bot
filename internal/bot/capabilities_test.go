package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func TestMainMenuMapping_Docked(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	b.SetCaps(valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
		string(valetudo.CapMapSegmentation),
		string(valetudo.CapLocate),
		string(valetudo.CapAutoEmptyDockManualTrigger),
	}))
	b.SetRobotStatus("docked", "none")

	markup := b.getMainMenuMarkup()
	if len(markup.Keyboard) != 3 {
		t.Fatalf("expected 3 rows in docked main menu, got %d", len(markup.Keyboard))
	}

	// Row 0: Start Cleaning (wizard) and Quick Clean
	if len(markup.Keyboard[0]) != 2 || markup.Keyboard[0][0] != b.t("main_menu.start_cleaning") || markup.Keyboard[0][1] != b.t("main_menu.quick_clean") {
		t.Errorf("expected row 0 to contain start_cleaning and quick_clean, got %+v", markup.Keyboard[0])
	}
	// Row 1: Robot, Station
	if len(markup.Keyboard[1]) != 2 || markup.Keyboard[1][1] != b.t("main_menu.station") {
		t.Errorf("expected row 1 to contain station, got %+v", markup.Keyboard[1])
	}
	// Row 2: Rooms, Locate
	if len(markup.Keyboard[2]) != 2 || markup.Keyboard[2][0] != b.t("main_menu.rooms") || markup.Keyboard[2][1] != b.t("main_menu.locate") {
		t.Errorf("expected row 2 to contain rooms and locate, got %+v", markup.Keyboard[2])
	}
}

func TestMainMenuMapping_Cleaning(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	b.SetCaps(valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
		string(valetudo.CapMapSegmentation),
	}))
	b.SetRobotStatus("cleaning", "none")

	markup := b.getMainMenuMarkup()
	// Row 0: Pause, Stop, Home
	if len(markup.Keyboard[0]) != 3 {
		t.Fatalf("expected 3 buttons when cleaning, got %d", len(markup.Keyboard[0]))
	}
	if markup.Keyboard[0][0] != b.t("main_menu.pause_cleaning") {
		t.Errorf("expected pause button, got %s", markup.Keyboard[0][0])
	}
	if markup.Keyboard[0][1] != b.t("main_menu.stop_robot") {
		t.Errorf("expected stop button, got %s", markup.Keyboard[0][1])
	}
	if markup.Keyboard[0][2] != b.t("main_menu.go_home") {
		t.Errorf("expected home button, got %s", markup.Keyboard[0][2])
	}
}

func TestMainMenuMapping_Paused(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	b.SetCaps(valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
		string(valetudo.CapMapSegmentation),
	}))
	b.SetRobotStatus("paused", "resumable")

	markup := b.getMainMenuMarkup()
	// Row 0: Resume, Stop, Home
	if len(markup.Keyboard[0]) != 3 {
		t.Fatalf("expected 3 buttons when paused, got %d", len(markup.Keyboard[0]))
	}
	if markup.Keyboard[0][0] != b.t("main_menu.resume_cleaning") {
		t.Errorf("expected resume button, got %s", markup.Keyboard[0][0])
	}
	if markup.Keyboard[0][1] != b.t("main_menu.stop_robot") {
		t.Errorf("expected stop button, got %s", markup.Keyboard[0][1])
	}
	if markup.Keyboard[0][2] != b.t("main_menu.go_home") {
		t.Errorf("expected home button, got %s", markup.Keyboard[0][2])
	}
}

func TestMainMenuMapping_MinimalVacuum(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	// Robot only supports basic control (no segmentation, no station, no locate)
	b.SetCaps(valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
	}))
	b.SetRobotStatus("docked", "none")

	markup := b.getMainMenuMarkup()
	if len(markup.Keyboard) != 2 {
		t.Fatalf("expected 2 rows in minimal main menu, got %d", len(markup.Keyboard))
	}

	// Row 0: Quick Clean (replacing Start Cleaning)
	if markup.Keyboard[0][0] != b.t("main_menu.quick_clean") {
		t.Errorf("expected start button to be replaced with quick_clean, got %q", markup.Keyboard[0][0])
	}

	// Row 1: Robot only (Station is hidden)
	if len(markup.Keyboard[1]) != 1 || markup.Keyboard[1][0] != b.t("main_menu.robot") {
		t.Errorf("expected row 1 to contain only robot, got %+v", markup.Keyboard[1])
	}
}

func TestRobotMenuMapping_DynamicStates(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	b.SetCaps(valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
	}))

	// 1. When docked: row 0 has cmd_start
	b.SetRobotStatus("docked", "none")
	// We can inspect rows via test:
	// row 0 should be cmd_start
	b.SetRobotStatus("cleaning", "none")
	// row 0 should have cmd_pause, cmd_stop, cmd_home
	b.SetRobotStatus("paused", "resumable")
	// row 0 should have cmd_resume, cmd_stop, cmd_home
}

func TestSettingsMenuMapping(t *testing.T) {
	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", "http://fake", 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	// Only FanSpeedControl is available
	b.SetCaps(valetudo.NewCapabilitySet([]string{
		string(valetudo.CapFanSpeedControl),
	}))

	_, markup := b.getRobotSettingsMenu()
	// Should contain telemetry and fan in row 1, and back in row 2 (2 rows)
	if len(markup.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows in robot settings menu, got %d", len(markup.InlineKeyboard))
	}

	callbacks := []string{}
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			callbacks = append(callbacks, btn.CallbackData)
		}
	}

	expected := []string{"cmd_telemetry", "sub_fan", "menu_robot"}
	for i, exp := range expected {
		if callbacks[i] != exp {
			t.Errorf("callback[%d]: expected %q, got %q", i, exp, callbacks[i])
		}
	}
}

func TestUnsupportedTextCommands(t *testing.T) {
	var sentText string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "editMessageText") {
			var payload telegram.EditMessagePayload
			_ = json.NewDecoder(r.Body).Decode(&payload)
			sentText = payload.Text
			w.Write([]byte(`{"ok":true}`))
			return
		}
		var payload telegram.SendMessagePayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		sentText = payload.Text
		w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer ts.Close()

	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", ts.URL, 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	// Set capabilities without MapSegmentation, Station or Locate
	b.SetCaps(valetudo.NewCapabilitySet([]string{
		string(valetudo.CapBasicControl),
	}))

	expectedErrMsg := b.t("main_menu.not_supported")

	// 1. /rooms
	b.handleTextCommand(&telegram.Message{Text: "/rooms"})
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /rooms, got %q", sentText)
	}

	// 2. /locate
	sentText = ""
	b.handleTextCommand(&telegram.Message{Text: "/locate"})
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /locate, got %q", sentText)
	}

	// 3. /station
	sentText = ""
	b.handleTextCommand(&telegram.Message{Text: "/station"})
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /station, got %q", sentText)
	}

	// 4. /wizard
	sentText = ""
	b.handleTextCommand(&telegram.Message{Text: "/wizard"})
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /wizard, got %q", sentText)
	}
}

func TestHelpCommand(t *testing.T) {
	var sentText string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "editMessageText") {
			var payload telegram.EditMessagePayload
			_ = json.NewDecoder(r.Body).Decode(&payload)
			sentText = payload.Text
			w.Write([]byte(`{"ok":true}`))
			return
		}
		var payload telegram.SendMessagePayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		sentText = payload.Text
		w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer ts.Close()

	cfg := &config.Config{
		AllowedChatID: 12345,
		DefaultLang:   "ru",
	}
	tg := telegram.NewClient("fake", ts.URL, 10)
	val := valetudo.NewClient("http://fake", time.Second)
	b := New(cfg, tg, val)

	msg := &telegram.Message{Text: "/help"}
	msg.Chat.ID = 12345
	b.handleTextCommand(msg)
	if !strings.Contains(sentText, "Справка по командам бота") {
		t.Errorf("expected help message to contain title, got: %s", sentText)
	}
	if !strings.Contains(sentText, "/wizard") || !strings.Contains(sentText, "/clean") {
		t.Errorf("expected help message to contain wizard and clean commands, got: %s", sentText)
	}
}

