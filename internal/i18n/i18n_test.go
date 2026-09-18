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

	if ruStart != "🪄 Старт уборки" {
		t.Errorf("expected '🪄 Старт уборки', got '%s'", ruStart)
	}
	if enStart != "🪄 Start Cleaning" {
		t.Errorf("expected '🪄 Start Cleaning', got '%s'", enStart)
	}
	if deStart != "🪄 Reinigung starten" {
		t.Errorf("expected '🪄 Reinigung starten', got '%s'", deStart)
	}
	if zhStart != "🪄 开始清扫" {
		t.Errorf("expected '🪄 开始清扫', got '%s'", zhStart)
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
		{"🪄 Старт уборки", "main_menu.start_cleaning", true},
		{"🪄 Start Cleaning", "main_menu.start_cleaning", true},
		{"🪄 Reinigung starten", "main_menu.start_cleaning", true},
		{"🪄 开始清扫", "main_menu.start_cleaning", true},
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
