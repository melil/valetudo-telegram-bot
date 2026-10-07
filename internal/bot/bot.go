package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	delivery "tgbot/internal/bot/delivery/telegram"
	"tgbot/internal/bot/domain"
	"tgbot/internal/bot/service/auth"
	"tgbot/internal/bot/service/cleaning"
	"tgbot/internal/bot/service/consumables"
	"tgbot/internal/bot/service/session"
	"tgbot/internal/bot/service/system"
	"tgbot/internal/bot/service/update"
	"tgbot/internal/bot/service/watcher"
	"tgbot/internal/config"
	"tgbot/internal/database"
	"tgbot/internal/i18n"
	"tgbot/internal/netutil"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

// Re-export domain models as type aliases for seamless compatibility.
type CleaningReport = domain.CleaningReport
type CleaningSession = domain.CleaningSession
type ConsumableDisplayInfo = domain.ConsumableDisplayInfo
type RoomInfo = domain.RoomInfo
type WizardSession = domain.WizardSession
type RuntimeStats = domain.RuntimeStats
type HostStats = domain.HostStats

// Bot acts as the composition root and orchestrator.
type Bot struct {
	cfg *config.Config
	tg  *telegram.Client
	val *valetudo.Client
	db  *database.DB

	// Services
	authSvc        *auth.Service
	cleaningSvc    *cleaning.WizardService
	consumablesSvc *consumables.Service
	sessionSvc     *session.Service
	systemSvc      *system.Service
	watcherSvc     *watcher.Service
	updateSvc      *update.Service
	handler        *delivery.Handler

	capsMu sync.RWMutex
	caps   *valetudo.CapabilitySet

	statusMu   sync.RWMutex
	lastStatus string
	lastFlag   string
	lastError  *valetudo.RobotError

	langMu sync.RWMutex
	lang   i18n.Locale

	dashMu         sync.Mutex
	dashboards     map[int64]int
	dashboardMsgID int
	activeChatID   int64

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

	b := &Bot{
		cfg:        cfg,
		tg:         tg,
		val:        val,
		db:         userDB,
		caps:       valetudo.NewCapabilitySet(nil),
		lastStatus: "docked",
		lastFlag:   "none",
		lang:       i18n.NormalizeLocale(cfg.DefaultLang),
		dashboards: initialDashboards,
		startTime:  time.Now(),
	}

	var uRepo domain.UserRepository
	if userDB != nil {
		uRepo = userDB
	}

	// Initialize application services (SOLID Dependency Inversion)
	b.authSvc = auth.NewService(cfg.AllowedChatID, uRepo, tg, b.GetUserLang)
	b.cleaningSvc = cleaning.NewWizardService(val, cfg.RoomAliases)
	b.consumablesSvc = consumables.NewService(val)
	b.sessionSvc = session.NewService(val)
	b.systemSvc = system.NewService()
	b.updateSvc = update.NewService(
		update.Config{
			Repo:          cfg.GitHubRepo,
			GitHubToken:   cfg.GitHubToken,
			CheckInterval: cfg.UpdateCheckInterval,
			AutoNotify:    cfg.AutoUpdateNotify,
			AllowedChatID: cfg.AllowedChatID,
		},
		uRepo,
		tg,
		b.authSvc,
		b.GetUserLang,
	)

	b.handler = delivery.NewHandler(
		tg,
		val,
		uRepo,
		b.authSvc,
		b.cleaningSvc,
		b.consumablesSvc,
		b.sessionSvc,
		b.systemSvc,
		b.updateSvc,
		b,
	)

	b.watcherSvc = watcher.NewService(
		val,
		tg,
		b.authSvc,
		b.sessionSvc,
		watcher.Config{
			Interval:    cfg.WatcherInterval,
			IsDNDActive: cfg.IsDNDActive,
			OnStatusUpdate: func(status, flag string) {
				b.SetRobotStatus(status, flag)
			},
			OnErrorUpdate: func(rErr *valetudo.RobotError) {
				b.SetRobotError(rErr)
			},
			OnLoadCapabilities: func() error {
				if len(b.Caps().List()) == 0 {
					return b.LoadCapabilities()
				}
				return nil
			},
			OnBroadcastDash: func() {
				b.broadcastMainDashboard()
			},
			FormatCaption: func(report *domain.CleaningReport, chatID int64) string {
				return b.formatReportCaptionForChat(report, chatID)
			},
			TranslateUser: func(chatID int64, key string, args ...any) string {
				return b.tUser(chatID, key, args...)
			},
			FormatErrorNotification: func(chatID int64, status string, rErr *valetudo.RobotError) string {
				return delivery.FormatErrorNotification(status, rErr, b.GetUserLang(chatID))
			},
		},
	)

	return b
}

