package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/database"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

type Bot struct {
	cfg *config.Config
	tg  *telegram.Client
	val *valetudo.Client
	db  *database.DB

	capsMu sync.RWMutex
	caps   *valetudo.CapabilitySet

	statusMu   sync.RWMutex
	lastStatus string
	lastFlag   string

	langMu sync.RWMutex
	lang   i18n.Locale

	dashMu         sync.Mutex
	dashboards     map[int64]int
	dashboardMsgID int
	activeChatID   int64

	wizardMu      sync.Mutex
	activeWizards map[int64]*WizardSession

	sessionMu  sync.RWMutex
	session    CleaningSession
	lastReport *CleaningReport

	authReqMu       sync.Mutex
	lastAuthReqTime map[int64]time.Time

	startTime time.Time
}

func New(cfg *config.Config, tg *telegram.Client, val *valetudo.Client, db ...*database.DB) *Bot {
	var userDB *database.DB
	if len(db) > 0 {
		userDB = db[0]
	}

	initialDashboards := make(map[int64]int)
	if userDB != nil {
		if savedMap, err := userDB.GetAllDashboardMsgIDs(); err == nil {
			initialDashboards = savedMap
		}
	}

	return &Bot{
		cfg:             cfg,
		tg:              tg,
		val:             val,
		db:              userDB,
		caps:            valetudo.NewCapabilitySet(nil),
		lastStatus:      "docked",
		lastFlag:        "none",
		lang:            i18n.NormalizeLocale(cfg.DefaultLang),
		dashboards:      initialDashboards,
		activeWizards:   make(map[int64]*WizardSession),
		lastAuthReqTime: make(map[int64]time.Time),
		startTime:       time.Now(),
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
	norm := i18n.NormalizeLocale(string(loc))
	b.langMu.Lock()
	b.lang = norm
	b.langMu.Unlock()
	if b.db != nil && b.cfg.AllowedChatID != 0 {
		_ = b.db.SetUserLocale(b.cfg.AllowedChatID, string(norm))
	}
}

func (b *Bot) GetUserLang(chatID int64) i18n.Locale {
	if b.db != nil && chatID != 0 {
		if u, err := b.db.GetUser(chatID); err == nil && u != nil && u.Locale != "" {
			return i18n.NormalizeLocale(u.Locale)
		}
	}
	return b.GetLang()
}

func (b *Bot) t(key string, args ...any) string {
	return i18n.T(b.GetUserLang(b.GetActiveChatID()), key, args...)
}

func (b *Bot) tUser(chatID int64, key string, args ...any) string {
	return i18n.T(b.GetUserLang(chatID), key, args...)
}

func (b *Bot) LogAction(chatID int64, action, details string) {
	if b.db == nil {
		return
	}
	var username string
	if u, err := b.db.GetUser(chatID); err == nil && u != nil {
		username = u.Username
	}
	_ = b.db.LogAction(chatID, username, action, details)
}

func (b *Bot) SetActiveChatID(chatID int64) {
	b.dashMu.Lock()
	defer b.dashMu.Unlock()
	b.activeChatID = chatID
}

func (b *Bot) GetActiveChatID() int64 {
	b.dashMu.Lock()
	defer b.dashMu.Unlock()
	if b.activeChatID != 0 {
		return b.activeChatID
	}
	return b.cfg.AllowedChatID
}

func (b *Bot) GetDashboardMsgID(chatID ...int64) int {
	b.dashMu.Lock()
	defer b.dashMu.Unlock()
	targetID := b.cfg.AllowedChatID
	if len(chatID) > 0 && chatID[0] != 0 {
		targetID = chatID[0]
	} else if b.activeChatID != 0 {
		targetID = b.activeChatID
	}
	if id, ok := b.dashboards[targetID]; ok && id != 0 {
		return id
	}
	return b.dashboardMsgID
}

func (b *Bot) SetDashboardMsgID(msgID int) {
	b.SetDashboardMsgIDForChat(b.GetActiveChatID(), msgID)
}

func (b *Bot) SetDashboardMsgIDForChat(chatID int64, msgID int) {
	b.dashMu.Lock()
	defer b.dashMu.Unlock()
	b.dashboards[chatID] = msgID
	b.dashboardMsgID = msgID

	if b.db != nil && chatID != 0 {
		_ = b.db.SetUserDashboardMsgID(chatID, msgID)
	}
}

func (b *Bot) renderDashboard(text string, markup *telegram.InlineKeyboardMarkup) error {
	return b.renderDashboardForChat(b.GetActiveChatID(), text, markup)
}

func (b *Bot) renderDashboardForChat(chatID int64, text string, markup *telegram.InlineKeyboardMarkup) error {
	if chatID == 0 {
		chatID = b.cfg.AllowedChatID
	}

	b.dashMu.Lock()
	msgID := b.dashboards[chatID]
	if msgID == 0 && chatID == b.cfg.AllowedChatID {
		msgID = b.dashboardMsgID
	}
	b.dashMu.Unlock()

	if msgID != 0 {
		err := b.tg.EditMessage(chatID, msgID, text, markup)
		if err == nil {
			return nil
		}
		log.Printf("renderDashboard: не удалось обновить сообщение %d в чате %d (%v), создаю новое...", msgID, chatID, err)
	}

	newID, err := b.tg.SendPayload(telegram.SendMessagePayload{
		ChatID:              chatID,
		Text:                text,
		ParseMode:           "HTML",
		ReplyMarkup:         markup,
		DisableNotification: b.cfg.IsDNDActive(),
	})
	if err == nil && newID != 0 {
		b.dashMu.Lock()
		b.dashboards[chatID] = newID
		if chatID == b.cfg.AllowedChatID {
			b.dashboardMsgID = newID
		}
		b.dashMu.Unlock()

		if b.db != nil && chatID != 0 {
			_ = b.db.SetUserDashboardMsgID(chatID, newID)
		}
	}
	return err
}

func (b *Bot) Run(ctx context.Context) error {
	log.Printf("Бот запущен. Слушаю входящие обновления Telegram...\n")

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

			if update.Message != nil {
				if b.isUserAllowed(update.Message.Chat.ID) {
					b.handleTextCommand(update.Message)
				} else {
					b.handleUnauthorizedAccess(update.Message)
				}
			}

			if update.CallbackQuery != nil {
				// 1. Проверяем действия авторизации администратором (кнопки Разрешить / Отклонить)
				if b.handleAuthCallback(update.CallbackQuery) {
					continue
				}

				// 2. Для остальных кнопок проверяем, разрешен ли доступ пользователю
				if b.isUserAllowed(update.CallbackQuery.From.ID) {
					b.handleCallback(update.CallbackQuery)
				} else {
					_ = b.tg.AnswerCallbackQueryAlert(update.CallbackQuery.ID, "⛔ У вас нет доступа.", true)
				}
			}
		}
	}
}
