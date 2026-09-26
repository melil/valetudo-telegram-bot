package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func TestMainMenuMapping_Full(t *testing.T) {
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

	markup := b.getMainMenuMarkup()
	if len(markup.Keyboard) != 3 {
		t.Fatalf("expected 3 rows in full main menu, got %d", len(markup.Keyboard))
	}

	// Row 0: Start Cleaning, Stop Cleaning
	if len(markup.Keyboard[0]) != 2 || markup.Keyboard[0][0] != b.t("main_menu.start_cleaning") {
		t.Errorf("expected row 0 to contain start_cleaning, got %+v", markup.Keyboard[0])
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

	markup := b.getMainMenuMarkup()
	if len(markup.Keyboard) != 2 {
		t.Fatalf("expected 2 rows in minimal main menu, got %d", len(markup.Keyboard))
	}

	// Row 0: Full Clean (replacing Start Cleaning), Stop Cleaning
	if markup.Keyboard[0][0] != b.t("main_menu.full_clean") {
		t.Errorf("expected start button to be replaced with full_clean, got %q", markup.Keyboard[0][0])
	}

	// Row 1: Robot only (Station is hidden)
	if len(markup.Keyboard[1]) != 1 || markup.Keyboard[1][0] != b.t("main_menu.robot") {
		t.Errorf("expected row 1 to contain only robot, got %+v", markup.Keyboard[1])
	}
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

	_, markup := b.getSettingsMainMenu()
	// Should contain: btn_fan, btn_lang, btn_back (3 rows)
	if len(markup.InlineKeyboard) != 3 {
		t.Fatalf("expected 3 items in settings menu, got %d", len(markup.InlineKeyboard))
	}

	callbacks := []string{}
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			callbacks = append(callbacks, btn.CallbackData)
		}
	}

	expected := []string{"sub_fan", "sub_lang", "menu_robot"}
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
	b.handleTextCommand("/rooms")
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /rooms, got %q", sentText)
	}

	// 2. /locate
	sentText = ""
	b.handleTextCommand("/locate")
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /locate, got %q", sentText)
	}

	// 3. /station
	sentText = ""
	b.handleTextCommand("/station")
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /station, got %q", sentText)
	}

	// 4. /wizard
	sentText = ""
	b.handleTextCommand("/wizard")
	if sentText != expectedErrMsg {
		t.Errorf("expected not_supported message for /wizard, got %q", sentText)
	}
}
