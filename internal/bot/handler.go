package bot

import (
	"fmt"
	"strings"
	"time"

	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func (b *Bot) getMainMenuMarkup() *telegram.ReplyKeyboardMarkup {
	caps := b.Caps()
	status, flag := b.GetRobotStatus()
	var keyboard [][]string

	// Ряд 1: Управление уборкой в зависимости от статуса робота
	var cleanRow []string
	if caps.Has(valetudo.CapBasicControl) {
		switch {
		case status == "cleaning" || status == "moving" || status == "returning":
			cleanRow = append(cleanRow, b.t("main_menu.pause_cleaning"), b.t("main_menu.stop_robot"), b.t("main_menu.go_home"))
		case status == "paused" || flag == "resumable":
			cleanRow = append(cleanRow, b.t("main_menu.resume_cleaning"), b.t("main_menu.stop_robot"), b.t("main_menu.go_home"))
		default: // "docked", "idle", "error", etc.
			if caps.Has(valetudo.CapMapSegmentation) {
				cleanRow = append(cleanRow, b.t("main_menu.start_cleaning"))
			} else {
				cleanRow = append(cleanRow, b.t("main_menu.full_clean"))
			}
		}
	}
	if len(cleanRow) > 0 {
		keyboard = append(keyboard, cleanRow)
	}

	// Ряд 2: Устройства (Робот, Станция)
	var deviceRow []string
	deviceRow = append(deviceRow, b.t("main_menu.robot"))
	if caps.HasStation() {
		deviceRow = append(deviceRow, b.t("main_menu.station"))
	}
	keyboard = append(keyboard, deviceRow)

	// Ряд 3: Дополнительно (Комнаты, Поиск робота)
	var utilRow []string
	if caps.Has(valetudo.CapMapSegmentation) {
		utilRow = append(utilRow, b.t("main_menu.rooms"))
	}
	if caps.Has(valetudo.CapLocate) {
		utilRow = append(utilRow, b.t("main_menu.locate"))
	}
	if len(utilRow) > 0 {
		keyboard = append(keyboard, utilRow)
	}

	return &telegram.ReplyKeyboardMarkup{
		Keyboard:       keyboard,
		ResizeKeyboard: true,
	}
}

func (b *Bot) handleTextCommand(msg *telegram.Message) {
	if msg == nil {
		return
	}
	// Удаляем входящее сообщение пользователя из чата, чтобы чат оставался чистым
	_ = b.tg.DeleteMessage(msg.Chat.ID, msg.MessageID)

	cleanText := strings.TrimSpace(msg.Text)
	caps := b.Caps()

	switch {
	case cleanText == "/start" || cleanText == "Меню" || cleanText == "Menu" || cleanText == "Menü" || cleanText == "菜单":
		b.sendMainDashboard()

	case cleanText == "/wizard" || i18n.Matches(cleanText, "main_menu.start_cleaning"):
		if !caps.Has(valetudo.CapMapSegmentation) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		b.startCleaningWizard()

	case cleanText == "/resume" || i18n.Matches(cleanText, "main_menu.resume_cleaning") || i18n.Matches(cleanText, "robot_menu.btn_resume"):
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("start")
		b.SetRobotStatus("cleaning", "none")
		b.sendMainDashboard()

	case cleanText == "/stop" || i18n.Matches(cleanText, "main_menu.stop_robot") || i18n.Matches(cleanText, "main_menu.stop_cleaning") || i18n.Matches(cleanText, "robot_menu.btn_stop"):
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("home")
		b.SetRobotStatus("returning", "none")
		b.sendMainDashboard()

	case cleanText == "/pause" || i18n.Matches(cleanText, "main_menu.pause_cleaning") || i18n.Matches(cleanText, "robot_menu.btn_pause"):
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("pause")
		b.SetRobotStatus("paused", "resumable")
		b.sendMainDashboard()

	case cleanText == "/home" || i18n.Matches(cleanText, "main_menu.go_home") || i18n.Matches(cleanText, "robot_menu.btn_home") || cleanText == "🏠 На базу" || cleanText == "🏠 Return Home" || cleanText == "🏠 Домой":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("home")
		b.SetRobotStatus("returning", "none")
		b.sendMainDashboard()

	case cleanText == "/robot" || i18n.Matches(cleanText, "main_menu.robot"):
		b.sendRobotMenu()

	case cleanText == "/station" || i18n.Matches(cleanText, "main_menu.station"):
		if !caps.HasStation() {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		b.sendStationMenu()

	case cleanText == "/locate" || i18n.Matches(cleanText, "main_menu.locate"):
		if !caps.Has(valetudo.CapLocate) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerLocate()
		b.sendMainDashboard()

	case cleanText == "/rooms" || i18n.Matches(cleanText, "main_menu.rooms"):
		if !caps.Has(valetudo.CapMapSegmentation) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		b.sendRoomsMenu()

	case cleanText == "/settings" || i18n.Matches(cleanText, "robot_menu.btn_settings"):
		text, markup := b.getSettingsMainMenu()
		_ = b.renderDashboard(text, markup)

	case cleanText == "/start_clean" || i18n.Matches(cleanText, "robot_menu.btn_start") || i18n.Matches(cleanText, "main_menu.full_clean") || cleanText == "🚀 Вся уборка" || cleanText == "🚀 Full Clean":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("start")
		b.SetRobotStatus("cleaning", "none")
		b.sendMainDashboard()

	case cleanText == "/telemetry" || i18n.Matches(cleanText, "robot_menu.btn_telemetry"):
		b.sendTelemetryMenu()

	case cleanText == "/consumables" || i18n.Matches(cleanText, "robot_menu.btn_consumables"):
		if !caps.Has(valetudo.CapConsumableMonitoring) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		b.sendConsumablesMenu()

	case cleanText == "/resources" || cleanText == "/res" || i18n.Matches(cleanText, "robot_menu.btn_resources"):
		b.sendResourcesMenu()

	case cleanText == "/reload_caps":
		if err := b.LoadCapabilities(); err != nil {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard("❌ Ошибка загрузки возможностей: "+err.Error(), markup)
		} else {
			b.sendMainDashboard()
		}

	case strings.HasPrefix(cleanText, "/lang"):
		parts := strings.Fields(cleanText)
		if len(parts) >= 2 {
			newLocale := i18n.NormalizeLocale(parts[1])
			b.SetLang(newLocale)
			b.sendMainDashboard()
		} else {
			text, markup := b.getLanguageMenu()
			_ = b.renderDashboard(text, markup)
		}

	default:
		b.sendMainDashboard()
	}
}

func (b *Bot) handleCallback(cb *telegram.CallbackQuery) {
	data := cb.Data
	if data == "noop" {
		_ = b.tg.AnswerCallbackQuery(cb.ID)
		return
	}

	if data == "view_last_report" {
		alertText := b.formatReportAlert(b.GetLastReport())
		_ = b.tg.AnswerCallbackQueryAlert(cb.ID, alertText, true)
		return
	}

	_ = b.tg.AnswerCallbackQuery(cb.ID)

	caps := b.Caps()

	// --- Главное меню и базовые переходы ---
	switch data {
	case "menu_main":
		b.sendMainDashboard()
		return
	case "wiz_start", "cmd_wizard":
		b.startCleaningWizard()
		return
	case "menu_rooms":
		b.sendRoomsMenu()
		return
	case "cmd_locate":
		if !caps.Has(valetudo.CapLocate) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerLocate()
		b.sendMainDashboard()
		return
	}

	// --- Обработка мастера уборки (Wizard) ---
	if b.handleWizardCallback(cb) {
		return
	}

	// --- Действия меню Робот ---
	switch data {
	case "menu_robot":
		b.sendRobotMenu()
		return
	case "cmd_start":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("start")
		b.SetRobotStatus("cleaning", "none")
		b.StartSession(nil, b.getBatteryLevel())
		b.sendMainDashboard()
		return
	case "cmd_resume":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("start")
		b.SetRobotStatus("cleaning", "none")
		b.sendMainDashboard()
		return
	case "cmd_pause":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("pause")
		b.SetRobotStatus("paused", "resumable")
		b.sendMainDashboard()
		return
	case "cmd_stop":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("home")
		b.SetRobotStatus("returning", "none")
		b.sendMainDashboard()
		return
	case "cmd_home":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("home")
		b.SetRobotStatus("returning", "none")
		b.sendMainDashboard()
		return
	case "cmd_telemetry":
		b.sendTelemetryMenu()
		return
	case "cmd_consumables":
		b.sendConsumablesMenu()
		return
	case "cmd_resources", "cmd_resources_refresh":
		b.sendResourcesMenu()
		return
	}

	// --- Управление Станцией ---
	switch data {
	case "menu_station":
		b.sendStationMenu()
		return
	case "cmd_station_home":
		if !caps.Has(valetudo.CapBasicControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerAction("home")
		b.SetRobotStatus("returning", "none")
		b.sendStationMenu()
		return
	case "dock_empty":
		if !caps.Has(valetudo.CapAutoEmptyDockManualTrigger) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerCapabilityAction("AutoEmptyDockManualTriggerCapability", "trigger")
		b.sendStationMenu()
		return
	case "dock_wash":
		if !caps.Has(valetudo.CapMopDockCleanManualTrigger) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerCapabilityAction("MopDockCleanManualTriggerCapability", "start")
		b.sendStationMenu()
		return
	case "dock_dry_start":
		if !caps.Has(valetudo.CapMopDockDryManualTrigger) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "start")
		b.sendStationMenu()
		return
	case "dock_dry_stop":
		if !caps.Has(valetudo.CapMopDockDryManualTrigger) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "stop")
		b.sendStationMenu()
		return
	case "menu_wash_temp":
		if !caps.Has(valetudo.CapMopDockMopWashTemperatureControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		text := b.t("wash_temp_menu.title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("wash_temp_menu.cold"), CallbackData: "set_wash_temp:cold"}, {Text: b.t("wash_temp_menu.warm"), CallbackData: "set_wash_temp:warm"}},
			{{Text: b.t("wash_temp_menu.hot"), CallbackData: "set_wash_temp:hot"}, {Text: b.t("wash_temp_menu.scalding"), CallbackData: "set_wash_temp:scalding"}},
			{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
		}
		_ = b.renderDashboard(text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	case "menu_dry_time":
		if !caps.Has(valetudo.CapMopDockMopDryingTimeControl) {
			markup := &telegram.InlineKeyboardMarkup{
				InlineKeyboard: [][]telegram.InlineKeyboardButton{
					{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
				},
			}
			_ = b.renderDashboard(b.t("main_menu.not_supported"), markup)
			return
		}
		text := b.t("dry_time_menu.title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("dry_time_menu.2h"), CallbackData: "set_dry_time:2h"}, {Text: b.t("dry_time_menu.3h"), CallbackData: "set_dry_time:3h"}},
			{{Text: b.t("dry_time_menu.4h"), CallbackData: "set_dry_time:4h"}, {Text: b.t("dry_time_menu.cold"), CallbackData: "set_dry_time:cold"}},
			{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
		}
		_ = b.renderDashboard(text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	}

	// --- Навигация по подменю настроек ---
	switch data {
	case "menu_settings":
		text, markup := b.getSettingsMainMenu()
		_ = b.renderDashboard(text, markup)
		return

	case "sub_mode":
		if !caps.Has(valetudo.CapOperationModeControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_mode_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("modes.vacuum"), CallbackData: "set_mode:vacuum"}, {Text: b.t("modes.mop"), CallbackData: "set_mode:mop"}},
			{{Text: b.t("modes.vacuum_and_mop"), CallbackData: "set_mode:vacuum_and_mop"}},
			{{Text: b.t("modes.vacuum_then_mop"), CallbackData: "set_mode:vacuum_then_mop"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.renderDashboard(text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_fan":
		if !caps.Has(valetudo.CapFanSpeedControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_fan_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.fan_min"), CallbackData: "set_fan:min"}, {Text: b.t("settings_menu.fan_low"), CallbackData: "set_fan:low"}},
			{{Text: b.t("settings_menu.fan_high"), CallbackData: "set_fan:high"}, {Text: b.t("settings_menu.fan_max"), CallbackData: "set_fan:max"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.renderDashboard(text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_water":
		if !caps.Has(valetudo.CapWaterUsageControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_water_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.water_min"), CallbackData: "set_water:min"}, {Text: b.t("settings_menu.water_medium"), CallbackData: "set_water:medium"}, {Text: b.t("settings_menu.water_max"), CallbackData: "set_water:max"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.renderDashboard(text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_mopextend":
		if !caps.Has(valetudo.CapMopExtensionControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_mopextend_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.enable"), CallbackData: "set_mopextend:enable"}, {Text: b.t("settings_menu.disable"), CallbackData: "set_mopextend:disable"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.renderDashboard(text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_lang":
		text, markup := b.getLanguageMenu()
		_ = b.renderDashboard(text, markup)
		return
	}

	// --- Смена языка ---
	if strings.HasPrefix(data, "set_lang:") {
		code := strings.TrimPrefix(data, "set_lang:")
		newLocale := i18n.NormalizeLocale(code)
		b.SetLang(newLocale)

		text, markup := b.getSettingsMainMenu()
		notice := fmt.Sprintf(b.t("settings_menu.lang_updated"), i18n.LocaleName(newLocale))
		_ = b.renderDashboard(notice+"\n\n"+text, markup)
		return
	}

	// --- Установка настроек ---
	if strings.HasPrefix(data, "set_mode:") {
		if !caps.Has(valetudo.CapOperationModeControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_mode:")
		_ = b.val.SetOperationMode(val)
		text, markup := b.getSettingsMainMenu()
		_ = b.renderDashboard(fmt.Sprintf(b.t("settings_menu.setting_updated"), b.formatModeTitle(val), text), markup)
		return
	}
	if strings.HasPrefix(data, "set_fan:") {
		if !caps.Has(valetudo.CapFanSpeedControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_fan:")
		_ = b.val.SetPreset("FanSpeedControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.renderDashboard(fmt.Sprintf(b.t("settings_menu.setting_updated"), val, text), markup)
		return
	}
	if strings.HasPrefix(data, "set_water:") {
		if !caps.Has(valetudo.CapWaterUsageControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_water:")
		_ = b.val.SetPreset("WaterUsageControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.renderDashboard(fmt.Sprintf(b.t("settings_menu.setting_updated"), val, text), markup)
		return
	}
	if strings.HasPrefix(data, "set_wash_temp:") {
		if !caps.Has(valetudo.CapMopDockMopWashTemperatureControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_wash_temp:")
		_ = b.val.SetMopWashTemperature(val)
		b.sendStationMenu()
		return
	}
	if strings.HasPrefix(data, "set_dry_time:") {
		if !caps.Has(valetudo.CapMopDockMopDryingTimeControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_dry_time:")
		_ = b.val.SetMopDryingTime(val)
		b.sendStationMenu()
		return
	}
	if strings.HasPrefix(data, "set_mopextend:") {
		if !caps.Has(valetudo.CapMopExtensionControl) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_mopextend:")
		_ = b.val.SetMopExtension(val == "enable")
		text, markup := b.getSettingsMainMenu()
		status := b.t("settings_menu.mopextend_enabled")
		if val != "enable" {
			status = b.t("settings_menu.mopextend_disabled")
		}
		_ = b.renderDashboard(fmt.Sprintf(b.t("settings_menu.setting_updated"), status, text), markup)
		return
	}

	// --- Сброс расходников ---
	if strings.HasPrefix(data, "reset_cons:") {
		if !caps.Has(valetudo.CapConsumableMonitoring) {
			_ = b.renderDashboard(b.t("main_menu.not_supported"), nil)
			return
		}
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			cType, cSubType := parts[1], parts[2]
			_ = b.val.ResetConsumable(cType, cSubType)

			time.Sleep(300 * time.Millisecond)
			b.sendConsumablesMenu()
		}
		return
	}
}
