package telegram

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"tgbot/internal/bot/domain"
	"tgbot/internal/bot/service/update"
	"tgbot/internal/database"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func GetMainMenuMarkup(caps *valetudo.CapabilitySet, status, flag string, loc i18n.Locale) *telegram.ReplyKeyboardMarkup {
	var keyboard [][]string

	// Ряд 1: Управление уборкой в зависимости от статуса робота
	var cleanRow []string
	if caps.Has(valetudo.CapBasicControl) {
		switch {
		case status == "cleaning" || status == "moving" || status == "returning":
			cleanRow = append(cleanRow, i18n.T(loc, "main_menu.pause_cleaning"), i18n.T(loc, "main_menu.stop_robot"), i18n.T(loc, "main_menu.go_home"))
		case status == "paused" || flag == "resumable":
			cleanRow = append(cleanRow, i18n.T(loc, "main_menu.resume_cleaning"), i18n.T(loc, "main_menu.stop_robot"), i18n.T(loc, "main_menu.go_home"))
		default: // "docked", "idle", "error", etc.
			if caps.Has(valetudo.CapMapSegmentation) {
				cleanRow = append(cleanRow, i18n.T(loc, "main_menu.start_cleaning"))
			} else {
				cleanRow = append(cleanRow, i18n.T(loc, "main_menu.full_clean"))
			}
		}
	}
	if len(cleanRow) > 0 {
		keyboard = append(keyboard, cleanRow)
	}

	// Ряд 2: Устройства (Робот, Станция)
	var deviceRow []string
	deviceRow = append(deviceRow, i18n.T(loc, "main_menu.robot"))
	if caps.HasStation() {
		deviceRow = append(deviceRow, i18n.T(loc, "main_menu.station"))
	}
	keyboard = append(keyboard, deviceRow)

	// Ряд 3: Дополнительно (Комнаты, Поиск робота)
	var utilRow []string
	if caps.Has(valetudo.CapMapSegmentation) {
		utilRow = append(utilRow, i18n.T(loc, "main_menu.rooms"))
	}
	if caps.Has(valetudo.CapLocate) {
		utilRow = append(utilRow, i18n.T(loc, "main_menu.locate"))
	}
	if len(utilRow) > 0 {
		keyboard = append(keyboard, utilRow)
	}

	return &telegram.ReplyKeyboardMarkup{
		Keyboard:       keyboard,
		ResizeKeyboard: true,
	}
}

