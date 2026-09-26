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
	var keyboard [][]string

	// Ряд 1: Управление уборкой
	var cleanRow []string
	if caps.Has(valetudo.CapMapSegmentation) {
		cleanRow = append(cleanRow, b.t("main_menu.start_cleaning"))
	} else if caps.Has(valetudo.CapBasicControl) {
		cleanRow = append(cleanRow, b.t("main_menu.full_clean"))
	}
	if caps.Has(valetudo.CapBasicControl) {
		cleanRow = append(cleanRow, b.t("main_menu.stop_cleaning"))
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

func (b *Bot) handleTextCommand(text string) {
	cleanText := strings.TrimSpace(text)
	caps := b.Caps()

	switch {
	case cleanText == "/start" || cleanText == "Меню" || cleanText == "Menu" || cleanText == "Menü" || cleanText == "菜单":
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.ready"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case cleanText == "/wizard" || i18n.Matches(cleanText, "main_menu.start_cleaning"):
		if !caps.Has(valetudo.CapMapSegmentation) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		b.startCleaningWizard()

	case cleanText == "/stop" || i18n.Matches(cleanText, "main_menu.stop_cleaning"):
		if !caps.Has(valetudo.CapBasicControl) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		if err := b.val.TriggerAction("home"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.stopped"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/robot" || i18n.Matches(cleanText, "main_menu.robot"):
		b.sendRobotMenu(0)

	case cleanText == "/station" || i18n.Matches(cleanText, "main_menu.station"):
		if !caps.HasStation() {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		b.sendStationMenu(0)

	case cleanText == "/locate" || i18n.Matches(cleanText, "main_menu.locate"):
		if !caps.Has(valetudo.CapLocate) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		_ = b.val.TriggerLocate()
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.located"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case cleanText == "/rooms" || i18n.Matches(cleanText, "main_menu.rooms"):
		if !caps.Has(valetudo.CapMapSegmentation) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		rooms, err := b.getRooms()
		if err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, fmt.Sprintf(b.t("rooms.err_get"), err.Error()), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			var lines []string
			for _, r := range rooms {
				lines = append(lines, fmt.Sprintf("• <b>%s</b> (ID: <code>%s</code>)", r.Name, r.ID))
			}
			report := fmt.Sprintf(b.t("rooms.title"), strings.Join(lines, "\n"))
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, report, b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/settings" || i18n.Matches(cleanText, "robot_menu.btn_settings"):
		text, markup := b.getSettingsMainMenu()
		_, _ = b.tg.SendPayload(telegram.SendMessagePayload{
			ChatID:              b.cfg.AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: b.cfg.IsDNDActive(),
		})

	case cleanText == "/start_clean" || i18n.Matches(cleanText, "robot_menu.btn_start") || i18n.Matches(cleanText, "main_menu.full_clean") || cleanText == "🚀 Вся уборка" || cleanText == "🚀 Full Clean":
		if !caps.Has(valetudo.CapBasicControl) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		if err := b.val.TriggerAction("start"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.started"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/pause" || i18n.Matches(cleanText, "robot_menu.btn_pause"):
		if !caps.Has(valetudo.CapBasicControl) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		if err := b.val.TriggerAction("pause"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.paused"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/home" || i18n.Matches(cleanText, "robot_menu.btn_home") || cleanText == "🏠 На базу" || cleanText == "🏠 Return Home":
		if !caps.Has(valetudo.CapBasicControl) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		if err := b.val.TriggerAction("home"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.returning"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/telemetry" || i18n.Matches(cleanText, "robot_menu.btn_telemetry"):
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.buildTelemetryReport(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case cleanText == "/consumables" || i18n.Matches(cleanText, "robot_menu.btn_consumables"):
		if !caps.Has(valetudo.CapConsumableMonitoring) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.not_supported"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
			return
		}
		b.sendConsumablesMenu(0)

	case cleanText == "/reload_caps":
		if err := b.LoadCapabilities(); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка загрузки возможностей: "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			capsList := strings.Join(b.Caps().List(), ", ")
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, fmt.Sprintf("✅ Возможности робота обновлены (%d):\n<code>%s</code>", len(b.Caps().List()), capsList), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case strings.HasPrefix(cleanText, "/lang"):
		parts := strings.Fields(cleanText)
		if len(parts) >= 2 {
			newLocale := i18n.NormalizeLocale(parts[1])
			b.SetLang(newLocale)
			msg := fmt.Sprintf(b.t("settings_menu.lang_updated"), i18n.LocaleName(newLocale))
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, msg, b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			text, markup := b.getLanguageMenu()
			_, _ = b.tg.SendPayload(telegram.SendMessagePayload{
				ChatID:              b.cfg.AllowedChatID,
				Text:                text,
				ParseMode:           "HTML",
				ReplyMarkup:         markup,
				DisableNotification: b.cfg.IsDNDActive(),
			})
		}

	default:
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.unrecognized"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
	}
}

func (b *Bot) handleCallback(cb *telegram.CallbackQuery) {
	_ = b.tg.AnswerCallbackQuery(cb.ID)

	data := cb.Data
	if data == "noop" {
		return
	}

	caps := b.Caps()

	// --- Обработка мастера уборки (Wizard) ---
	if b.handleWizardCallback(cb) {
		return
	}

	// --- Действия меню Робот ---
	switch data {
	case "menu_robot":
		b.sendRobotMenu(cb.Message.MessageID)
		return
	case "cmd_start":
		if !caps.Has(valetudo.CapBasicControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerAction("start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("robot_menu.started"), markup)
		return
	case "cmd_pause":
		if !caps.Has(valetudo.CapBasicControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerAction("pause")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("robot_menu.paused"), markup)
		return
	case "cmd_home":
		if !caps.Has(valetudo.CapBasicControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerAction("home")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("robot_menu.returning"), markup)
		return
	case "cmd_telemetry":
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.buildTelemetryReport(), markup)
		return
	case "cmd_consumables":
		b.sendConsumablesMenu(cb.Message.MessageID)
		return
	}

	// --- Управление Станцией ---
	switch data {
	case "menu_station":
		b.sendStationMenu(cb.Message.MessageID)
		return
	case "cmd_station_home":
		if !caps.Has(valetudo.CapBasicControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerAction("home")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("robot_menu.returning"), markup)
		return
	case "dock_empty":
		if !caps.Has(valetudo.CapAutoEmptyDockManualTrigger) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerCapabilityAction("AutoEmptyDockManualTriggerCapability", "trigger")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.emptying_started"), markup)
		return
	case "dock_wash":
		if !caps.Has(valetudo.CapMopDockCleanManualTrigger) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerCapabilityAction("MopDockCleanManualTriggerCapability", "start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.washing_started"), markup)
		return
	case "dock_dry_start":
		if !caps.Has(valetudo.CapMopDockDryManualTrigger) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.drying_started"), markup)
		return
	case "dock_dry_stop":
		if !caps.Has(valetudo.CapMopDockDryManualTrigger) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "stop")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.drying_stopped"), markup)
		return
	case "menu_wash_temp":
		if !caps.Has(valetudo.CapMopDockMopWashTemperatureControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("wash_temp_menu.title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("wash_temp_menu.cold"), CallbackData: "set_wash_temp:cold"}, {Text: b.t("wash_temp_menu.warm"), CallbackData: "set_wash_temp:warm"}},
			{{Text: b.t("wash_temp_menu.hot"), CallbackData: "set_wash_temp:hot"}, {Text: b.t("wash_temp_menu.scalding"), CallbackData: "set_wash_temp:scalding"}},
			{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	case "menu_dry_time":
		if !caps.Has(valetudo.CapMopDockMopDryingTimeControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("dry_time_menu.title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("dry_time_menu.2h"), CallbackData: "set_dry_time:2h"}, {Text: b.t("dry_time_menu.3h"), CallbackData: "set_dry_time:3h"}},
			{{Text: b.t("dry_time_menu.4h"), CallbackData: "set_dry_time:4h"}, {Text: b.t("dry_time_menu.cold"), CallbackData: "set_dry_time:cold"}},
			{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	}

	// --- Навигация по подменю настроек ---
	switch data {
	case "menu_settings":
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, markup)
		return

	case "sub_mode":
		if !caps.Has(valetudo.CapOperationModeControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_mode_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("modes.vacuum"), CallbackData: "set_mode:vacuum"}, {Text: b.t("modes.mop"), CallbackData: "set_mode:mop"}},
			{{Text: b.t("modes.vacuum_and_mop"), CallbackData: "set_mode:vacuum_and_mop"}},
			{{Text: b.t("modes.vacuum_then_mop"), CallbackData: "set_mode:vacuum_then_mop"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_fan":
		if !caps.Has(valetudo.CapFanSpeedControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_fan_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.fan_min"), CallbackData: "set_fan:min"}, {Text: b.t("settings_menu.fan_low"), CallbackData: "set_fan:low"}},
			{{Text: b.t("settings_menu.fan_high"), CallbackData: "set_fan:high"}, {Text: b.t("settings_menu.fan_max"), CallbackData: "set_fan:max"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_water":
		if !caps.Has(valetudo.CapWaterUsageControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_water_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.water_min"), CallbackData: "set_water:min"}, {Text: b.t("settings_menu.water_medium"), CallbackData: "set_water:medium"}, {Text: b.t("settings_menu.water_max"), CallbackData: "set_water:max"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_mopextend":
		if !caps.Has(valetudo.CapMopExtensionControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		text := b.t("settings_menu.sub_mopextend_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.enable"), CallbackData: "set_mopextend:enable"}, {Text: b.t("settings_menu.disable"), CallbackData: "set_mopextend:disable"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_lang":
		text, markup := b.getLanguageMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, markup)
		return
	}

	// --- Смена языка ---
	if strings.HasPrefix(data, "set_lang:") {
		code := strings.TrimPrefix(data, "set_lang:")
		newLocale := i18n.NormalizeLocale(code)
		b.SetLang(newLocale)

		text, markup := b.getSettingsMainMenu()
		notice := fmt.Sprintf(b.t("settings_menu.lang_updated"), i18n.LocaleName(newLocale))
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, notice+"\n\n"+text, markup)

		// Обновляем нижнюю клавиатуру на новом языке
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, notice, b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		return
	}

	// --- Установка самих настроек ---
	if strings.HasPrefix(data, "set_mode:") {
		if !caps.Has(valetudo.CapOperationModeControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_mode:")
		_ = b.val.SetOperationMode(val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, fmt.Sprintf(b.t("settings_menu.setting_updated"), b.formatModeTitle(val), text), markup)
		return
	}
	if strings.HasPrefix(data, "set_fan:") {
		if !caps.Has(valetudo.CapFanSpeedControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_fan:")
		_ = b.val.SetPreset("FanSpeedControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, fmt.Sprintf(b.t("settings_menu.setting_updated"), val, text), markup)
		return
	}
	if strings.HasPrefix(data, "set_water:") {
		if !caps.Has(valetudo.CapWaterUsageControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_water:")
		_ = b.val.SetPreset("WaterUsageControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, fmt.Sprintf(b.t("settings_menu.setting_updated"), val, text), markup)
		return
	}
	if strings.HasPrefix(data, "set_wash_temp:") {
		if !caps.Has(valetudo.CapMopDockMopWashTemperatureControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_wash_temp:")
		_ = b.val.SetMopWashTemperature(val)
		b.sendStationMenu(cb.Message.MessageID)
		return
	}
	if strings.HasPrefix(data, "set_dry_time:") {
		if !caps.Has(valetudo.CapMopDockMopDryingTimeControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_dry_time:")
		_ = b.val.SetMopDryingTime(val)
		b.sendStationMenu(cb.Message.MessageID)
		return
	}
	if strings.HasPrefix(data, "set_mopextend:") {
		if !caps.Has(valetudo.CapMopExtensionControl) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		val := strings.TrimPrefix(data, "set_mopextend:")
		_ = b.val.SetMopExtension(val == "enable")
		text, markup := b.getSettingsMainMenu()
		status := b.t("settings_menu.mopextend_enabled")
		if val != "enable" {
			status = b.t("settings_menu.mopextend_disabled")
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, fmt.Sprintf(b.t("settings_menu.setting_updated"), status, text), markup)
		return
	}

	// --- Сброс расходников ---
	if strings.HasPrefix(data, "reset_cons:") {
		if !caps.Has(valetudo.CapConsumableMonitoring) {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("main_menu.not_supported"), nil)
			return
		}
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			cType, cSubType := parts[1], parts[2]
			_ = b.val.ResetConsumable(cType, cSubType)

			// Даем Valetudo 300мс на обновление атрибутов перед отрисовкой меню
			time.Sleep(300 * time.Millisecond)
			b.sendConsumablesMenu(cb.Message.MessageID)
		}
		return
	}
}
