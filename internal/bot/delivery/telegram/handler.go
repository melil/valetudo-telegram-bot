package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"tgbot/internal/bot/domain"
	"tgbot/internal/bot/service/auth"
	"tgbot/internal/bot/service/cleaning"
	"tgbot/internal/bot/service/consumables"
	"tgbot/internal/bot/service/session"
	"tgbot/internal/bot/service/system"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

type BotFacade interface {
	GetCaps() *valetudo.CapabilitySet
	GetRobotStatus() (string, string)
	SetRobotStatus(status, flag string)
	GetUserLang(chatID int64) i18n.Locale
	SetUserLang(chatID int64, loc i18n.Locale)
	RenderDashboard(chatID int64, text string, markup *telegram.InlineKeyboardMarkup) error
	GetDashboardMsgID(chatID ...int64) int
	ResetDashboard(chatID int64)
	IsDNDActive() bool
	GetStartTime() time.Time
}

type Handler struct {
	tg             domain.Messenger
	val            domain.RobotClient
	db             domain.UserRepository
	authSvc        *auth.Service
	cleaningSvc    *cleaning.WizardService
	consumablesSvc *consumables.Service
	sessionSvc     *session.Service
	systemSvc      *system.Service
	facade         BotFacade
}

func NewHandler(
	tg domain.Messenger,
	val domain.RobotClient,
	db domain.UserRepository,
	authSvc *auth.Service,
	cleaningSvc *cleaning.WizardService,
	consumablesSvc *consumables.Service,
	sessionSvc *session.Service,
	systemSvc *system.Service,
	facade BotFacade,
) *Handler {
	return &Handler{
		tg:             tg,
		val:            val,
		db:             db,
		authSvc:        authSvc,
		cleaningSvc:    cleaningSvc,
		consumablesSvc: consumablesSvc,
		sessionSvc:     sessionSvc,
		systemSvc:      systemSvc,
		facade:         facade,
	}
}

func (h *Handler) t(chatID int64, key string, args ...any) string {
	return i18n.T(h.facade.GetUserLang(chatID), key, args...)
}

func (h *Handler) logAction(chatID int64, action, details string) {
	if h.db != nil {
		var username string
		if u, err := h.db.GetUser(chatID); err == nil && u != nil {
			username = u.Username
		}
		_ = h.db.LogAction(chatID, username, action, details)
	}
}

func (h *Handler) SendMainDashboard(chatID int64) {
	status, flag := h.facade.GetRobotStatus()
	lastReport := h.sessionSvc.GetLastReport()
	loc := h.facade.GetUserLang(chatID)

	text, markup := GetMainDashboard(h.facade.GetCaps(), status, flag, lastReport, h.val, loc)
	_ = h.facade.RenderDashboard(chatID, text, markup)
}

func (h *Handler) SendMainMenuKeyboard(chatID int64) {
	status, flag := h.facade.GetRobotStatus()
	loc := h.facade.GetUserLang(chatID)
	markup := GetMainMenuMarkup(h.facade.GetCaps(), status, flag, loc)
	_, _ = h.tg.SendPayload(telegram.SendMessagePayload{
		ChatID:              chatID,
		Text:                h.t(chatID, "main_menu.ready"),
		ReplyMarkup:         markup,
		DisableNotification: true,
	})
}