func GetMainDashboard(
	caps *valetudo.CapabilitySet,
	status, flag string,
	lastReport *domain.CleaningReport,
	val domain.RobotClient,
	loc i18n.Locale,
) (string, *telegram.InlineKeyboardMarkup) {
	batStr := ""
	if val != nil {
		if attrs, err := val.GetAttributes(); err == nil {
			for _, attr := range attrs {
				if attr.Class == "BatteryStateAttribute" {
					batStr = fmt.Sprintf(" | 🔋 <b>%d%%</b>", attr.Level)
					break
				}
			}
		}
	}

	statusDisplay := FormatStatusDisplay(status, flag, loc)

	text := fmt.Sprintf("%s\n\n• <b>%s:</b> %s%s",
		i18n.T(loc, "main_menu.ready"),
		i18n.T(loc, "telemetry.lbl_status"),
		statusDisplay,
		batStr,
	)

	if lastReport != nil {
		text += fmt.Sprintf("\n• <b>%s:</b> ⏱ %d %s %d %s | 📐 %.1f м² | 🔋 -%d%%",
			i18n.T(loc, "report.last_clean"),
			lastReport.DurationMin, i18n.T(loc, "report.min"),
			lastReport.DurationSec, i18n.T(loc, "report.sec"),
			lastReport.AreaM2,
			lastReport.BatteryUsed,
		)
	}

	var rows [][]telegram.InlineKeyboardButton

	// Ряд 1: Контекстное управление уборкой в зависимости от статуса
	if caps.Has(valetudo.CapBasicControl) {
		switch {
		case status == "cleaning" || status == "moving":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: i18n.T(loc, "main_menu.pause_cleaning"), CallbackData: "cmd_pause"},
				{Text: i18n.T(loc, "main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "paused" || flag == "resumable":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: i18n.T(loc, "main_menu.resume_cleaning"), CallbackData: "cmd_resume"},
				{Text: i18n.T(loc, "main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "returning":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: i18n.T(loc, "main_menu.pause_cleaning"), CallbackData: "cmd_pause"},
				{Text: i18n.T(loc, "main_menu.resume_cleaning"), CallbackData: "cmd_resume"},
			})
		default: // "docked", "idle", "error", etc.
			var defaultRow []telegram.InlineKeyboardButton
			if caps.Has(valetudo.CapMapSegmentation) {
				defaultRow = append(defaultRow, telegram.InlineKeyboardButton{
					Text: i18n.T(loc, "main_menu.start_cleaning"), CallbackData: "wiz_start",
				})
			} else {
				defaultRow = append(defaultRow, telegram.InlineKeyboardButton{
					Text: i18n.T(loc, "main_menu.full_clean"), CallbackData: "cmd_start",
				})
			}
			if status != "docked" {
				defaultRow = append(defaultRow, telegram.InlineKeyboardButton{
					Text: i18n.T(loc, "main_menu.go_home"), CallbackData: "cmd_home",
				})
			}
			if len(defaultRow) > 0 {
				rows = append(rows, defaultRow)
			}
			if lastReport != nil {
				rows = append(rows, []telegram.InlineKeyboardButton{
					{Text: i18n.T(loc, "main_menu.btn_last_report"), CallbackData: "view_last_report"},
				})
			}
		}
	}

	// Ряд 2: Устройства (Робот, Станция)
	var deviceRow []telegram.InlineKeyboardButton
	deviceRow = append(deviceRow, telegram.InlineKeyboardButton{
		Text:         i18n.T(loc, "main_menu.robot"),
		CallbackData: "menu_robot",
	})
	if caps.HasStation() {
		deviceRow = append(deviceRow, telegram.InlineKeyboardButton{
			Text:         i18n.T(loc, "main_menu.station"),
			CallbackData: "menu_station",
		})
	}
	rows = append(rows, deviceRow)

	// Ряд 3: Дополнительно (Комнаты, Поиск робота)
	var utilRow []telegram.InlineKeyboardButton
	if caps.Has(valetudo.CapMapSegmentation) {
		utilRow = append(utilRow, telegram.InlineKeyboardButton{
			Text:         i18n.T(loc, "main_menu.rooms"),
			CallbackData: "menu_rooms",
		})
	}
	if caps.Has(valetudo.CapLocate) {
		utilRow = append(utilRow, telegram.InlineKeyboardButton{
			Text:         i18n.T(loc, "main_menu.locate"),
			CallbackData: "cmd_locate",
		})
	}
	if len(utilRow) > 0 {
		rows = append(rows, utilRow)
	}

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetRobotMenu(caps *valetudo.CapabilitySet, status, flag string, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	text := fmt.Sprintf("%s\n\n• <b>%s:</b> %s", i18n.T(loc, "robot_menu.title"), i18n.T(loc, "telemetry.lbl_status"), FormatStatusDisplay(status, flag, loc))
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapBasicControl) {
		switch {
		case status == "cleaning" || status == "moving":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: i18n.T(loc, "robot_menu.btn_pause"), CallbackData: "cmd_pause"},
				{Text: i18n.T(loc, "main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "paused" || flag == "resumable":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: i18n.T(loc, "robot_menu.btn_resume"), CallbackData: "cmd_resume"},
				{Text: i18n.T(loc, "main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "returning":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: i18n.T(loc, "robot_menu.btn_pause"), CallbackData: "cmd_pause"},
				{Text: i18n.T(loc, "robot_menu.btn_resume"), CallbackData: "cmd_resume"},
			})
		default: // "docked", "idle", "error", etc.
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: i18n.T(loc, "robot_menu.btn_start"), CallbackData: "cmd_start"},
			})
		}
	}

	var statusRow []telegram.InlineKeyboardButton
	statusRow = append(statusRow, telegram.InlineKeyboardButton{
		Text:         i18n.T(loc, "robot_menu.btn_telemetry"),
		CallbackData: "cmd_telemetry",
	})
	if caps.Has(valetudo.CapConsumableMonitoring) {
		statusRow = append(statusRow, telegram.InlineKeyboardButton{
			Text:         i18n.T(loc, "robot_menu.btn_consumables"),
			CallbackData: "cmd_consumables",
		})
	}
	rows = append(rows, statusRow)

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "robot_menu.btn_resources"), CallbackData: "cmd_resources"},
	})
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "robot_menu.btn_settings"), CallbackData: "menu_settings"},
	})
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "main_menu.btn_back_main"), CallbackData: "menu_main"},
	})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetStationMenu(caps *valetudo.CapabilitySet, status string, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	if !caps.HasStation() {
		text := i18n.T(loc, "main_menu.not_supported")
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "main_menu.btn_back_main"), CallbackData: "menu_main"}},
			},
		}
		return text, markup
	}

	text := i18n.T(loc, "station_menu.title")
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapBasicControl) && status != "docked" {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "station_menu.btn_dock_home"), CallbackData: "cmd_station_home"},
		})
	}
	if caps.Has(valetudo.CapAutoEmptyDockManualTrigger) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "station_menu.btn_dock_empty"), CallbackData: "dock_empty"},
		})
	}
	if caps.Has(valetudo.CapMopDockCleanManualTrigger) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "station_menu.btn_dock_wash"), CallbackData: "dock_wash"},
		})
	}
	if caps.Has(valetudo.CapMopDockDryManualTrigger) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "station_menu.btn_dock_dry_start"), CallbackData: "dock_dry_start"},
			{Text: i18n.T(loc, "station_menu.btn_dock_dry_stop"), CallbackData: "dock_dry_stop"},
		})
	}

	var dockSettingsRow []telegram.InlineKeyboardButton
	if caps.Has(valetudo.CapMopDockMopWashTemperatureControl) {
		dockSettingsRow = append(dockSettingsRow, telegram.InlineKeyboardButton{
			Text:         i18n.T(loc, "station_menu.btn_wash_temp"),
			CallbackData: "menu_wash_temp",
		})
	}
	if caps.Has(valetudo.CapMopDockMopDryingTimeControl) {
		dockSettingsRow = append(dockSettingsRow, telegram.InlineKeyboardButton{
			Text:         i18n.T(loc, "station_menu.btn_dry_time"),
			CallbackData: "menu_dry_time",
		})
	}
	if len(dockSettingsRow) > 0 {
		rows = append(rows, dockSettingsRow)
	}

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "main_menu.btn_back_main"), CallbackData: "menu_main"},
	})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetSettingsMainMenu(caps *valetudo.CapabilitySet, hasDB bool, isAdmin bool, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	text := i18n.T(loc, "settings_menu.title")
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapOperationModeControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "settings_menu.btn_mode"), CallbackData: "sub_mode"},
		})
	}
	if caps.Has(valetudo.CapFanSpeedControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "settings_menu.btn_fan"), CallbackData: "sub_fan"},
		})
	}
	if caps.Has(valetudo.CapWaterUsageControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "settings_menu.btn_water"), CallbackData: "sub_water"},
		})
	}
	if caps.Has(valetudo.CapMopExtensionControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "settings_menu.btn_mopextend"), CallbackData: "sub_mopextend"},
		})
	}

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "settings_menu.btn_lang"), CallbackData: "sub_lang"},
	})

	if hasDB {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "settings_menu.btn_notifications"), CallbackData: "sub_notifications"},
		})
	}

	if hasDB && isAdmin {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "settings_menu.btn_users"), CallbackData: "sub_users"},
			{Text: i18n.T(loc, "settings_menu.btn_audit"), CallbackData: "sub_audit"},
		})
	}

	if isAdmin {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: i18n.T(loc, "settings_menu.btn_updates"), CallbackData: "sub_updates"},
		})
	}

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_robot"},
	})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// GetUpToDateMenu displays a message when the bot version is already up to date.
