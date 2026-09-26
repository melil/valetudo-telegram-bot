package bot

import (
	"fmt"
	"strings"

	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func (b *Bot) getMainDashboard() (string, *telegram.InlineKeyboardMarkup) {
	caps := b.Caps()
	status, flag := b.GetRobotStatus()

	batStr := ""
	if attrs, err := b.val.GetAttributes(); err == nil {
		for _, attr := range attrs {
			if attr.Class == "BatteryStateAttribute" {
				batStr = fmt.Sprintf(" | 🔋 <b>%d%%</b>", attr.Level)
				break
			}
		}
	}

	statusDisplay := b.formatStatusDisplay(status, flag)

	text := fmt.Sprintf("%s\n\n• <b>%s:</b> %s%s",
		b.t("main_menu.ready"),
		b.t("telemetry.lbl_status"),
		statusDisplay,
		batStr,
	)

	var rows [][]telegram.InlineKeyboardButton

	// Ряд 1: Контекстное управление уборкой в зависимости от статуса
	if caps.Has(valetudo.CapBasicControl) {
		switch {
		case status == "cleaning" || status == "moving":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: b.t("main_menu.pause_cleaning"), CallbackData: "cmd_pause"},
				{Text: b.t("main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "paused" || flag == "resumable":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: b.t("main_menu.resume_cleaning"), CallbackData: "cmd_resume"},
				{Text: b.t("main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "returning":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: b.t("main_menu.pause_cleaning"), CallbackData: "cmd_pause"},
				{Text: b.t("main_menu.resume_cleaning"), CallbackData: "cmd_resume"},
			})
		default: // "docked", "idle", "error", etc.
			var defaultRow []telegram.InlineKeyboardButton
			if caps.Has(valetudo.CapMapSegmentation) {
				defaultRow = append(defaultRow, telegram.InlineKeyboardButton{
					Text: b.t("main_menu.start_cleaning"), CallbackData: "wiz_start",
				})
			} else {
				defaultRow = append(defaultRow, telegram.InlineKeyboardButton{
					Text: b.t("main_menu.full_clean"), CallbackData: "cmd_start",
				})
			}
			if status != "docked" {
				defaultRow = append(defaultRow, telegram.InlineKeyboardButton{
					Text: b.t("main_menu.go_home"), CallbackData: "cmd_home",
				})
			}
			if len(defaultRow) > 0 {
				rows = append(rows, defaultRow)
			}
		}
	}

	// Ряд 2: Устройства (Робот, Станция)
	var deviceRow []telegram.InlineKeyboardButton
	deviceRow = append(deviceRow, telegram.InlineKeyboardButton{
		Text:         b.t("main_menu.robot"),
		CallbackData: "menu_robot",
	})
	if caps.HasStation() {
		deviceRow = append(deviceRow, telegram.InlineKeyboardButton{
			Text:         b.t("main_menu.station"),
			CallbackData: "menu_station",
		})
	}
	rows = append(rows, deviceRow)

	// Ряд 3: Дополнительно (Комнаты, Поиск робота)
	var utilRow []telegram.InlineKeyboardButton
	if caps.Has(valetudo.CapMapSegmentation) {
		utilRow = append(utilRow, telegram.InlineKeyboardButton{
			Text:         b.t("main_menu.rooms"),
			CallbackData: "menu_rooms",
		})
	}
	if caps.Has(valetudo.CapLocate) {
		utilRow = append(utilRow, telegram.InlineKeyboardButton{
			Text:         b.t("main_menu.locate"),
			CallbackData: "cmd_locate",
		})
	}
	if len(utilRow) > 0 {
		rows = append(rows, utilRow)
	}

	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) sendMainDashboard() {
	text, markup := b.getMainDashboard()
	_ = b.renderDashboard(text, markup)
}

func (b *Bot) sendRobotMenu() {
	caps := b.Caps()
	status, flag := b.GetRobotStatus()
	text := fmt.Sprintf("%s\n\n• <b>%s:</b> %s", b.t("robot_menu.title"), b.t("telemetry.lbl_status"), b.formatStatusDisplay(status, flag))
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapBasicControl) {
		switch {
		case status == "cleaning" || status == "moving":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: b.t("robot_menu.btn_pause"), CallbackData: "cmd_pause"},
				{Text: b.t("main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "paused" || flag == "resumable":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: b.t("robot_menu.btn_resume"), CallbackData: "cmd_resume"},
				{Text: b.t("main_menu.stop_cleaning"), CallbackData: "cmd_stop"},
			})
		case status == "returning":
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: b.t("robot_menu.btn_pause"), CallbackData: "cmd_pause"},
				{Text: b.t("robot_menu.btn_resume"), CallbackData: "cmd_resume"},
			})
		default: // "docked", "idle", "error", etc.
			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: b.t("robot_menu.btn_start"), CallbackData: "cmd_start"},
			})
		}
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
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"},
	})

	markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
	_ = b.renderDashboard(text, markup)
}

func (b *Bot) sendStationMenu() {
	caps := b.Caps()
	if !caps.HasStation() {
		text := b.t("main_menu.not_supported")
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
			},
		}
		_ = b.renderDashboard(text, markup)
		return
	}

	status, _ := b.GetRobotStatus()
	text := b.t("station_menu.title")
	var rows [][]telegram.InlineKeyboardButton

	if caps.Has(valetudo.CapBasicControl) && status != "docked" {
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

	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"},
	})

	markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
	_ = b.renderDashboard(text, markup)
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

func (b *Bot) sendConsumablesMenu() {
	if !b.Caps().Has(valetudo.CapConsumableMonitoring) {
		text := b.t("main_menu.not_supported")
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}},
			},
		}
		_ = b.renderDashboard(text, markup)
		return
	}

	displays, err := b.getConsumablesDisplay()
	if err != nil {
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}},
			},
		}
		_ = b.renderDashboard(b.t("consumables.err_api"), markup)
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
	_ = b.renderDashboard(text, markup)
}

func (b *Bot) sendRoomsMenu() {
	rooms, err := b.getRooms()
	var text string
	if err != nil {
		text = fmt.Sprintf(b.t("rooms.err_get"), err.Error())
	} else {
		var lines []string
		for _, r := range rooms {
			lines = append(lines, fmt.Sprintf("• <b>%s</b> (ID: <code>%s</code>)", r.Name, r.ID))
		}
		text = fmt.Sprintf(b.t("rooms.title"), strings.Join(lines, "\n"))
	}
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: b.t("main_menu.btn_back_main"), CallbackData: "menu_main"}},
		},
	}
	_ = b.renderDashboard(text, markup)
}

func (b *Bot) sendTelemetryMenu() {
	text := b.buildTelemetryReport()
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"}},
		},
	}
	_ = b.renderDashboard(text, markup)
}
