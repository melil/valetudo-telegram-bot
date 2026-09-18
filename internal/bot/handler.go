package bot

import (
	"fmt"
	"strings"
	"time"

	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
)

func (b *Bot) getMainMenuMarkup() *telegram.ReplyKeyboardMarkup {
	return &telegram.ReplyKeyboardMarkup{
		Keyboard: [][]string{
			{b.t("main_menu.start_cleaning"), b.t("main_menu.stop_cleaning")},
			{b.t("main_menu.robot"), b.t("main_menu.station")},
			{b.t("main_menu.rooms"), b.t("main_menu.locate")},
		},
		ResizeKeyboard: true,
	}
}

func (b *Bot) handleTextCommand(text string) {
	cleanText := strings.TrimSpace(text)
	switch {
	case cleanText == "/start" || cleanText == "Меню" || cleanText == "Menu" || cleanText == "Menü" || cleanText == "菜单":
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("main_menu.ready"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case cleanText == "/wizard" || i18n.Matches(cleanText, "main_menu.start_cleaning"):
		b.startCleaningWizard()

	case cleanText == "/stop" || i18n.Matches(cleanText, "main_menu.stop_cleaning"):
		if err := b.val.TriggerAction("home"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.stopped"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/robot" || i18n.Matches(cleanText, "main_menu.robot"):
		b.sendRobotMenu(0)

	case cleanText == "/station" || i18n.Matches(cleanText, "main_menu.station"):
		b.sendStationMenu(0)

	case cleanText == "/locate" || i18n.Matches(cleanText, "main_menu.locate"):
		_ = b.val.TriggerLocate()
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.located"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case cleanText == "/rooms" || i18n.Matches(cleanText, "main_menu.rooms"):
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

	case cleanText == "/start_clean" || i18n.Matches(cleanText, "robot_menu.btn_start") || cleanText == "🚀 Вся уборка" || cleanText == "🚀 Full Clean":
		if err := b.val.TriggerAction("start"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.started"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/pause" || i18n.Matches(cleanText, "robot_menu.btn_pause"):
		if err := b.val.TriggerAction("pause"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.paused"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/home" || i18n.Matches(cleanText, "robot_menu.btn_home") || cleanText == "🏠 На базу" || cleanText == "🏠 Return Home":
		if err := b.val.TriggerAction("home"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("robot_menu.returning"), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case cleanText == "/telemetry" || i18n.Matches(cleanText, "robot_menu.btn_telemetry"):
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.buildTelemetryReport(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case cleanText == "/consumables" || i18n.Matches(cleanText, "robot_menu.btn_consumables"):
		b.sendConsumablesMenu(0)

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
		_ = b.val.TriggerAction("start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("robot_menu.started"), markup)
		return
	case "cmd_pause":
		_ = b.val.TriggerAction("pause")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("robot_menu.paused"), markup)
		return
	case "cmd_home":
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
		_ = b.val.TriggerAction("home")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("robot_menu.returning"), markup)
		return
	case "dock_empty":
		_ = b.val.TriggerCapabilityAction("AutoEmptyDockManualTriggerCapability", "trigger")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.emptying_started"), markup)
		return
	case "dock_wash":
		_ = b.val.TriggerCapabilityAction("MopDockCleanManualTriggerCapability", "start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.washing_started"), markup)
		return
	case "dock_dry_start":
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.drying_started"), markup)
		return
	case "dock_dry_stop":
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "stop")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, b.t("station_menu.drying_stopped"), markup)
		return
	case "menu_wash_temp":
		text := b.t("wash_temp_menu.title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("wash_temp_menu.cold"), CallbackData: "set_wash_temp:cold"}, {Text: b.t("wash_temp_menu.warm"), CallbackData: "set_wash_temp:warm"}},
			{{Text: b.t("wash_temp_menu.hot"), CallbackData: "set_wash_temp:hot"}, {Text: b.t("wash_temp_menu.scalding"), CallbackData: "set_wash_temp:scalding"}},
			{{Text: b.t("station_menu.btn_back"), CallbackData: "menu_station"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	case "menu_dry_time":
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
		text := b.t("settings_menu.sub_fan_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.fan_min"), CallbackData: "set_fan:min"}, {Text: b.t("settings_menu.fan_low"), CallbackData: "set_fan:low"}},
			{{Text: b.t("settings_menu.fan_high"), CallbackData: "set_fan:high"}, {Text: b.t("settings_menu.fan_max"), CallbackData: "set_fan:max"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_water":
		text := b.t("settings_menu.sub_water_title")
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: b.t("settings_menu.water_min"), CallbackData: "set_water:min"}, {Text: b.t("settings_menu.water_medium"), CallbackData: "set_water:medium"}, {Text: b.t("settings_menu.water_max"), CallbackData: "set_water:max"}},
			{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_mopextend":
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
		val := strings.TrimPrefix(data, "set_mode:")
		_ = b.val.SetOperationMode(val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, fmt.Sprintf(b.t("settings_menu.setting_updated"), b.formatModeTitle(val), text), markup)
		return
	}
	if strings.HasPrefix(data, "set_fan:") {
		val := strings.TrimPrefix(data, "set_fan:")
		_ = b.val.SetPreset("FanSpeedControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, fmt.Sprintf(b.t("settings_menu.setting_updated"), val, text), markup)
		return
	}
	if strings.HasPrefix(data, "set_water:") {
		val := strings.TrimPrefix(data, "set_water:")
		_ = b.val.SetPreset("WaterUsageControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, fmt.Sprintf(b.t("settings_menu.setting_updated"), val, text), markup)
		return
	}
	if strings.HasPrefix(data, "set_wash_temp:") {
		val := strings.TrimPrefix(data, "set_wash_temp:")
		_ = b.val.SetMopWashTemperature(val)
		b.sendStationMenu(cb.Message.MessageID)
		return
	}
	if strings.HasPrefix(data, "set_dry_time:") {
		val := strings.TrimPrefix(data, "set_dry_time:")
		_ = b.val.SetMopDryingTime(val)
		b.sendStationMenu(cb.Message.MessageID)
		return
	}
	if strings.HasPrefix(data, "set_mopextend:") {
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