func GetUpToDateMenu(curVer string, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	text := fmt.Sprintf(i18n.T(loc, "updates.up_to_date"), curVer)
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: i18n.T(loc, "updates.btn_check_again"), CallbackData: "sub_updates"}},
			{{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"}},
		},
	}
	return text, markup
}

// GetUpdateMenu displays a message when a new version is available to install.
func GetUpdateMenu(curVer string, rel *update.ReleaseInfo, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	body := rel.Body
	if len(body) > 400 {
		body = body[:400] + "..."
	}
	if body == "" {
		body = i18n.T(loc, "updates.no_changelog")
	}

	text := fmt.Sprintf(i18n.T(loc, "updates.available_menu"), curVer, rel.Version, body)
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: i18n.T(loc, "updates.btn_update_now"), CallbackData: "action_update_bot:" + rel.Version},
				{Text: i18n.T(loc, "updates.btn_later"), CallbackData: "action_update_later:" + rel.Version},
			},
			{
				{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"},
			},
		},
	}
	return text, markup
}

func GetUsersMenu(db domain.UserRepository, currentChatID int64, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	if db == nil {
		text := "⚠️ База данных пользователей не подключена."
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"}},
			},
		}
		return text, markup
	}

	users, err := db.GetAllUsers()
	if err != nil {
		text := "❌ Ошибка получения списка пользователей: " + err.Error()
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"}},
			},
		}
		return text, markup
	}

	text := fmt.Sprintf(i18n.T(loc, "settings_menu.sub_users_title"), len(users))
	var rows [][]telegram.InlineKeyboardButton

	for _, u := range users {
		userLabel := ""
		if u.Role == database.RoleAdmin {
			userLabel = "👑 "
		} else {
			userLabel = "👤 "
		}

		if u.Username != "" {
			userLabel += "@" + u.Username
		} else {
			userLabel += fmt.Sprintf("ID: %d", u.ChatID)
		}

		if u.Role == database.RoleAdmin {
			userLabel += " (admin)"
		}

		if u.ChatID == currentChatID {
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: userLabel + " (Вы)", CallbackData: "noop"},
			})
		} else {
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: userLabel, CallbackData: "noop"},
				{Text: "🗑 Удалить", CallbackData: fmt.Sprintf("user_del:%d", u.ChatID)},
			})
		}
	}

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"},
	})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetLanguageMenu(loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	text := i18n.T(loc, "settings_menu.sub_lang_title")
	var rows [][]telegram.InlineKeyboardButton
	for _, l := range i18n.SupportedLocales() {
		btn := telegram.InlineKeyboardButton{
			Text:         i18n.LocaleName(l),
			CallbackData: "set_lang:" + string(l),
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"}})
	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetNotificationsMenu(db domain.UserRepository, chatID int64, hasStation bool, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	if db == nil {
		text := "⚠️ База данных не подключена."
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"}},
			},
		}
		return text, markup
	}

	u, err := db.GetUser(chatID)
	if err != nil || u == nil {
		u = &database.User{NotifyErrors: true, NotifyReports: true, NotifyStation: true}
	}

	formatStatus := func(enabled bool) string {
		if enabled {
			return i18n.T(loc, "notifications.status_on")
		}
		return i18n.T(loc, "notifications.status_off")
	}

	text := i18n.T(loc, "notifications.title")
	var rows [][]telegram.InlineKeyboardButton

	rows = append(rows, []telegram.InlineKeyboardButton{
		{
			Text:         fmt.Sprintf("%s: %s", i18n.T(loc, "notifications.btn_errors"), formatStatus(u.NotifyErrors)),
			CallbackData: "toggle_notify:errors",
		},
	})
	rows = append(rows, []telegram.InlineKeyboardButton{
		{
			Text:         fmt.Sprintf("%s: %s", i18n.T(loc, "notifications.btn_reports"), formatStatus(u.NotifyReports)),
			CallbackData: "toggle_notify:reports",
		},
	})
	if hasStation {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("%s: %s", i18n.T(loc, "notifications.btn_station"), formatStatus(u.NotifyStation)),
				CallbackData: "toggle_notify:station",
			},
		})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"},
	})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func FormatActionTitle(action, details string, loc i18n.Locale) string {
	switch action {
	case "start_cleaning":
		return "🚀 " + i18n.T(loc, "audit.action_start")
	case "pause_cleaning":
		return "⏸ " + i18n.T(loc, "audit.action_pause")
	case "resume_cleaning":
		return "▶️ " + i18n.T(loc, "audit.action_resume")
	case "stop_cleaning":
		return "🛑 " + i18n.T(loc, "audit.action_stop")
	case "go_home":
		return "🏠 " + i18n.T(loc, "audit.action_home")
	case "locate":
		return "📢 " + i18n.T(loc, "audit.action_locate")
	case "wizard_clean":
		if details != "" {
			return fmt.Sprintf("🪄 %s (%s)", i18n.T(loc, "audit.action_wizard"), details)
		}
		return "🪄 " + i18n.T(loc, "audit.action_wizard")
	case "set_fan":
		return fmt.Sprintf("💨 %s: <code>%s</code>", i18n.T(loc, "audit.action_fan"), details)
	case "set_water":
		return fmt.Sprintf("💧 %s: <code>%s</code>", i18n.T(loc, "audit.action_water"), details)
	case "set_mode":
		return fmt.Sprintf("🛠 %s: <code>%s</code>", i18n.T(loc, "audit.action_mode"), details)
	case "set_lang":
		return fmt.Sprintf("🌐 %s: <code>%s</code>", i18n.T(loc, "audit.action_lang"), details)
	case "set_mopextend":
		return fmt.Sprintf("🦵 %s: <code>%s</code>", i18n.T(loc, "audit.action_mopextend"), details)
	case "user_approved":
		return fmt.Sprintf("✅ %s (%s)", i18n.T(loc, "audit.action_user_approved"), details)
	case "user_rejected":
		return fmt.Sprintf("❌ %s (%s)", i18n.T(loc, "audit.action_user_rejected"), details)
	case "user_deleted":
		return fmt.Sprintf("🗑 %s (%s)", i18n.T(loc, "audit.action_user_deleted"), details)
	default:
		if details != "" {
			return fmt.Sprintf("%s (%s)", action, details)
		}
		return action
	}
}

