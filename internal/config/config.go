package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ValetudoBaseURL string
	BotToken        string
	AllowedChatID   int64
	TgAPIBase       string
	PollTimeoutSec  int
	WatcherInterval time.Duration

	DNDEnabled   bool
	DNDStartHour int
	DNDEndHour   int

	RoomAliases map[string]string
}

func Load() (*Config, error) {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	chatIDStr := os.Getenv("CHAT_ID")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || chatID == 0 {
		return nil, fmt.Errorf("valid numeric CHAT_ID is required")
	}

	valetudoURL := os.Getenv("VALETUDO_BASE_URL")
	if valetudoURL == "" {
		valetudoURL = "http://127.0.0.1/api/v2/robot"
	}
	valetudoURL = strings.TrimRight(valetudoURL, "/")

	tgBase := os.Getenv("TG_API_BASE")
	if tgBase == "" {
		tgBase = "https://api.telegram.org"
	}
	tgBase = strings.TrimRight(tgBase, "/")

	cfg := &Config{
		ValetudoBaseURL: valetudoURL,
		BotToken:        token,
		AllowedChatID:   chatID,
		TgAPIBase:       tgBase,
		PollTimeoutSec:  25,
		WatcherInterval: 12 * time.Second,
		DNDEnabled:      true,
		DNDStartHour:    23,
		DNDEndHour:      8,
		RoomAliases: map[string]string{
			"1": "Кухня",
			"2": "Диван",
			"3": "Коридор",
			"4": "Воркспейс",
		},
	}

	if dnd := os.Getenv("DND_ENABLED"); dnd != "" {
		cfg.DNDEnabled = (dnd == "true" || dnd == "1")
	}
	if sh := os.Getenv("DND_START_HOUR"); sh != "" {
		if h, err := strconv.Atoi(sh); err == nil && h >= 0 && h <= 23 {
			cfg.DNDStartHour = h
		}
	}
	if eh := os.Getenv("DND_END_HOUR"); eh != "" {
		if h, err := strconv.Atoi(eh); err == nil && h >= 0 && h <= 23 {
			cfg.DNDEndHour = h
		}
	}

	return cfg, nil
}

func (c *Config) IsDNDActive() bool {
	if !c.DNDEnabled {
		return false
	}
	h := time.Now().Hour()
	if c.DNDStartHour > c.DNDEndHour {
		return h >= c.DNDStartHour || h < c.DNDEndHour
	}
	return h >= c.DNDStartHour && h < c.DNDEndHour
}
