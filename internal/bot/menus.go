package bot

import (
	"fmt"

	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func (b *Bot) sendRobotMenu(msgID int) {
	text := b.t("robot_menu.title")
	caps := b.Caps()
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapBasicControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("robot_menu.btn_start"), CallbackData: "cmd_start"},
		})
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("robot_menu.btn_pause"), CallbackData: "cmd_pause"},
			{Text: b.t("robot_menu.btn_home"), CallbackData: "cmd_home"},
		})
	}

	var statusRow []telegram.InlineKeyboardButton
	statusRow = append(statusRow, telegram.InlineKeyboardButton{
		Text:         b.t("robot_menu.btn_telemetry"),
		CallbackData: "cmd_telemetry",
	})
	if caps.Has(valetudo.CapConsumableMonitoring) {
		statusRow = append(statusRow, telegram.InlineKeyboardButton{
			Text:         b.t("robot_menu.btn_consumables"),
			CallbackData: "cmd_consumables",
		})
	}
	rows = append(rows, statusRow)

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: b.t("robot_menu.btn_settings"), CallbackData: "menu_settings"},
	})

	markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}

	if msgID == 0 {
		_, _ = b.tg.SendPayload(telegram.SendMessagePayload{
			ChatID:              b.cfg.AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: b.cfg.IsDNDActive(),
		})
	} else {
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, text, markup)
	}
}

func (b *Bot) sendStationMenu(msgID int) {
	caps := b.Caps()
	if !caps.HasStation() {
		text := b.t("main_menu.not_supported")
		if msgID == 0 {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, text, b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, text, nil)
		}
		return
	}

	text := b.t("station_menu.title")
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapBasicControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("station_menu.btn_dock_home"), CallbackData: "cmd_station_home"},
		})
	}
	if caps.Has(valetudo.CapAutoEmptyDockManualTrigger) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("station_menu.btn_dock_empty"), CallbackData: "dock_empty"},
		})
	}
	if caps.Has(valetudo.CapMopDockCleanManualTrigger) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("station_menu.btn_dock_wash"), CallbackData: "dock_wash"},
		})
	}
	if caps.Has(valetudo.CapMopDockDryManualTrigger) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("station_menu.btn_dock_dry_start"), CallbackData: "dock_dry_start"},
			{Text: b.t("station_menu.btn_dock_dry_stop"), CallbackData: "dock_dry_stop"},
		})
	}

	var dockSettingsRow []telegram.InlineKeyboardButton
	if caps.Has(valetudo.CapMopDockMopWashTemperatureControl) {
		dockSettingsRow = append(dockSettingsRow, telegram.InlineKeyboardButton{
			Text:         b.t("station_menu.btn_wash_temp"),
			CallbackData: "menu_wash_temp",
		})
	}
	if caps.Has(valetudo.CapMopDockMopDryingTimeControl) {
		dockSettingsRow = append(dockSettingsRow, telegram.InlineKeyboardButton{
			Text:         b.t("station_menu.btn_dry_time"),
			CallbackData: "menu_dry_time",
		})
	}
	if len(dockSettingsRow) > 0 {
		rows = append(rows, dockSettingsRow)
	}

	markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}

	if msgID == 0 {
		_, _ = b.tg.SendPayload(telegram.SendMessagePayload{
			ChatID:              b.cfg.AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: b.cfg.IsDNDActive(),
		})
	} else {
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, text, markup)
	}
}

func (b *Bot) getSettingsMainMenu() (string, *telegram.InlineKeyboardMarkup) {
	text := b.t("settings_menu.title")
	caps := b.Caps()
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapOperationModeControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("settings_menu.btn_mode"), CallbackData: "sub_mode"},
		})
	}
	if caps.Has(valetudo.CapFanSpeedControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("settings_menu.btn_fan"), CallbackData: "sub_fan"},
		})
	}
	if caps.Has(valetudo.CapWaterUsageControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("settings_menu.btn_water"), CallbackData: "sub_water"},
		})
	}
	if caps.Has(valetudo.CapMopExtensionControl) {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: b.t("settings_menu.btn_mopextend"), CallbackData: "sub_mopextend"},
		})
	}

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: b.t("settings_menu.btn_lang"), CallbackData: "sub_lang"},
	})
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_robot"},
	})

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) getLanguageMenu() (string, *telegram.InlineKeyboardMarkup) {
	text := b.t("settings_menu.sub_lang_title")
	var rows [][]telegram.InlineKeyboardButton
	for _, loc := range i18n.SupportedLocales() {
		btn := telegram.InlineKeyboardButton{
			Text:         i18n.LocaleName(loc),
			CallbackData: "set_lang:" + string(loc),
		}
		rows = append(rows, []telegram.InlineKeyboardButton{btn})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{{Text: b.t("settings_menu.btn_back"), CallbackData: "menu_settings"}})
	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) sendConsumablesMenu(msgID int) {
	if !b.Caps().Has(valetudo.CapConsumableMonitoring) {
		text := b.t("main_menu.not_supported")
		if msgID == 0 {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, text, b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, text, nil)
		}
		return
	}

	displays, err := b.getConsumablesDisplay()
	if err != nil {
		if msgID == 0 {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("consumables.err_api"), b.cfg.IsDNDActive(), nil)
		} else {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, b.t("consumables.err_api"), nil)
		}
		return
	}

	text := b.t("consumables.menu_title")
	for _, d := range displays {
		bar := renderProgressBar(d.Percent)
		if d.IsDepleted {
			text += fmt.Sprintf("%s <b>%s:</b> %s\n• <code>[%s] 0%%</code>\n\n", d.Icon, d.Name, b.t("consumables.depleted"), bar)
		} else if d.IsMinutes {
			text += fmt.Sprintf(
				"%s <b>%s:</b>\n• %s\n• <code>[%s] %d%%</code>\n\n",
				d.Icon, d.Name, fmt.Sprintf(b.t("consumables.remaining_minutes"), d.RemainingFormatted, d.MaxH), bar, d.Percent,
			)
		} else {
			text += fmt.Sprintf(
				"%s <b>%s:</b>\n• %s\n• <code>[%s] %d%%</code>\n\n",
				d.Icon, d.Name, fmt.Sprintf(b.t("consumables.remaining_percent"), d.Percent), bar, d.Percent,
			)
		}
	}

	// Кнопки сброса в 2 компактные колонки
	var rows [][]telegram.InlineKeyboardButton
	var currentRow []telegram.InlineKeyboardButton
	for _, d := range displays {
		btn := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf(b.t("consumables.btn_reset"), d.ShortName),
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

	rows = append(rows, []telegram.InlineKeyboardButton{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}})
	markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}

	if msgID == 0 {
		_, _ = b.tg.SendPayload(telegram.SendMessagePayload{
			ChatID:              b.cfg.AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: b.cfg.IsDNDActive(),
		})
	} else {
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, text, markup)
	}
}
