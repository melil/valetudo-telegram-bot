package i18n

import (
	"testing"
)

func TestI18nLoading(t *testing.T) {
	bundle := GetBundle()
	locales := SupportedLocales()
	if len(locales) != 4 {
		t.Fatalf("expected 4 supported locales, got %d", len(locales))
	}

	ruMap := bundle.translations[LocaleRU]
	if len(ruMap) == 0 {
		t.Fatalf("ru translations map is empty")
	}

	for _, loc := range locales {
		locMap, ok := bundle.translations[loc]
		if !ok || len(locMap) == 0 {
			t.Fatalf("locale %s translations map is empty or missing", loc)
		}

		// Verify every RU key exists in this locale
		for k := range ruMap {
			if _, exists := locMap[k]; !exists {
				t.Errorf("missing key in %s.json: %s", loc, k)
			}
		}

		// Verify every key in this locale exists in RU
		for k := range locMap {
			if _, exists := ruMap[k]; !exists {
				t.Errorf("extra key in %s.json not present in ru.json: %s", loc, k)
			}
		}
	}
}

func TestTranslationFormatting(t *testing.T) {
	// Test basic string
	ruStart := T(LocaleRU, "main_menu.start_cleaning")
	enStart := T(LocaleEN, "main_menu.start_cleaning")
	deStart := T(LocaleDE, "main_menu.start_cleaning")
	zhStart := T(LocaleZH, "main_menu.start_cleaning")

	if ruStart != "🪄 Мастер уборки" {
		t.Errorf("expected '🪄 Мастер уборки', got '%s'", ruStart)
	}
	if enStart != "🪄 Cleaning Wizard" {
		t.Errorf("expected '🪄 Cleaning Wizard', got '%s'", enStart)
	}
	if deStart != "🪄 Reinigungsassistent" {
		t.Errorf("expected '🪄 Reinigungsassistent', got '%s'", deStart)
	}
	if zhStart != "🪄 清扫向导" {
		t.Errorf("expected '🪄 清扫向导', got '%s'", zhStart)
	}

	ruQuick := T(LocaleRU, "main_menu.quick_clean")
	enQuick := T(LocaleEN, "main_menu.quick_clean")
	if ruQuick != "▶️ Быстрая уборка" {
		t.Errorf("expected '▶️ Быстрая уборка', got '%s'", ruQuick)
	}
	if enQuick != "▶️ Quick Clean" {
		t.Errorf("expected '▶️ Quick Clean', got '%s'", enQuick)
	}

	// Test formatted string
	ruFmt := T(LocaleRU, "wizard.started_title", "Только сухая", "Кухня", 2)
	enFmt := T(LocaleEN, "wizard.started_title", "Vacuum only", "Kitchen", 2)
	deFmt := T(LocaleDE, "wizard.started_title", "Nur Saugen", "Küche", 2)
	zhFmt := T(LocaleZH, "wizard.started_title", "仅扫地", "厨房", 2)

	expectedRU := "🚀 <b>Уборка запущена!</b>\n\n• <b>Режим:</b> <code>Только сухая</code>\n• <b>Комнаты:</b> Кухня\n• <b>Проходов:</b> 2"
	expectedEN := "🚀 <b>Cleaning Started!</b>\n\n• <b>Mode:</b> <code>Vacuum only</code>\n• <b>Rooms:</b> Kitchen\n• <b>Passes:</b> 2"
	expectedDE := "🚀 <b>Reinigung gestartet!</b>\n\n• <b>Modus:</b> <code>Nur Saugen</code>\n• <b>Räume:</b> Küche\n• <b>Durchgänge:</b> 2"
	expectedZH := "🚀 <b>清扫已启动！</b>\n\n• <b>模式：</b> <code>仅扫地</code>\n• <b>房间：</b> 厨房\n• <b>遍数：</b> 2"

	if ruFmt != expectedRU {
		t.Errorf("RU formatting mismatch:\nGot: %s\nExp: %s", ruFmt, expectedRU)
	}
	if enFmt != expectedEN {
		t.Errorf("EN formatting mismatch:\nGot: %s\nExp: %s", enFmt, expectedEN)
	}
	if deFmt != expectedDE {
		t.Errorf("DE formatting mismatch:\nGot: %s\nExp: %s", deFmt, expectedDE)
	}
	if zhFmt != expectedZH {
		t.Errorf("ZH formatting mismatch:\nGot: %s\nExp: %s", zhFmt, expectedZH)
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		text string
		key  string
		want bool
	}{
		{"🪄 Мастер уборки", "main_menu.start_cleaning", true},
		{"🪄 Cleaning Wizard", "main_menu.start_cleaning", true},
		{"🪄 Reinigungsassistent", "main_menu.start_cleaning", true},
		{"🪄 清扫向导", "main_menu.start_cleaning", true},
		{"▶️ Быстрая уборка", "main_menu.quick_clean", true},
		{"▶️ Quick Clean", "main_menu.quick_clean", true},
		{"Something Else", "main_menu.start_cleaning", false},
		{"", "main_menu.start_cleaning", false},
	}

	for _, tc := range cases {
		got := Matches(tc.text, tc.key)
		if got != tc.want {
			t.Errorf("Matches(%q, %q) = %v; want %v", tc.text, tc.key, got, tc.want)
		}
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		input string
		want  Locale
	}{
		{"ru", LocaleRU},
		{"RUS", LocaleRU},
		{"russian", LocaleRU},
		{"en", LocaleEN},
		{"eng", LocaleEN},
		{"ENGLISH", LocaleEN},
		{"de", LocaleDE},
		{"ger", LocaleDE},
		{"german", LocaleDE},
		{"DEUTSCH", LocaleDE},
		{"zh", LocaleZH},
		{"cn", LocaleZH},
		{"chinese", LocaleZH},
		{"zh-cn", LocaleZH},
		{"zh-hans", LocaleZH},
		{"unknown", LocaleRU}, // fallback to default
	}

	for _, tt := range tests {
		got := NormalizeLocale(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeLocale(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFallback(t *testing.T) {
	// Non-existent key should return key itself
	unknown := T(LocaleEN, "non_existent_key_12345")
	if unknown != "non_existent_key_12345" {
		t.Errorf("expected key fallback, got '%s'", unknown)
	}
}

func TestTranslateRobotError(t *testing.T) {
	// 1. Wheel lost floor contact (the user's exact case when lifting robot)
	ruLift := TranslateRobotError(LocaleRU, "Wheel lost floor contact")
	if ruLift != "Колеса потеряли контакт с полом (робот поднят)" {
		t.Errorf("expected Russian lifted message, got %q", ruLift)
	}
	enLift := TranslateRobotError(LocaleEN, "Wheel lost floor contact")
	if enLift != "Wheel lost floor contact" {
		t.Errorf("expected English lifted message, got %q", enLift)
	}
	deLift := TranslateRobotError(LocaleDE, "Wheel lost floor contact")
	if deLift != "Rad hat Bodenkontakt verloren (Roboter angehoben)" {
		t.Errorf("expected German lifted message, got %q", deLift)
	}
	zhLift := TranslateRobotError(LocaleZH, "Wheel lost floor contact")
	if zhLift != "轮子悬空（机器人被抬起）" {
		t.Errorf("expected Chinese lifted message, got %q", zhLift)
	}

	// 2. Viomi lifted wheels message
	ruViomi := TranslateRobotError(LocaleRU, "Wheels suspended - place the vacuum on a flat surface")
	if ruViomi != "Колеса зависли в воздухе — опустите робота на ровную поверхность" {
		t.Errorf("expected Viomi Russian message, got %q", ruViomi)
	}

	// 3. Dynamic error messages (Internal error / Unknown error)
	ruUnknown := TranslateRobotError(LocaleRU, "Unknown error 42")
	if ruUnknown != "Неизвестная ошибка 42" {
		t.Errorf("expected dynamic unknown error in RU, got %q", ruUnknown)
	}
	ruInternal := TranslateRobotError(LocaleRU, "Internal error AVA_TEST")
	if ruInternal != "Внутренняя ошибка AVA_TEST" {
		t.Errorf("expected dynamic internal error in RU, got %q", ruInternal)
	}

	// 4. Empty / none fallback
	ruEmpty := TranslateRobotError(LocaleRU, "")
	if ruEmpty != "Неизвестная ошибка" {
		t.Errorf("expected unknown error for empty string, got %q", ruEmpty)
	}
	ruNone := TranslateRobotError(LocaleRU, "none")
	if ruNone != "Неизвестная ошибка" {
		t.Errorf("expected unknown error for 'none', got %q", ruNone)
	}

	// 5. Unmapped custom error returned as-is
	custom := TranslateRobotError(LocaleRU, "Custom Hardware Fault")
	if custom != "Custom Hardware Fault" {
		t.Errorf("expected raw message for unmapped error, got %q", custom)
	}
}

