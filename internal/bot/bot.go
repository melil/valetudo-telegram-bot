package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

type Bot struct {
	cfg *config.Config
	tg  *telegram.Client
	val *valetudo.Client

	capsMu sync.RWMutex
	caps   *valetudo.CapabilitySet

	statusMu   sync.RWMutex
	lastStatus string
	lastFlag   string

	langMu sync.RWMutex
	lang   i18n.Locale

	dashMu         sync.Mutex
	dashboardMsgID int

	wizardMu      sync.Mutex
	activeWizards map[int64]*WizardSession

	sessionMu  sync.RWMutex
	session    CleaningSession
	lastReport *CleaningReport
}

func New(cfg *config.Config, tg *telegram.Client, val *valetudo.Client) *Bot {
	return &Bot{
		cfg:           cfg,
		tg:            tg,
		val:           val,
		caps:          valetudo.NewCapabilitySet(nil),
		lastStatus:    "docked",
		lastFlag:      "none",
		lang:          i18n.NormalizeLocale(cfg.DefaultLang),
		activeWizards: make(map[int64]*WizardSession),
	}
}

func (b *Bot) GetRobotStatus() (string, string) {
	b.statusMu.RLock()
	s, f := b.lastStatus, b.lastFlag
	b.statusMu.RUnlock()
	if s != "" {
		return s, f
	}
	st, err := b.val.GetStatus()
	if err == nil {
		b.SetRobotStatus(st.Value, st.Flag)
		return st.Value, st.Flag
	}
	return "idle", "none"
}

func (b *Bot) SetRobotStatus(status, flag string) {
	b.statusMu.Lock()
	b.lastStatus = status
	b.lastFlag = flag
	b.statusMu.Unlock()
}

func (b *Bot) RefreshRobotStatus() (string, string) {
	st, err := b.val.GetStatus()
	if err == nil {
		b.SetRobotStatus(st.Value, st.Flag)
		return st.Value, st.Flag
	}
	return b.GetRobotStatus()
}

func (b *Bot) formatStatusDisplay(status, flag string) string {
	var icon, title string

	switch status {
	case "docked":
		icon = "🏠"
		title = b.t("statuses.docked")
	case "cleaning":
		switch flag {
		case "segment":
			icon = "🧹"
			title = b.t("statuses.cleaning_segment")
		case "zone":
			icon = "🧹"
			title = b.t("statuses.cleaning_zone")
		case "spot":
			icon = "🎯"
			title = b.t("statuses.cleaning_spot")
		case "mapping":
			icon = "🗺"
			title = b.t("statuses.cleaning_mapping")
		default:
			icon = "🧹"
			title = b.t("statuses.cleaning")
		}
	case "paused":
		icon = "⏸"
		title = b.t("statuses.paused")
	case "returning":
		icon = "🏠"
		title = b.t("statuses.returning")
	case "idle":
		icon = "💤"
		title = b.t("statuses.idle")
	case "moving":
		icon = "🚗"
		title = b.t("statuses.moving")
	case "manual_control":
		icon = "🎮"
		title = b.t("statuses.manual_control")
	case "error":
		icon = "🚨"
		title = b.t("statuses.error")
		if flag != "" && flag != "none" {
			title += " (" + flag + ")"
		}
	default:
		icon = "🤖"
		title = status
		if title == "" {
			title = b.t("statuses.unknown")
		}
	}

	return fmt.Sprintf("%s %s", icon, title)
}

func (b *Bot) Caps() *valetudo.CapabilitySet {
	b.capsMu.RLock()
	defer b.capsMu.RUnlock()
	return b.caps
}

func (b *Bot) SetCaps(cs *valetudo.CapabilitySet) {
	b.capsMu.Lock()
	defer b.capsMu.Unlock()
	b.caps = cs
}

func (b *Bot) LoadCapabilities() error {
	rawCaps, err := b.val.GetCapabilities()
	if err != nil {
		return err
	}
	b.SetCaps(valetudo.NewCapabilitySet(rawCaps))
	log.Printf("Загружено возможностей робота: %d (%v)", len(rawCaps), rawCaps)
	return nil
}

func (b *Bot) GetLang() i18n.Locale {
	b.langMu.RLock()
	defer b.langMu.RUnlock()
	return b.lang
}

func (b *Bot) SetLang(loc i18n.Locale) {
	b.langMu.Lock()
	defer b.langMu.Unlock()
	b.lang = loc
}

func (b *Bot) t(key string, args ...any) string {
	return i18n.T(b.GetLang(), key, args...)
}

func (b *Bot) GetDashboardMsgID() int {
	b.dashMu.Lock()
	defer b.dashMu.Unlock()
	return b.dashboardMsgID
}

func (b *Bot) SetDashboardMsgID(msgID int) {
	b.dashMu.Lock()
	defer b.dashMu.Unlock()
	b.dashboardMsgID = msgID
}

func (b *Bot) renderDashboard(text string, markup *telegram.InlineKeyboardMarkup) error {
	b.dashMu.Lock()
	msgID := b.dashboardMsgID
	b.dashMu.Unlock()

	if msgID != 0 {
		err := b.tg.EditMessage(b.cfg.AllowedChatID, msgID, text, markup)
		if err == nil {
			return nil
		}
		log.Printf("renderDashboard: не удалось обновить сообщение %d (%v), создаю новое...", msgID, err)
	}

	newID, err := b.tg.SendPayload(telegram.SendMessagePayload{
		ChatID:              b.cfg.AllowedChatID,
		Text:                text,
		ParseMode:           "HTML",
		ReplyMarkup:         markup,
		DisableNotification: b.cfg.IsDNDActive(),
	})
	if err == nil && newID != 0 {
		b.dashMu.Lock()
		b.dashboardMsgID = newID
		b.dashMu.Unlock()
	}
	return err
}

func (b *Bot) Run(ctx context.Context) error {
	log.Printf("Бот запущен. Слушаю сообщения для ChatID: %d\n", b.cfg.AllowedChatID)

	if err := b.LoadCapabilities(); err != nil {
		log.Printf("Внимание: не удалось загрузить возможности робота: %v", err)
	}

	// Запуск фонового мониторинга
	go b.statusWatcher(ctx)

	offset := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("Остановка цикла long polling...")
			return nil
		default:
		}

		updates, err := b.tg.GetUpdates(offset)
		if err != nil {
			log.Printf("Ошибка poll-запроса: %v. Повтор через 3с...", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.Message != nil && update.Message.Chat.ID == b.cfg.AllowedChatID {
				b.handleTextCommand(update.Message)
			}

			if update.CallbackQuery != nil && update.CallbackQuery.From.ID == b.cfg.AllowedChatID {
				b.handleCallback(update.CallbackQuery)
			}
		}
	}
}