func (h *Handler) HandleTextCommand(msg *telegram.Message) {
	if msg == nil {
		return
	}
	chatID := msg.Chat.ID
	_ = h.tg.DeleteMessage(chatID, msg.MessageID)

	cleanText := strings.TrimSpace(msg.Text)
	if cleanText == "/start" {
		h.cleaningSvc.CancelSession(chatID)
	}

	h.facade.ResetDashboard(chatID)

	caps := h.facade.GetCaps()
	loc := h.facade.GetUserLang(chatID)

	switch {
	case cleanText == "/start" || cleanText == "Меню" || cleanText == "Menu" || cleanText == "Menü" || cleanText == "菜单":
		h.SendMainDashboard(chatID)

	case cleanText == "/wizard" || i18n.Matches(cleanText, "main_menu.start_cleaning"):
		if !caps.Has(valetudo.CapMapSegmentation) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: h.t(chatID, "main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = h.facade.RenderDashboard(chatID, h.t(chatID, "main_menu.not_supported"), markup)
			return
		}
		h.startCleaningWizard(chatID)

	case cleanText == "/resume" || i18n.Matches(cleanText, "main_menu.resume_cleaning") || i18n.Matches(cleanText, "robot_menu.btn_resume"):
		if !caps.Has(valetudo.CapBasicControl) {
			h.showNotSupported(chatID)
			return
		}
		_ = h.val.TriggerAction("start")
		h.facade.SetRobotStatus("cleaning", "none")
		h.logAction(chatID, "resume_cleaning", "")
		h.SendMainDashboard(chatID)

	case cleanText == "/stop" || i18n.Matches(cleanText, "main_menu.stop_robot") || i18n.Matches(cleanText, "main_menu.stop_cleaning") || i18n.Matches(cleanText, "robot_menu.btn_stop"):
		if !caps.Has(valetudo.CapBasicControl) {
			h.showNotSupported(chatID)
			return
		}
		_ = h.val.TriggerAction("stop")
		h.facade.SetRobotStatus("idle", "none")
		h.logAction(chatID, "stop_cleaning", "")
		h.SendMainDashboard(chatID)

	case cleanText == "/pause" || i18n.Matches(cleanText, "main_menu.pause_cleaning") || i18n.Matches(cleanText, "robot_menu.btn_pause"):
		if !caps.Has(valetudo.CapBasicControl) {
			h.showNotSupported(chatID)
			return
		}
		_ = h.val.TriggerAction("pause")
		h.facade.SetRobotStatus("paused", "none")
		h.logAction(chatID, "pause_cleaning", "")
		h.SendMainDashboard(chatID)

	case cleanText == "/home" || i18n.Matches(cleanText, "main_menu.go_home") || i18n.Matches(cleanText, "station_menu.btn_dock_home"):
		if !caps.Has(valetudo.CapBasicControl) {
			h.showNotSupported(chatID)
			return
		}
		_ = h.val.TriggerAction("home")
		h.facade.SetRobotStatus("returning", "none")
		h.logAction(chatID, "go_home", "")
		h.SendMainDashboard(chatID)

	case cleanText == "/locate" || i18n.Matches(cleanText, "main_menu.locate"):
		if !caps.Has(valetudo.CapLocate) {
			h.showNotSupported(chatID)
			return
		}
		_ = h.val.TriggerCapabilityAction("LocateCapability", "locate")
		h.logAction(chatID, "locate", "")
		h.SendMainDashboard(chatID)

	case cleanText == "/robot" || i18n.Matches(cleanText, "main_menu.robot"):
		h.sendRobotMenu(chatID)

	case cleanText == "/station" || i18n.Matches(cleanText, "main_menu.station"):
		if !caps.HasStation() {
			h.showNotSupported(chatID)
			return
		}
		h.sendStationMenu(chatID)

	case cleanText == "/stats" || cleanText == "/telemetry":
		h.sendTelemetryMenu(chatID)

	case cleanText == "/status":
		h.SendMainDashboard(chatID)

	case cleanText == "/resources":
		h.sendResourcesMenu(chatID)

	case cleanText == "/users":
		if !h.authSvc.IsUserAdmin(chatID) {
			_ = h.facade.RenderDashboard(chatID, h.t(chatID, "auth.admin_only"), nil)
			return
		}
		text, markup := GetUsersMenu(h.db, chatID, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case cleanText == "/lang":
		text, markup := GetLanguageMenu(loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case cleanText == "/map":
		h.sendMap(chatID)

	case cleanText == "/rooms" || i18n.Matches(cleanText, "main_menu.rooms"):
		if !caps.Has(valetudo.CapMapSegmentation) {
			h.showNotSupported(chatID)
			return
		}
		h.sendRoomsMenu(chatID)

	case cleanText == "/clean" || i18n.Matches(cleanText, "main_menu.full_clean") || i18n.Matches(cleanText, "robot_menu.btn_start"):
		if !caps.Has(valetudo.CapBasicControl) {
			h.showNotSupported(chatID)
			return
		}
		_ = h.val.TriggerAction("start")
		h.facade.SetRobotStatus("cleaning", "none")
		h.sessionSvc.StartSession(nil, h.sessionSvc.GetBatteryLevel())
		h.logAction(chatID, "start_cleaning", "")
		h.SendMainDashboard(chatID)

	default:
		h.SendMainDashboard(chatID)
	}
}

func (h *Handler) showNotSupported(chatID int64) {
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: h.t(chatID, "main_menu.btn_back_main"), CallbackData: "menu_main"}},
		},
	}
	_ = h.facade.RenderDashboard(chatID, h.t(chatID, "main_menu.not_supported"), markup)
}