func GetAuditLogMenu(db domain.UserRepository, page int, expandedID int64, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	if db == nil {
		text := "⚠️ База данных не подключена."
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"}},
			},
		}
		return text, markup
	}

	const pageSize = 10
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * pageSize
	logs, totalCount, err := db.GetAuditLogsPaginated(offset, pageSize)
	if err != nil || totalCount == 0 {
		text := i18n.T(loc, "audit.empty")
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"}},
			},
		}
		return text, markup
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
		offset = (page - 1) * pageSize
		logs, _, _ = db.GetAuditLogsPaginated(offset, pageSize)
	}

	pageInfo := fmt.Sprintf(i18n.T(loc, "audit.page_info"), page, totalPages)
	text := fmt.Sprintf("%s (<b>%d</b>) • <i>%s</i>\n\n", i18n.T(loc, "audit.title"), totalCount, pageInfo)

	var itemButtons []telegram.InlineKeyboardButton
	var expandedCardButton *telegram.InlineKeyboardButton

	for i, l := range logs {
		itemIndex := offset + i + 1
		userStr := "ID " + strconv.FormatInt(l.ChatID, 10)
		if l.Username != "" {
			userStr = "@" + l.Username
		}
		timeStr := l.CreatedAt.Local().Format("02.01 15:04")
		actionTitle := FormatActionTitle(l.Action, l.Details, loc)

		if l.ID == expandedID {
			timeFull := l.CreatedAt.Local().Format("02.01.2006 15:04:05")
			detailsStr := l.Details
			if detailsStr == "" {
				detailsStr = "—"
			}
			text += fmt.Sprintf("<b>%d.</b> <code>%s</code> <b>%s</b>: %s\n", itemIndex, timeStr, userStr, actionTitle)
			text += fmt.Sprintf("   ┌ 🆔 <b>ID:</b> #%d\n", l.ID)
			text += fmt.Sprintf("   ├ ⏰ <b>Время:</b> <code>%s</code>\n", timeFull)
			text += fmt.Sprintf("   ├ 👤 <b>Пользователь:</b> <code>%s</code> (ID %d)\n", userStr, l.ChatID)
			text += fmt.Sprintf("   ├ 🏷 <b>Действие:</b> <code>%s</code>\n", html.EscapeString(l.Action))
			text += fmt.Sprintf("   └ 📝 <b>Детали:</b> <code>%s</code>\n\n", html.EscapeString(detailsStr))

			itemButtons = append(itemButtons, telegram.InlineKeyboardButton{
				Text:         fmt.Sprintf("🔼 #%d", itemIndex),
				CallbackData: fmt.Sprintf("audit:%d:0", page),
			})

			cardBtn := telegram.InlineKeyboardButton{
				Text:         fmt.Sprintf("🔍 Карточка #%d", itemIndex),
				CallbackData: fmt.Sprintf("audit_card:%d:%d", l.ID, page),
			}
			expandedCardButton = &cardBtn
		} else {
			text += fmt.Sprintf("<b>%d.</b> <code>%s</code> <b>%s</b>: %s\n", itemIndex, timeStr, userStr, actionTitle)
			itemButtons = append(itemButtons, telegram.InlineKeyboardButton{
				Text:         fmt.Sprintf("#%d", itemIndex),
				CallbackData: fmt.Sprintf("audit:%d:%d", page, l.ID),
			})
		}
	}

	var keyboard [][]telegram.InlineKeyboardButton

	// Ряды кнопок действий по 5 в ряд
	for i := 0; i < len(itemButtons); i += 5 {
		end := i + 5
		if end > len(itemButtons) {
			end = len(itemButtons)
		}
		keyboard = append(keyboard, itemButtons[i:end])
	}

	// Кнопка перехода к отдельной карточке для раскрытого элемента
	if expandedCardButton != nil {
		keyboard = append(keyboard, []telegram.InlineKeyboardButton{*expandedCardButton})
	}

	// Ряд пагинации
	var navRow []telegram.InlineKeyboardButton
	if page > 1 {
		navRow = append(navRow, telegram.InlineKeyboardButton{
			Text:         "◀️",
			CallbackData: fmt.Sprintf("audit:%d:0", page-1),
		})
	}
	navRow = append(navRow, telegram.InlineKeyboardButton{
		Text:         pageInfo,
		CallbackData: fmt.Sprintf("audit:%d:%d", page, expandedID),
	})
	if page < totalPages {
		navRow = append(navRow, telegram.InlineKeyboardButton{
			Text:         "▶️",
			CallbackData: fmt.Sprintf("audit:%d:0", page+1),
		})
	}
	keyboard = append(keyboard, navRow)

	// Нижний ряд действий
	keyboard = append(keyboard, []telegram.InlineKeyboardButton{
		{Text: i18n.T(loc, "audit.btn_refresh"), CallbackData: fmt.Sprintf("audit:%d:%d", page, expandedID)},
		{Text: i18n.T(loc, "settings_menu.btn_back"), CallbackData: "menu_settings"},
	})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

