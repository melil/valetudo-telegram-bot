package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

type Locale string

const (
	LocaleRU Locale = "ru"
	LocaleEN Locale = "en"
	LocaleDE Locale = "de"
	LocaleZH Locale = "zh"
)

var defaultLocale = LocaleRU

type Bundle struct {
	mu           sync.RWMutex
	translations map[Locale]map[string]string
}

var globalBundle *Bundle
var once sync.Once

func GetBundle() *Bundle {
	once.Do(func() {
		globalBundle = newBundle()
	})
	return globalBundle
}

func newBundle() *Bundle {
	b := &Bundle{
		translations: make(map[Locale]map[string]string),
	}
	b.loadEmbedded()
	return b
}

func (b *Bundle) loadEmbedded() {
	files, err := localesFS.ReadDir("locales")
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		localeName := Locale(strings.TrimSuffix(f.Name(), ".json"))
		data, err := localesFS.ReadFile("locales/" + f.Name())
		if err != nil {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}

		flattened := make(map[string]string)
		flattenMap("", raw, flattened)

		b.translations[localeName] = flattened
	}
}

func flattenMap(prefix string, m map[string]any, out map[string]string) {
	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch val := v.(type) {
		case string:
			out[fullKey] = val
		case map[string]any:
			flattenMap(fullKey, val, out)
		}
	}
}

// T translates a key for a given locale with optional format arguments
func T(loc Locale, key string, args ...any) string {
	b := GetBundle()
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 1. Попытка получить в запрошенной локали
	var msg string
	var found bool

	if m, ok := b.translations[loc]; ok {
		msg, found = m[key]
	}

	// 2. Fallback к русской локали (или дефолтной)
	if !found && loc != defaultLocale {
		if m, ok := b.translations[defaultLocale]; ok {
			msg, found = m[key]
		}
	}

	// 3. Если ключ не найден вообще
	if !found {
		return key
	}

	// 4. Форматирование аргументов, если переданы
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// SupportedLocales возвращает список доступных локалей
func SupportedLocales() []Locale {
	return []Locale{LocaleRU, LocaleEN, LocaleDE, LocaleZH}
}

// LocaleName возвращает отображаемое имя для кнопок меню
func LocaleName(loc Locale) string {
	switch loc {
	case LocaleRU:
		return "🇷🇺 Русский"
	case LocaleEN:
		return "🇬🇧 English"
	case LocaleDE:
		return "🇩🇪 Deutsch"
	case LocaleZH:
		return "🇨🇳 简体中文"
	default:
		return string(loc)
	}
}

// NormalizeLocale приводит код к поддерживаемой локали
func NormalizeLocale(code string) Locale {
	lower := strings.ToLower(strings.TrimSpace(code))
	switch lower {
	case "en", "eng", "english":
		return LocaleEN
	case "de", "ger", "german", "deutsch":
		return LocaleDE
	case "zh", "cn", "chi", "chinese", "zh-cn", "zh-hans":
		return LocaleZH
	case "ru", "rus", "russian":
		return LocaleRU
	default:
		return defaultLocale
	}
}

// Matches проверяет, совпадает ли текст со значением ключа в любой поддерживаемой локали
func Matches(text, key string) bool {
	if text == "" || key == "" {
		return false
	}
	b := GetBundle()
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, loc := range SupportedLocales() {
		if m, ok := b.translations[loc]; ok {
			if m[key] == text {
				return true
			}
		}
	}
	return false
}