func (h *Handler) sendRobotMenu(chatID int64) {
	status, flag := h.facade.GetRobotStatus()
	loc := h.facade.GetUserLang(chatID)
	text, markup := GetRobotMenu(h.facade.GetCaps(), status, flag, loc)
	_ = h.facade.RenderDashboard(chatID, text, markup)
}

func (h *Handler) sendStationMenu(chatID int64) {
	status, _ := h.facade.GetRobotStatus()
	loc := h.facade.GetUserLang(chatID)
	text, markup := GetStationMenu(h.facade.GetCaps(), status, loc)
	_ = h.facade.RenderDashboard(chatID, text, markup)
}

func (h *Handler) sendTelemetryMenu(chatID int64) {
	loc := h.facade.GetUserLang(chatID)
	status, flag := h.facade.GetRobotStatus()
	hStats := h.systemSvc.CollectHost()
	text := BuildTelemetryReport(h.val, h.consumablesSvc, hStats.OSUptime, loc, FormatStatusDisplay(status, flag, loc))
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: h.t(chatID, "robot_menu.btn_back"), CallbackData: "menu_robot"}},
		},
	}
	_ = h.facade.RenderDashboard(chatID, text, markup)
}

func (h *Handler) sendResourcesMenu(chatID int64) {
	loc := h.facade.GetUserLang(chatID)
	rStats := h.systemSvc.CollectRuntime(h.facade.GetStartTime())
	hStats := h.systemSvc.CollectHost()
	text := BuildResourcesReport(rStats, hStats, loc)
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: h.t(chatID, "resources.btn_refresh"), CallbackData: "cmd_resources_refresh"}},
			{{Text: h.t(chatID, "robot_menu.btn_back"), CallbackData: "menu_robot"}},
		},
	}
	_ = h.facade.RenderDashboard(chatID, text, markup)
}

func (h *Handler) sendRoomsMenu(chatID int64) {
	loc := h.facade.GetUserLang(chatID)
	rooms, err := h.cleaningSvc.GetRooms(loc)
	text, markup := GetRoomsMenu(rooms, err, loc)
	_ = h.facade.RenderDashboard(chatID, text, markup)
}

func (h *Handler) sendMap(chatID int64) {
	reader, err := h.val.GetMapReader()
	if err != nil {
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: h.t(chatID, "main_menu.btn_back_main"), CallbackData: "menu_main"}},
			},
		}
		_ = h.facade.RenderDashboard(chatID, h.t(chatID, "map.err_get")+"\n"+err.Error(), markup)
		return
	}
	defer reader.Close()

	caption := h.t(chatID, "map.caption")
	_ = h.tg.SendPhoto(chatID, reader, caption, h.facade.IsDNDActive())
}

func (h *Handler) startCleaningWizard(chatID int64) {
	loc := h.facade.GetUserLang(chatID)
	rooms, err := h.cleaningSvc.GetRooms(loc)
	if err != nil || len(rooms) == 0 {
		errMsg := h.t(chatID, "wizard.err_get_rooms")
		if err != nil {
			errMsg += "\n" + err.Error()
		}
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: h.t(chatID, "main_menu.btn_back_main"), CallbackData: "menu_main"}},
			},
		}
		_ = h.facade.RenderDashboard(chatID, errMsg, markup)
		return
	}

	session := h.cleaningSvc.StartSession(chatID, h.facade.GetDashboardMsgID(chatID), rooms)

	var text string
	var markup *telegram.InlineKeyboardMarkup
	if h.facade.GetCaps().Has(valetudo.CapOperationModeControl) {
		text, markup = GetWizardStep1(loc)
	} else {
		text, markup = GetWizardStep2(session, loc)
	}
	_ = h.facade.RenderDashboard(chatID, text, markup)
}

