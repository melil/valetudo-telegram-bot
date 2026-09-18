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
		{{Text: "🌡 Температура стирки", CallbackData: "menu_wash_temp"}, {Text: "⏱ Время сушки", CallbackData: "menu_dry_time"}},
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
		{{Text: "🦵 Выдвижная швабра (MopExtend)", CallbackData: "sub_mopextend"}},
		{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}},
	}
	return text, &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (b *Bot) sendConsumablesMenu(msgID int) {
	displays, err := b.getConsumablesDisplay()
	if err != nil {
		if msgID == 0 {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "❌ Ошибка связи с Valetudo API", b.cfg.IsDNDActive(), nil)
		} else {
			_ = b.tg.EditMessage(b.cfg.AllowedChatID, msgID, "❌ Ошибка связи с Valetudo API", nil)
		}
		return
	}

	text := "🧹 <b>Состояние расходников:</b>\n\n"
	for _, d := range displays {
		bar := renderProgressBar(d.Percent)
		if d.IsDepleted {
			text += fmt.Sprintf("%s <b>%s:</b> ⚠️ <b>Ресурс исчерпан!</b>\n• <code>[%s] 0%%</code>\n\n", d.Icon, d.Name, bar)
		} else if d.IsMinutes {
			text += fmt.Sprintf(
				"%s <b>%s:</b>\n• Осталось: <b>%s</b> из %d ч\n• <code>[%s] %d%%</code>\n\n",
				d.Icon, d.Name, d.RemainingFormatted, d.MaxH, bar, d.Percent,
			)
		} else {
			text += fmt.Sprintf(
				"%s <b>%s:</b>\n• Осталось: <b>%d%%</b>\n• <code>[%s] %d%%</code>\n\n",
				d.Icon, d.Name, d.Percent, bar, d.Percent,
			)
		}
	}

	// Кнопки сброса в 2 компактные колонки
	var rows [][]telegram.InlineKeyboardButton
	var currentRow []telegram.InlineKeyboardButton
	for _, d := range displays {
		btn := telegram.InlineKeyboardButton{
			Text:         fmt.Sprintf("🔄 %s", d.ShortName),
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
