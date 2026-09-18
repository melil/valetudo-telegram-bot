package bot

import (
	"fmt"
	"strings"
	"time"

	"tgbot/internal/telegram"
)

func (b *Bot) getMainMenuMarkup() *telegram.ReplyKeyboardMarkup {
	return &telegram.ReplyKeyboardMarkup{
		Keyboard: [][]string{
			{"🪄 Старт уборки", "🛑 Закончить уборку"},
			{"🤖 Робот", "🏠 Станция"},
			{"🗺 Комнаты", "📢 Найти робота"},
		},
		ResizeKeyboard: true,
	}
}

func (b *Bot) handleTextCommand(text string) {
	switch strings.TrimSpace(text) {
	case "/start", "Меню":
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "🕹 <b>Управление Dreame X30 Pro готово:</b>", b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case "/wizard", "🪄 Старт уборки":
		b.startCleaningWizard()

	case "/stop", "🛑 Закончить уборку":
		if err := b.val.TriggerAction("home"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка: "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "🛑 Уборка прервана. Робот возвращается на базу.", b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case "/robot", "🤖 Робот":
		b.sendRobotMenu(0)

	case "/station", "🏠 Станция":
		b.sendStationMenu(0)

	case "/locate", "📢 Найти робота":
		_ = b.val.TriggerLocate()
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "🔊 Подаю звуковой сигнал!", b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case "/rooms", "🗺 Комнаты":
		rooms, err := b.getRooms()
		if err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка получения комнат: "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			var lines []string
			for _, r := range rooms {
				lines = append(lines, fmt.Sprintf("• <b>%s</b> (ID: <code>%s</code>)", r.Name, r.ID))
			}
			report := "🗺 <b>Комнаты, обнаруженные в Valetudo:</b>\n\n" + strings.Join(lines, "\n") + "\n\n<i>Для точечной уборки нажмите «🪄 Старт уборки»</i>"
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, report, b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case "/settings", "⚙️ Настройки":
		text, markup := b.getSettingsMainMenu()
		_, _ = b.tg.SendPayload(telegram.SendMessagePayload{
			ChatID:              b.cfg.AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: b.cfg.IsDNDActive(),
		})

	case "/start_clean", "🚀 Вся уборка":
		if err := b.val.TriggerAction("start"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка старта: "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "🚀 Запустил генеральную уборку с настройками по умолчанию.", b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case "/pause", "⏸ Пауза":
		if err := b.val.TriggerAction("pause"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка: "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "⏸ Робот на паузе.", b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case "/home", "🏠 На базу":
		if err := b.val.TriggerAction("home"); err != nil {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка: "+err.Error(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		} else {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "🏠 Еду на док-станцию.", b.cfg.IsDNDActive(), b.getMainMenuMarkup())
		}

	case "/telemetry", "🏎 Телеметрия":
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.buildTelemetryReport(), b.cfg.IsDNDActive(), b.getMainMenuMarkup())

	case "/consumables", "🧹 Расходники":
		b.sendConsumablesMenu(0)

	default:
		_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "Команда не распознана. Воспользуйтесь меню.", b.cfg.IsDNDActive(), b.getMainMenuMarkup())
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
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "🚀 <b>Запущена генеральная уборка.</b>", markup)
		return
	case "cmd_pause":
		_ = b.val.TriggerAction("pause")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "⏸ <b>Робот на паузе.</b>", markup)
		return
	case "cmd_home":
		_ = b.val.TriggerAction("home")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "🏠 <b>Робот возвращается на базу.</b>", markup)
		return
	case "cmd_telemetry":
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
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
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "🏠 <b>Робот возвращается на базу.</b>", markup)
		return
	case "dock_empty":
		_ = b.val.TriggerCapabilityAction("AutoEmptyDockManualTriggerCapability", "trigger")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "💨 <b>Запущена выгрузка пыли в док-станцию.</b>", markup)
		return
	case "dock_wash":
		_ = b.val.TriggerCapabilityAction("MopDockCleanManualTriggerCapability", "start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "🧼 <b>Запущена стирка швабр.</b>", markup)
		return
	case "dock_dry_start":
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "start")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "♨️ <b>Запущена сушка швабр.</b>", markup)
		return
	case "dock_dry_stop":
		_ = b.val.TriggerCapabilityAction("MopDockDryManualTriggerCapability", "stop")
		markup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "❄️ <b>Сушка швабр остановлена.</b>", markup)
		return
	case "menu_wash_temp":
		text := "🌡 <b>Температура воды для стирки швабр:</b>\n<i>Dreame X30 Pro поддерживает подогрев воды до 60°C.</i>"
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: "❄️ Холодная", CallbackData: "set_wash_temp:cold"}, {Text: "🌤 Тёплая", CallbackData: "set_wash_temp:warm"}},
			{{Text: "♨️ Горячая (60°C)", CallbackData: "set_wash_temp:hot"}, {Text: "🔥 Очень горячая", CallbackData: "set_wash_temp:scalding"}},
			{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	case "menu_dry_time":
		text := "⏱ <b>Время сушки швабр горячим воздухом:</b>"
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: "2 часа", CallbackData: "set_dry_time:2h"}, {Text: "3 часа", CallbackData: "set_dry_time:3h"}},
			{{Text: "4 часа", CallbackData: "set_dry_time:4h"}, {Text: "Без нагрева", CallbackData: "set_dry_time:cold"}},
			{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}},
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
		text := "🛠 <b>Выберите тип уборки по умолчанию:</b>"
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: "Только сухая", CallbackData: "set_mode:vacuum"}, {Text: "Только влажная", CallbackData: "set_mode:mop"}},
			{{Text: "Сухая + Влажная", CallbackData: "set_mode:vacuum_and_mop"}},
			{{Text: "Сначала сухая ➡️ затем влажная", CallbackData: "set_mode:vacuum_then_mop"}},
			{{Text: "⬅️ Назад", CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_fan":
		text := "💨 <b>Выберите мощность всасывания по умолчанию:</b>"
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: "Тихо", CallbackData: "set_fan:min"}, {Text: "Стандарт", CallbackData: "set_fan:low"}},
			{{Text: "Турбо", CallbackData: "set_fan:high"}, {Text: "Макс", CallbackData: "set_fan:max"}},
			{{Text: "⬅️ Назад", CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_water":
		text := "💧 <b>Выберите влажность швабр по умолчанию:</b>"
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: "Мин", CallbackData: "set_water:min"}, {Text: "Средне", CallbackData: "set_water:medium"}, {Text: "Макс", CallbackData: "set_water:max"}},
			{{Text: "⬅️ Назад", CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_mopextend":
		text := "🦵 <b>Выдвижная лапа швабры (Dreame MopExtend™):</b>\nПозволяет роботу мыть пол вплотную к плинтусам и огибать ножки мебели."
		rows := [][]telegram.InlineKeyboardButton{
			{{Text: "✅ Включить", CallbackData: "set_mopextend:enable"}, {Text: "❌ Выключить", CallbackData: "set_mopextend:disable"}},
			{{Text: "⬅️ Назад к настройкам", CallbackData: "menu_settings"}},
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	}

	// --- Установка самих настроек ---
	if strings.HasPrefix(data, "set_mode:") {
		val := strings.TrimPrefix(data, "set_mode:")
		_ = b.val.SetOperationMode(val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "✅ Установлено: <b>"+formatModeTitle(val)+"</b>\n\n"+text, markup)
		return
	}
	if strings.HasPrefix(data, "set_fan:") {
		val := strings.TrimPrefix(data, "set_fan:")
		_ = b.val.SetPreset("FanSpeedControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "✅ Мощность установлена на: <b>"+val+"</b>\n\n"+text, markup)
		return
	}
	if strings.HasPrefix(data, "set_water:") {
		val := strings.TrimPrefix(data, "set_water:")
		_ = b.val.SetPreset("WaterUsageControlCapability", val)
		text, markup := b.getSettingsMainMenu()
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "✅ Влажность установлена на: <b>"+val+"</b>\n\n"+text, markup)
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
		status := "включена"
		if val != "enable" {
			status = "выключена"
		}
		_ = b.tg.EditMessage(b.cfg.AllowedChatID, cb.Message.MessageID, "✅ Выдвижная лапа (MopExtend) <b>"+status+"</b>.\n\n"+text, markup)
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