func GetAuditLogCardMenu(db domain.UserRepository, id int64, page int, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	if db == nil {
		return "⚠️ База данных не подключена.", &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "audit.btn_back_list"), CallbackData: fmt.Sprintf("audit:%d:0", page)}},
			},
		}
	}

	logEntry, err := db.GetAuditLogByID(id)
	if err != nil || logEntry == nil {
		return "⚠️ Запись не найдена.", &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "audit.btn_back_list"), CallbackData: fmt.Sprintf("audit:%d:0", page)}},
			},
		}
	}

	userStr := "ID " + strconv.FormatInt(logEntry.ChatID, 10)
	if logEntry.Username != "" {
		userStr = "@" + logEntry.Username
	}
	timeFull := logEntry.CreatedAt.Local().Format("02.01.2006 15:04:05")
	detailsStr := logEntry.Details
	if detailsStr == "" {
		detailsStr = "—"
	}

	text := fmt.Sprintf(i18n.T(loc, "audit.details_header"), logEntry.ID) + "\n\n"
	text += fmt.Sprintf("⏰ <b>Время:</b> <code>%s</code>\n", timeFull)
	text += fmt.Sprintf("👤 <b>Пользователь:</b> <code>%s</code> (<code>%d</code>)\n", userStr, logEntry.ChatID)
	text += fmt.Sprintf("🎯 <b>Действие:</b> %s (<code>%s</code>)\n\n", FormatActionTitle(logEntry.Action, logEntry.Details, loc), logEntry.Action)
	text += fmt.Sprintf("📝 <b>Подробности:</b>\n<code>%s</code>\n", html.EscapeString(detailsStr))

	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: i18n.T(loc, "audit.btn_back_list"), CallbackData: fmt.Sprintf("audit:%d:%d", page, id)},
				{Text: i18n.T(loc, "audit.btn_refresh"), CallbackData: fmt.Sprintf("audit_card:%d:%d", id, page)},
			},
		},
	}
	return text, markup
}