func (b *Bot) GetCaps() *valetudo.CapabilitySet {
	return b.Caps()
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

func (b *Bot) GetRobotStatus() (string, string) {
	b.statusMu.RLock()
	s, f := b.lastStatus, b.lastFlag
	b.statusMu.RUnlock()
	if s != "" {
		return s, f
	}
	st, err := b.val.GetStatus()
	if err == nil {
		b.statusMu.Lock()
		b.lastStatus = st.Value
		b.lastFlag = st.Flag
		b.lastError = st.Error
		b.statusMu.Unlock()
		return st.Value, st.Flag
	}
	return "idle", "none"
}

func (b *Bot) GetRobotError() *valetudo.RobotError {
	b.statusMu.RLock()
	defer b.statusMu.RUnlock()
	return b.lastError
}

func (b *Bot) SetRobotStatus(status, flag string) {
	b.statusMu.Lock()
	b.lastStatus = status
	b.lastFlag = flag
	if status != "error" {
		b.lastError = nil
	}
	b.statusMu.Unlock()
}

func (b *Bot) SetRobotError(rErr *valetudo.RobotError) {
	b.statusMu.Lock()
	b.lastError = rErr
	b.statusMu.Unlock()
}

func (b *Bot) RefreshRobotStatus() (string, string) {
	st, err := b.val.GetStatus()
	if err == nil {
		b.statusMu.Lock()
		b.lastStatus = st.Value
		b.lastFlag = st.Flag
		b.lastError = st.Error
		b.statusMu.Unlock()
		return st.Value, st.Flag
	}
	return b.GetRobotStatus()
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

func (b *Bot) SetUserLang(chatID int64, loc i18n.Locale) {
	norm := i18n.NormalizeLocale(string(loc))
	if chatID == b.cfg.AllowedChatID {
		b.SetLang(norm)
	}
	if b.db != nil && chatID != 0 {
		_ = b.db.SetUserLocale(chatID, string(norm))
	}
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
	if b.dashboards[chatID] == msgID {
		return
	}
	b.dashboards[chatID] = msgID
	if chatID == b.cfg.AllowedChatID {
		b.dashboardMsgID = msgID
	}

	if b.db != nil && chatID != 0 {
		_ = b.db.SetUserDashboardMsgID(chatID, msgID)
	}
}

func (b *Bot) ResetDashboard(chatID int64) {
	b.dashMu.Lock()
	oldDashID := b.dashboards[chatID]
	if oldDashID == 0 && chatID == b.cfg.AllowedChatID {
		oldDashID = b.dashboardMsgID
	}
	b.dashboards[chatID] = 0
	if chatID == b.cfg.AllowedChatID {
		b.dashboardMsgID = 0
	}
	b.dashMu.Unlock()

	if b.db != nil {
		_ = b.db.SetUserDashboardMsgID(chatID, 0)
	}

	if oldDashID != 0 {
		_ = b.tg.DeleteMessage(chatID, oldDashID)
	}
}

func (b *Bot) RenderDashboard(chatID int64, text string, markup *telegram.InlineKeyboardMarkup) error {
	return b.renderDashboardForChat(chatID, text, markup)
}

func (b *Bot) RenderDashboardWithPhoto(chatID int64, photo io.Reader, caption string, markup *telegram.InlineKeyboardMarkup) error {
	return b.renderDashboardPhotoForChat(chatID, photo, caption, markup)
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
		_ = b.tg.DeleteMessage(chatID, msgID)
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

func (b *Bot) renderDashboardPhotoForChat(chatID int64, photoData io.Reader, caption string, markup *telegram.InlineKeyboardMarkup) error {
	if chatID == 0 {
		chatID = b.cfg.AllowedChatID
	}

	photoBytes, err := io.ReadAll(photoData)
	if err != nil {
		return b.renderDashboardForChat(chatID, caption, markup)
	}

	b.dashMu.Lock()
	msgID := b.dashboards[chatID]
	if msgID == 0 && chatID == b.cfg.AllowedChatID {
		msgID = b.dashboardMsgID
	}
	b.dashMu.Unlock()

	if msgID != 0 {
		err := b.tg.EditMessageMedia(chatID, msgID, bytes.NewReader(photoBytes), caption, markup)
		if err == nil {
			return nil
		}
		log.Printf("renderDashboardPhoto: не удалось обновить сообщение %d в чате %d (%v), создаю новое...", msgID, chatID, err)
		_ = b.tg.DeleteMessage(chatID, msgID)
	}

	newID, err := b.tg.SendPhotoWithMarkup(chatID, bytes.NewReader(photoBytes), caption, b.cfg.IsDNDActive(), markup)
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

func (b *Bot) IsDNDActive() bool {
	return b.cfg.IsDNDActive()
}

func (b *Bot) GetStartTime() time.Time {
	return b.startTime
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

func (b *Bot) t(key string, args ...any) string {
	return i18n.T(b.GetUserLang(b.GetActiveChatID()), key, args...)
}

func (b *Bot) tUser(chatID int64, key string, args ...any) string {
	return i18n.T(b.GetUserLang(chatID), key, args...)
}

// Delegates for backward compatibility with tests & existing code

func (b *Bot) isUserAllowed(chatID int64) bool {
	return b.authSvc.IsUserAllowed(chatID)
}

func (b *Bot) isUserAdmin(chatID int64) bool {
	return b.authSvc.IsUserAdmin(chatID)
}

func (b *Bot) handleUnauthorizedAccess(msg *telegram.Message) {
	b.authSvc.HandleUnauthorizedAccess(msg)
}

func (b *Bot) handleAuthCallback(cb *telegram.CallbackQuery) bool {
	return b.authSvc.HandleAuthCallback(cb)
}

func (b *Bot) getAdminChatIDs() []int64 {
	return b.authSvc.GetAdminChatIDs()
}

func (b *Bot) getAllUserChatIDs() []int64 {
	return b.authSvc.GetAllUserChatIDs()
}

func (b *Bot) getNotifyChatIDs(prefType string) []int64 {
	return b.authSvc.GetNotifyChatIDs(prefType)
}

func (b *Bot) handleTextCommand(msg *telegram.Message) {
	b.SetActiveChatID(msg.Chat.ID)
	b.handler.HandleTextCommand(msg)
}

func (b *Bot) handleCallback(cb *telegram.CallbackQuery) {
	b.SetActiveChatID(cb.From.ID)
	b.handler.HandleCallback(cb)
}

func (b *Bot) getMainMenuMarkup() *telegram.ReplyKeyboardMarkup {
	status, flag := b.GetRobotStatus()
	return delivery.GetMainMenuMarkup(b.Caps(), status, flag, b.GetLang())
}

func (b *Bot) sendMainDashboard() {
	b.handler.SendMainDashboard(b.GetActiveChatID())
}

func (b *Bot) getMainDashboard() (string, *telegram.InlineKeyboardMarkup) {
	status, flag := b.GetRobotStatus()
	return delivery.GetMainDashboard(b.Caps(), status, flag, b.sessionSvc.GetLastReport(), b.val, b.GetLang())
}

func (b *Bot) getMainDashboardForChat(chatID int64) (string, *telegram.InlineKeyboardMarkup) {
	status, flag := b.GetRobotStatus()
	return delivery.GetMainDashboard(b.Caps(), status, flag, b.sessionSvc.GetLastReport(), b.val, b.GetUserLang(chatID))
}

func (b *Bot) broadcastMainDashboard() {
	var mapBytes []byte
	if b.val != nil {
		if r, err := b.val.GetMapReader(); err == nil {
			mapBytes, _ = io.ReadAll(r)
			r.Close()
		}
	}
	for _, chatID := range b.authSvc.GetAllUserChatIDs() {
		text, markup := b.getMainDashboardForChat(chatID)
		if len(mapBytes) > 0 {
			_ = b.renderDashboardPhotoForChat(chatID, bytes.NewReader(mapBytes), text, markup)
		} else {
			_ = b.renderDashboardForChat(chatID, text, markup)
		}
	}
}

func (b *Bot) formatStatusDisplay(status, flag string) string {
	return delivery.FormatStatusDisplayWithError(status, flag, b.GetRobotError(), b.GetLang())
}

func (b *Bot) formatModeTitle(mode string) string {
	return delivery.FormatModeTitle(mode, b.GetLang())
}

func (b *Bot) formatReportAlert(r *CleaningReport) string {
	return delivery.FormatReportAlert(r, b.GetLang())
}

func (b *Bot) formatReportCaption(r *CleaningReport) string {
	return b.formatReportCaptionForChat(r, b.GetActiveChatID())
}

func (b *Bot) formatReportCaptionForChat(r *CleaningReport, chatID int64) string {
	return delivery.FormatReportCaptionForChat(r, chatID, b.val, b.GetUserLang(chatID))
}

func (b *Bot) buildTelemetryReport() string {
	hStats := b.systemSvc.CollectHost()
	status, flag := b.GetRobotStatus()
	return delivery.BuildTelemetryReport(b.val, b.consumablesSvc, hStats.OSUptime, b.GetLang(), b.formatStatusDisplay(status, flag))
}

func (b *Bot) buildResourcesReport() string {
	rStats := b.systemSvc.CollectRuntime(b.startTime)
	hStats := b.systemSvc.CollectHost()
	return delivery.BuildResourcesReport(rStats, hStats, b.GetLang())
}

func (b *Bot) getRooms() ([]RoomInfo, error) {
	return b.cleaningSvc.GetRooms(b.GetLang())
}

func (b *Bot) getConsumablesDisplay() ([]ConsumableDisplayInfo, error) {
	return b.consumablesSvc.GetConsumablesDisplay(b.GetLang())
}

func (b *Bot) StartSession(rooms []string, startBattery int) {
	b.sessionSvc.StartSession(rooms, startBattery)
}

func (b *Bot) SetSessionStartTime(t time.Time) {
	b.sessionSvc.SetStartTime(t)
}

func (b *Bot) UpdateSessionStats(min, sec int, areaM2 float64) {
	b.sessionSvc.UpdateStats(min, sec, areaM2)
}

func (b *Bot) FinishSession(endBattery int) *CleaningReport {
	return b.sessionSvc.FinishSession(endBattery, b.GetLang())
}

func (b *Bot) IsSessionActive() bool {
	return b.sessionSvc.IsActive()
}

func (b *Bot) GetLastReport() *CleaningReport {
	return b.sessionSvc.GetLastReport()
}

func (b *Bot) getBatteryLevel() int {
	return b.sessionSvc.GetBatteryLevel()
}

func (b *Bot) getSettingsMainMenu() (string, *telegram.InlineKeyboardMarkup) {
	return delivery.GetSettingsMainMenu(b.Caps(), b.db != nil, b.isUserAdmin(b.GetActiveChatID()), b.GetLang())
}

func (b *Bot) getRobotSettingsMenu() (string, *telegram.InlineKeyboardMarkup) {
	return delivery.GetRobotSettingsMenu(b.Caps(), b.GetLang())
}

func (b *Bot) getBotSettingsMenu() (string, *telegram.InlineKeyboardMarkup) {
	return delivery.GetBotSettingsMenu(b.db != nil, b.isUserAdmin(b.GetActiveChatID()), b.GetLang())
}

func (b *Bot) getUsersMenu(currentChatID int64) (string, *telegram.InlineKeyboardMarkup) {
	return delivery.GetUsersMenu(b.db, currentChatID, b.GetLang())
}

func (b *Bot) getAuditLogMenu() (string, *telegram.InlineKeyboardMarkup) {
	return delivery.GetAuditLogMenu(b.db, 1, 0, b.GetLang())
}

func (b *Bot) formatRemainingTime(remMin int) string {
	return b.consumablesSvc.FormatRemainingTime(remMin, b.GetLang())
}

func (b *Bot) Run(ctx context.Context) error {
	log.Printf("Бот запущен. Слушаю входящие обновления Telegram...\n")

	if err := b.LoadCapabilities(); err != nil {
		log.Printf("Внимание: не удалось загрузить возможности робота: %v", err)
	}

	// Проверка и уведомление о завершении предыдущего обновления
	b.updateSvc.CheckAndNotifyPostUpdate(ctx)

	// Запуск фонового мониторинга состояния робота
	go b.watcherSvc.Start(ctx)

	// Запуск фоновой периодической проверки обновлений
	go b.updateSvc.Start(ctx)

	// Начальный heartbeat для супервизора
	netutil.TouchAliveFile("")

	// Фоновый тикер для обновления heartbeat при нормальной работе
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	var consecutiveErrors int
	var firstErrorTime time.Time

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				if consecutiveErrors == 0 {
					netutil.TouchAliveFile("")
				}
			}
		}
	}()

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
			consecutiveErrors++
			if consecutiveErrors == 1 {
				firstErrorTime = time.Now()
			}
			duration := time.Since(firstErrorTime)
			log.Printf("Ошибка poll-запроса: %v. Непрерывных ошибок: %d (длительность: %v). Повтор через 3с...", err, consecutiveErrors, duration.Round(time.Second))

			// Если ошибки длятся более 5 минут непрерывно, завершаем работу для перезапуска супервизором
			if duration > 5*time.Minute {
				log.Printf("КРИТИЧЕСКАЯ ОШИБКА: связь с Telegram отсутствует более 5 минут (%d ошибок подряд). Завершение работы для перезапуска...", consecutiveErrors)
				return fmt.Errorf("continuous poll failure for %v (%d errors): %w", duration, consecutiveErrors, err)
			}

			time.Sleep(3 * time.Second)
			continue
		}

		if consecutiveErrors > 0 {
			log.Printf("Связь с Telegram восстановлена после %d ошибок (перерыв %v).", consecutiveErrors, time.Since(firstErrorTime).Round(time.Second))
			consecutiveErrors = 0
		}
		netutil.TouchAliveFile("")

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.Message != nil {
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("handleMessage: перехвачена паника: %v", r)
						}
					}()
					if b.isUserAllowed(update.Message.Chat.ID) {
						b.handleTextCommand(update.Message)
					} else {
						b.handleUnauthorizedAccess(update.Message)
					}
				}()
			}

			if update.CallbackQuery != nil {
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("handleCallbackQuery: перехвачена паника: %v", r)
						}
					}()
					if b.handleAuthCallback(update.CallbackQuery) {
						return
					}

					if b.isUserAllowed(update.CallbackQuery.From.ID) {
						b.handleCallback(update.CallbackQuery)
					} else {
						_ = b.tg.AnswerCallbackQueryAlert(update.CallbackQuery.ID, "⛔ У вас нет доступа.", true)
					}
				}()
			}
		}
	}
}
