package bot

import (
	"fmt"

	"tgbot/internal/telegram"
)

func (b *Bot) sendRobotMenu(msgID int) {
	text := "🤖 <b>Управление роботом</b>\nВыберите действие:"
	rows := [][]telegram.InlineKeyboardButton{
		{{Text: "🚀 Старт (Вся уборка)", CallbackData: "cmd_start"}},
		{{Text: "⏸ Пауза", CallbackData: "cmd_pause"}, {Text: "🏠 Домой", CallbackData: "cmd_home"}},
		{{Text: "🏎 Телеметрия", CallbackData: "cmd_telemetry"}, {Text: "🧹 Расходники", CallbackData: "cmd_consumables"}},
		{{Text: "⚙️ Настройки", CallbackData: "menu_settings"}},
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

func (b *Bot) sendStationMenu(msgID int) {
	text := "🏠 <b>Управление док-станцией</b>\nВыберите действие:"
	rows := [][]telegram.InlineKeyboardButton{
		{{Text: "🏠 Вернуть робота на базу", CallbackData: "cmd_station_home"}},
		{{Text: "💨 Вытряхнуть пыль", CallbackData: "dock_empty"}},
		{{Text: "🧼 Постирать швабры", CallbackData: "dock_wash"}},
		{{Text: "♨️ Старт сушки", CallbackData: "dock_dry_start"}, {Text: "❄️ Стоп сушки", CallbackData: "dock_dry_stop"}},
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
	text := "⚙️ <b>Настройки параметров по умолчанию</b>\nЭти параметры применяются при запуске <b>«🚀 Вся уборка»</b>. Выберите категорию:"
	rows := [][]telegram.InlineKeyboardButton{
		{{Text: "🛠 Тип уборки", CallbackData: "sub_mode"}},
		{{Text: "💨 Мощность всасывания", CallbackData: "sub_fan"}},
		{{Text: "💧 Влажность швабр", CallbackData: "sub_water"}},
		{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}},
	}
	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) sendConsumablesMenu(msgID int) {
	items, err := b.val.GetConsumables()
	if err != nil {
		if msgID == 0 {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка связи с Valetudo API", b.cfg.IsDNDActive(), nil)
		} else {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, "❌ Ошибка связи с Valetudo API", nil)
		}
		return
	}

	text := "🧹 <b>Состояние расходников:</b>\n\n"
	var rows [][]telegram.InlineKeyboardButton

	for _, item := range items {
		valMin := item.Remaining.Value / 60
		name := item.SubType
		var maxMin int

		switch {
		case item.Type == "brush" && item.SubType == "main":
			name = "Турбощетка"
			maxMin = 240 * 60
		case item.Type == "brush" && item.SubType == "side_right":
			name = "Боковая щетка"
			maxMin = 150 * 60
		case item.Type == "filter" && item.SubType == "main":
			name = "HEPA-фильтр"
			maxMin = 90 * 60
		case item.Type == "cleaning" && item.SubType == "sensor":
			name = "Сенсоры"
			maxMin = 30 * 60
		default:
			name = item.Type + "_" + item.SubType
			maxMin = 150 * 60
		}

		pct := (valMin * 100) / maxMin
		if pct > 100 {
			pct = 100
		}

		bar := renderProgressBar(pct)
		text += fmt.Sprintf("• <b>%s</b>: %d ч\n<code>[%s] %d%%</code>\n\n", name, valMin/60, bar, pct)

		rows = append(rows, []telegram.InlineKeyboardButton{{
			Text:         fmt.Sprintf("🔄 Сбросить: %s", name),
			CallbackData: fmt.Sprintf("reset_cons:%s:%s", item.Type, item.SubType),
		}})
	}

	rows = append(rows, []telegram.InlineKeyboardButton{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}})
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