func (h *Handler) HandleWizardCallback(cb *telegram.CallbackQuery) bool {
	data := cb.Data
	if !strings.HasPrefix(data, "wiz_") {
		return false
	}

	chatID := cb.From.ID
	loc := h.facade.GetUserLang(chatID)

	if data == "wiz_start" {
		h.startCleaningWizard(chatID)
		return true
	}

	ws, exists := h.cleaningSvc.GetSession(chatID)
	if !exists || ws == nil {
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: h.t(chatID, "main_menu.start_cleaning"), CallbackData: "wiz_start"},
					{Text: h.t(chatID, "main_menu.btn_back_main"), CallbackData: "menu_main"},
				},
			},
		}
		_ = h.facade.RenderDashboard(chatID, h.t(chatID, "wizard.session_expired"), markup)
		return true
	}

	switch {
	case data == "wiz_cancel":
		h.cleaningSvc.CancelSession(chatID)
		h.SendMainDashboard(chatID)

	case strings.HasPrefix(data, "wiz_mode:"):
		mode := strings.TrimPrefix(data, "wiz_mode:")
		h.cleaningSvc.SetMode(chatID, mode)
		text, markup := GetWizardStep2(ws, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case strings.HasPrefix(data, "wiz_toggle_room:"):
		roomID := strings.TrimPrefix(data, "wiz_toggle_room:")
		h.cleaningSvc.ToggleRoom(chatID, roomID)
		text, markup := GetWizardStep2(ws, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "wiz_select_all":
		h.cleaningSvc.SelectAll(chatID)
		text, markup := GetWizardStep2(ws, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "wiz_select_none":
		h.cleaningSvc.SelectNone(chatID)
		text, markup := GetWizardStep2(ws, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "wiz_to_step3":
		text, markup := GetWizardStep3(ws, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "wiz_back_to_step2":
		text, markup := GetWizardStep2(ws, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case strings.HasPrefix(data, "wiz_iter:"):
		iterStr := strings.TrimPrefix(data, "wiz_iter:")
		iterations, _ := strconv.Atoi(iterStr)
		if iterations <= 0 {
			iterations = 1
		}

		hasModeCap := h.facade.GetCaps().Has(valetudo.CapOperationModeControl)
		mode, targetNames, iters, err := h.cleaningSvc.ExecuteCleaning(chatID, iterations, hasModeCap)
		if err != nil {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: h.t(chatID, "main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = h.facade.RenderDashboard(chatID, fmt.Sprintf(h.t(chatID, "wizard.err_start_segments"), err.Error()), markup)
			return true
		}

		h.facade.SetRobotStatus("cleaning", "none")
		h.sessionSvc.StartSession(targetNames, h.sessionSvc.GetBatteryLevel())
		h.logAction(chatID, "wizard_clean", strings.Join(targetNames, ", "))

		modeTitle := "—"
		if mode != "" {
			modeTitle = FormatModeTitle(mode, loc)
		}

		successMsg := fmt.Sprintf(
			h.t(chatID, "wizard.started_title"),
			modeTitle, strings.Join(targetNames, ", "), iters,
		)
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: h.t(chatID, "main_menu.btn_back_main"), CallbackData: "menu_main"},
					{Text: h.t(chatID, "robot_menu.btn_back"), CallbackData: "menu_robot"},
				},
			},
		}
		_ = h.facade.RenderDashboard(chatID, successMsg, markup)
	}

	return true
}

func (h *Handler) HandleCallback(cb *telegram.CallbackQuery) {
	if cb == nil {
		return
	}
	chatID := cb.From.ID
	loc := h.facade.GetUserLang(chatID)

	// Делегируем шаги визарда
	if h.HandleWizardCallback(cb) {
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		return
	}

	data := cb.Data
	caps := h.facade.GetCaps()

	switch {
	case data == "noop":
		_ = h.tg.AnswerCallbackQuery(cb.ID)

	case data == "menu_main":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.SendMainDashboard(chatID)

	case data == "menu_robot":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendRobotMenu(chatID)

	case data == "menu_station":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendStationMenu(chatID)

	case data == "menu_rooms":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendRoomsMenu(chatID)

	case data == "menu_settings":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		isAdmin := h.authSvc.IsUserAdmin(chatID)
		text, markup := GetSettingsMainMenu(caps, h.db != nil, isAdmin, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "sub_users":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		if !h.authSvc.IsUserAdmin(chatID) {
			_ = h.tg.AnswerCallbackQueryAlert(cb.ID, h.t(chatID, "auth.admin_only"), true)
			return
		}
		text, markup := GetUsersMenu(h.db, chatID, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "sub_lang":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		text, markup := GetLanguageMenu(loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case strings.HasPrefix(data, "set_lang:"):
		newLang := strings.TrimPrefix(data, "set_lang:")
		norm := i18n.NormalizeLocale(newLang)
		h.facade.SetUserLang(chatID, norm)
		h.logAction(chatID, "set_lang", string(norm))
		_ = h.tg.AnswerCallbackQueryAlert(cb.ID, fmt.Sprintf(h.t(chatID, "lang.changed"), i18n.LocaleName(norm)), false)
		h.SendMainMenuKeyboard(chatID)
		h.SendMainDashboard(chatID)

	case data == "sub_notifications":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		text, markup := GetNotificationsMenu(h.db, chatID, caps.HasStation(), loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case strings.HasPrefix(data, "toggle_notify:"):
		prefType := strings.TrimPrefix(data, "toggle_notify:")
		if h.db != nil {
			if u, err := h.db.GetUser(chatID); err == nil && u != nil {
				currentVal := true
				switch prefType {
				case "errors":
					currentVal = u.NotifyErrors
				case "reports":
					currentVal = u.NotifyReports
				case "station":
					currentVal = u.NotifyStation
				}
				_ = h.db.SetUserNotificationPref(chatID, prefType, !currentVal)
			}
		}
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		text, markup := GetNotificationsMenu(h.db, chatID, caps.HasStation(), loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "sub_audit":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		if !h.authSvc.IsUserAdmin(chatID) {
			_ = h.tg.AnswerCallbackQueryAlert(cb.ID, h.t(chatID, "auth.admin_only"), true)
			return
		}
		if h.db == nil {
			_ = h.facade.RenderDashboard(chatID, "⚠️ База данных не подключена.", nil)
			return
		}
		logs, err := h.db.GetRecentAuditLogs(10)
		if err != nil || len(logs) == 0 {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: h.t(chatID, "settings_menu.btn_back"), CallbackData: "menu_settings"}},
				},
			}
			_ = h.facade.RenderDashboard(chatID, h.t(chatID, "audit.empty"), markup)
			return
		}
		text := h.t(chatID, "audit.title") + "\n\n"
		for _, l := range logs {
			userStr := "ID " + strconv.FormatInt(l.ChatID, 10)
			if l.Username != "" {
				userStr = "@" + l.Username
			}
			timeStr := l.CreatedAt.Local().Format("02.01 15:04")
			text += fmt.Sprintf("• <code>%s</code> <b>%s</b>: %s\n", timeStr, userStr, FormatActionTitle(l.Action, l.Details, loc))
		}
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: h.t(chatID, "audit.btn_refresh"), CallbackData: "sub_audit"},
					{Text: h.t(chatID, "settings_menu.btn_back"), CallbackData: "menu_settings"},
				},
			},
		}
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case strings.HasPrefix(data, "user_del:"):
		if !h.authSvc.IsUserAdmin(chatID) {
			_ = h.tg.AnswerCallbackQueryAlert(cb.ID, h.t(chatID, "auth.admin_only"), true)
			return
		}
		targetIDStr := strings.TrimPrefix(data, "user_del:")
		targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
		if err != nil {
			_ = h.tg.AnswerCallbackQueryAlert(cb.ID, "Ошибка парсинга ID", false)
			return
		}
		if targetID == chatID {
			_ = h.tg.AnswerCallbackQueryAlert(cb.ID, h.t(chatID, "settings_menu.cannot_delete_self"), true)
			return
		}
		var targetUsername string
		if h.db != nil {
			if u, err := h.db.GetUser(targetID); err == nil && u != nil {
				targetUsername = u.Username
			}
			if err := h.db.DeleteUser(targetID); err != nil {
				log.Printf("Ошибка удаления пользователя %d: %v", targetID, err)
				_ = h.tg.AnswerCallbackQueryAlert(cb.ID, "Ошибка удаления из базы данных", false)
				return
			}
		}

		h.facade.ResetDashboard(targetID)
		h.cleaningSvc.CancelSession(targetID)

		// Уведомление удаленному пользователю
		_, _ = h.tg.SendTextMessage(targetID, "⛔ Ваш доступ к управлению роботом был отозван администратором.", false, nil)

		userDisplay := fmt.Sprintf("ID %d", targetID)
		if targetUsername != "" {
			userDisplay = "@" + targetUsername
		}
		h.logAction(cb.From.ID, "user_deleted", userDisplay)

		alertText := fmt.Sprintf(h.t(chatID, "settings_menu.user_deleted"), userDisplay)
		_ = h.tg.AnswerCallbackQueryAlert(cb.ID, alertText, false)

		text, markup := GetUsersMenu(h.db, chatID, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "cmd_start":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerAction("start")
		h.facade.SetRobotStatus("cleaning", "none")
		h.sessionSvc.StartSession(nil, h.sessionSvc.GetBatteryLevel())
		h.logAction(chatID, "start_cleaning", "")
		h.SendMainDashboard(chatID)

	case data == "cmd_pause":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerAction("pause")
		h.facade.SetRobotStatus("paused", "none")
		h.logAction(chatID, "pause_cleaning", "")
		h.SendMainDashboard(chatID)

	case data == "cmd_resume":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerAction("start")
		h.facade.SetRobotStatus("cleaning", "none")
		h.logAction(chatID, "resume_cleaning", "")
		h.SendMainDashboard(chatID)

	case data == "cmd_stop":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerAction("stop")
		h.facade.SetRobotStatus("idle", "none")
		h.logAction(chatID, "stop_cleaning", "")
		h.SendMainDashboard(chatID)

	case data == "cmd_home" || data == "cmd_station_home":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerAction("home")
		h.facade.SetRobotStatus("returning", "none")
		h.logAction(chatID, "go_home", "")
		h.SendMainDashboard(chatID)

	case data == "cmd_locate":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerCapabilityAction("LocateCapability", "locate")
		h.logAction(chatID, "locate", "")
		h.SendMainDashboard(chatID)

	case data == "dock_empty":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerCapabilityAction("AutoEmptyDockManualTriggerCapability", "trigger")
		h.logAction(chatID, "dock_empty", "")
		h.sendStationMenu(chatID)

	case data == "dock_wash":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerCapabilityAction("MopDockCleanManualTriggerCapability", "start")
		h.logAction(chatID, "dock_wash", "")
		h.sendStationMenu(chatID)

	case data == "dock_dry_start":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "start")
		h.logAction(chatID, "dock_dry_start", "")
		h.sendStationMenu(chatID)

	case data == "dock_dry_stop":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		_ = h.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "stop")
		h.logAction(chatID, "dock_dry_stop", "")
		h.sendStationMenu(chatID)

	case data == "cmd_telemetry":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendTelemetryMenu(chatID)

	case data == "cmd_consumables":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		displays, err := h.consumablesSvc.GetConsumablesDisplay(loc)
		text, markup := GetConsumablesMenu(caps, displays, err, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case strings.HasPrefix(data, "reset_cons:"):
		parts := strings.Split(strings.TrimPrefix(data, "reset_cons:"), ":")
		if len(parts) >= 2 {
			cType, subType := parts[0], parts[1]
			_ = h.consumablesSvc.ResetConsumable(cType, subType)
			name, _, _ := h.consumablesSvc.GetConsumableMeta(cType, subType, loc)
			_ = h.tg.AnswerCallbackQueryAlert(cb.ID, fmt.Sprintf(h.t(chatID, "consumables.reset_success"), name), false)
		} else {
			_ = h.tg.AnswerCallbackQuery(cb.ID)
		}
		displays, err := h.consumablesSvc.GetConsumablesDisplay(loc)
		text, markup := GetConsumablesMenu(caps, displays, err, loc)
		_ = h.facade.RenderDashboard(chatID, text, markup)

	case data == "cmd_resources":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendResourcesMenu(chatID)

	case data == "cmd_resources_refresh":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendResourcesMenu(chatID)

	case data == "view_last_report":
		report := h.sessionSvc.GetLastReport()
		alertText := FormatReportAlert(report, loc)
		_ = h.tg.AnswerCallbackQueryAlert(cb.ID, alertText, true)

	case data == "sub_mode":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		modes := []struct {
			ID   string
			Name string
		}{
			{"vacuum_and_mop", h.t(chatID, "modes.vacuum_and_mop")},
			{"vacuum_then_mop", h.t(chatID, "modes.vacuum_then_mop")},
			{"vacuum", h.t(chatID, "modes.vacuum")},
			{"mop", h.t(chatID, "modes.mop")},
		}
		var rows [][]telegram.InlineKeyboardButton
		for _, m := range modes {
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: m.Name, CallbackData: "set_mode:" + m.ID},
			})
		}
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: h.t(chatID, "settings_menu.btn_back"), CallbackData: "menu_settings"},
		})
		_ = h.facade.RenderDashboard(chatID, h.t(chatID, "modes.select_title"), &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})

	case strings.HasPrefix(data, "set_mode:"):
		mode := strings.TrimPrefix(data, "set_mode:")
		_ = h.val.SetOperationMode(mode)
		h.logAction(chatID, "set_mode", mode)
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendRobotMenu(chatID)

	case data == "sub_fan":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		speeds := []string{"off", "quiet", "turbo", "max"}
		var rows [][]telegram.InlineKeyboardButton
		for _, sp := range speeds {
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: h.t(chatID, "fan."+sp), CallbackData: "set_fan:" + sp},
			})
		}
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: h.t(chatID, "settings_menu.btn_back"), CallbackData: "menu_settings"},
		})
		_ = h.facade.RenderDashboard(chatID, h.t(chatID, "fan.select_title"), &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})

	case strings.HasPrefix(data, "set_fan:"):
		speed := strings.TrimPrefix(data, "set_fan:")
		_ = h.val.SetPreset("FanSpeedControlCapability", speed)
		h.logAction(chatID, "set_fan", speed)
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendRobotMenu(chatID)

	case data == "sub_water":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		grades := []string{"off", "low", "medium", "high"}
		var rows [][]telegram.InlineKeyboardButton
		for _, g := range grades {
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: h.t(chatID, "water."+g), CallbackData: "set_water:" + g},
			})
		}
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: h.t(chatID, "settings_menu.btn_back"), CallbackData: "menu_settings"},
		})
		_ = h.facade.RenderDashboard(chatID, h.t(chatID, "water.select_title"), &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})

	case strings.HasPrefix(data, "set_water:"):
		grade := strings.TrimPrefix(data, "set_water:")
		_ = h.val.SetPreset("WaterUsageControlCapability", grade)
		h.logAction(chatID, "set_water", grade)
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendRobotMenu(chatID)

	case data == "sub_mopextend":
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		modes := []string{"off", "standard", "frequent"}
		var rows [][]telegram.InlineKeyboardButton
		for _, m := range modes {
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: h.t(chatID, "mopextend."+m), CallbackData: "set_mopextend:" + m},
			})
		}
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: h.t(chatID, "settings_menu.btn_back"), CallbackData: "menu_settings"},
		})
		_ = h.facade.RenderDashboard(chatID, h.t(chatID, "mopextend.select_title"), &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})

	case strings.HasPrefix(data, "set_mopextend:"):
		mode := strings.TrimPrefix(data, "set_mopextend:")
		_ = h.val.TriggerCapabilityAction("MopExtensionControlCapability", mode)
		h.logAction(chatID, "set_mopextend", mode)
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		h.sendRobotMenu(chatID)

	default:
		_ = h.tg.AnswerCallbackQuery(cb.ID)
		log.Printf("Неизвестный callback_data: %s", data)
	}
}