func GetConsumablesMenu(
	caps *valetudo.CapabilitySet,
	displays []domain.ConsumableDisplayInfo,
	err error,
	loc i18n.Locale,
) (string, *telegram.InlineKeyboardMarkup) {
	if !caps.Has(valetudo.CapConsumableMonitoring) {
		text := i18n.T(loc, "main_menu.not_supported")
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "robot_menu.btn_back"), CallbackData: "menu_robot"}},
			},
		}
		return text, markup
	}

	if err != nil {
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: i18n.T(loc, "robot_menu.btn_back"), CallbackData: "menu_robot"}},
			},
		}
		return i18n.T(loc, "consumables.err_api"), markup
	}

	text := i18n.T(loc, "consumables.menu_title")
	for _, d := range displays {
		bar := RenderProgressBar(d.Percent)
		if d.IsDepleted {
			text += fmt.Sprintf("%s <b>%s:</b> %s\n• <code>[%s] 0%%</code>\n\n", d.Icon, d.Name, i18n.T(loc, "consumables.depleted"), bar)
		} else if d.IsMinutes {
			text += fmt.Sprintf(
				"%s <b>%s:</b>\n• %s\n• <code>[%s] %d%%</code>\n\n",
				d.Icon, d.Name, fmt.Sprintf(i18n.T(loc, "consumables.remaining_minutes"), d.RemainingFormatted, d.MaxH), bar, d.Percent,
			)
		} else {
			text += fmt.Sprintf(
				"%s <b>%s:</b>\n• %s\n• <code>[%s] %d%%</code>\n\n",
				d.Icon, d.Name, fmt.Sprintf(i18n.T(loc, "consumables.remaining_percent"), d.Percent), bar, d.Percent,
			)
		}
	}

	var rows [][]telegram.InlineKeyboardButton
	var currentRow []telegram.InlineKeyboardButton
	for _, d := range displays {
		btn := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf(i18n.T(loc, "consumables.btn_reset"), d.ShortName),
			CallbackData: fmt.Sprintf("reset_cons:%s:%s", d.Item.Type, d.Item.SubType),
		}
		currentRow = append(currentRow, btn)
		if len(currentRow) == 2 {
			rows = append(rows, currentRow)
			currentRow = nil
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	rows = append(rows, []telegram.InlineKeyboardButton{{Text: i18n.T(loc, "robot_menu.btn_back"), CallbackData: "menu_robot"}})
	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetRoomsMenu(rooms []domain.RoomInfo, err error, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	var text string
	if err != nil {
		text = fmt.Sprintf(i18n.T(loc, "rooms.err_get"), err.Error())
	} else {
		var lines []string
		for _, r := range rooms {
			lines = append(lines, fmt.Sprintf("• <b>%s</b> (ID: <code>%s</code>)", r.Name, r.ID))
		}
		text = fmt.Sprintf(i18n.T(loc, "rooms.title"), strings.Join(lines, "\n"))
	}
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: i18n.T(loc, "main_menu.btn_back_main"), CallbackData: "menu_main"}},
		},
	}
	return text, markup
}

func GetWizardStep1(loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	text := i18n.T(loc, "wizard.step1_title")
	modes := []struct {
		ID   string
		Name string
	}{
		{"vacuum_and_mop", i18n.T(loc, "modes.wizard_vacuum_and_mop")},
		{"vacuum_then_mop", i18n.T(loc, "modes.wizard_vacuum_then_mop")},
		{"vacuum", i18n.T(loc, "modes.wizard_vacuum")},
		{"mop", i18n.T(loc, "modes.wizard_mop")},
	}

	var rows [][]telegram.InlineKeyboardButton
	for _, m := range modes {
		btn := telegram.InlineKeyboardButton{
			Text:         m.Name,
			CallbackData: "wiz_mode:" + m.ID,
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{{Text: i18n.T(loc, "wizard.btn_cancel"), CallbackData: "wiz_cancel"}})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetWizardStep2(ws *domain.WizardSession, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	var text string
	if ws.Mode != "" {
		text = fmt.Sprintf(i18n.T(loc, "wizard.step2_title"), FormatModeTitle(ws.Mode, loc))
	} else {
		text = fmt.Sprintf(i18n.T(loc, "wizard.step2_title"), "—")
	}

	var rows [][]telegram.InlineKeyboardButton
	allSelected := len(ws.Rooms) > 0
	hasSelected := false
	for _, r := range ws.Rooms {
		icon := "◻️"
		if ws.SelectedRooms[r.ID] {
			icon = "✅"
			hasSelected = true
		} else {
			allSelected = false
		}
		btn := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", icon, r.Name),
			CallbackData: "wiz_toggle_room:" + r.ID,
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}

	var quickRow []telegram.InlineKeyboardButton
	if allSelected {
		quickRow = append(quickRow, telegram.InlineKeyboardButton{Text: i18n.T(loc, "wizard.btn_select_none"), CallbackData: "wiz_select_none"})
	} else {
		quickRow = append(quickRow, telegram.InlineKeyboardButton{Text: i18n.T(loc, "wizard.btn_select_all"), CallbackData: "wiz_select_all"})
	}
	rows = append(rows, quickRow)

	var controlRow []telegram.InlineKeyboardButton
	if hasSelected {
		controlRow = append(controlRow, telegram.InlineKeyboardButton{Text: i18n.T(loc, "wizard.btn_next"), CallbackData: "wiz_to_step3"})
	}
	controlRow = append(controlRow, telegram.InlineKeyboardButton{Text: i18n.T(loc, "wizard.btn_cancel"), CallbackData: "wiz_cancel"})
	rows = append(rows, controlRow)

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func GetWizardStep3(ws *domain.WizardSession, loc i18n.Locale) (string, *telegram.InlineKeyboardMarkup) {
	var roomNames []string
	for _, r := range ws.Rooms {
		if ws.SelectedRooms[r.ID] {
			roomNames = append(roomNames, r.Name)
		}
	}

	modeTitle := "—"
	if ws.Mode != "" {
		modeTitle = FormatModeTitle(ws.Mode, loc)
	}

	text := fmt.Sprintf(
		i18n.T(loc, "wizard.step3_title"),
		modeTitle, strings.Join(roomNames, ", "),
	)

	rows := [][]telegram.InlineKeyboardButton{
		{
			{Text: i18n.T(loc, "wizard.pass_1"), CallbackData: "wiz_iter:1"},
			{Text: i18n.T(loc, "wizard.pass_2"), CallbackData: "wiz_iter:2"},
		},
		{
			{Text: i18n.T(loc, "wizard.pass_3"), CallbackData: "wiz_iter:3"},
			{Text: i18n.T(loc, "wizard.pass_4"), CallbackData: "wiz_iter:4"},
		},
		{
			{Text: i18n.T(loc, "wizard.btn_back_to_rooms"), CallbackData: "wiz_back_to_step2"},
			{Text: i18n.T(loc, "wizard.btn_cancel"), CallbackData: "wiz_cancel"},
		},
	}

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}
