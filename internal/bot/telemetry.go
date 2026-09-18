package bot

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func renderProgressBar(percent int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := percent / 10
	empty := 10 - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func formatModeTitle(mode string) string {
	switch mode {
	case "vacuum":
		return "Только сухая"
	case "mop":
		return "Только влажная"
	case "vacuum_and_mop":
		return "Вместе (сухая + влажная)"
	case "vacuum_then_mop":
		return "Сначала сухая, затем влажная"
	default:
		return mode
	}
}

func formatDockSensor(val string) string {
	if val == "ok" {
		return "🟢 OK"
	}
	return "🔴 " + strings.ToUpper(val)
}

func getSystemUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "неизвестно"
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "неизвестно"
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "неизвестно"
	}
	d := time.Duration(sec) * time.Second
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func (b *Bot) buildTelemetryReport() string {
	batLevel := "?"
	robotStatus := "на базе"
	dockAction := ""
	currentMode := "vacuum_and_mop"

	cleanWater := "ok"
	dirtyWater := "ok"
	detergent := "ok"
	dustbag := "ok"

	if attrs, err := b.val.GetAttributes(); err == nil {
		for _, attr := range attrs {
			switch attr.Class {
			case "BatteryStateAttribute":
				batLevel = strconv.Itoa(attr.Level)
			case "StatusStateAttribute":
				if val, ok := attr.Value.(string); ok {
					robotStatus = val
				}
			case "DockStatusStateAttribute":
				if val, ok := attr.Value.(string); ok && val != "idle" && val != "none" {
					dockAction = val
				}
			case "PresetSelectionStateAttribute":
				if attr.Type == "operation_mode" {
					if val, ok := attr.Value.(string); ok {
						currentMode = val
					}
				}
			case "DockComponentStateAttribute":
				valStr, _ := attr.Value.(string)
				switch attr.Type {
				case "water_tank_clean":
					cleanWater = valStr
				case "water_tank_dirty":
					dirtyWater = valStr
				case "detergent":
					detergent = valStr
				case "dustbag":
					dustbag = valStr
				}
			}
		}
	}

	var consLines []string
	if displays, err := b.getConsumablesDisplay(); err == nil && len(displays) > 0 {
		for _, d := range displays {
			if d.IsMinutes {
				consLines = append(consLines, fmt.Sprintf("• %s: <b>%d ч</b> из %d ч (%d%%)", d.Name, d.RemainingH, d.MaxH, d.Percent))
			} else {
				consLines = append(consLines, fmt.Sprintf("• %s: <b>%d%%</b>", d.Name, d.Percent))
			}
		}
	} else {
		consLines = []string{"• Данные о расходниках недоступны"}
	}

	uptimeStr := getSystemUptime()

	statusExtra := ""
	if dockAction != "" {
		if dockAction == "drying" {
			statusExtra = " (сушка швабр)"
		} else {
			statusExtra = fmt.Sprintf(" (%s)", dockAction)
		}
	}

	lastMin, lastSec, lastArea := b.val.GetCurrentSessionStats()
	totHours, totCount, totArea := b.val.GetTotalStats()

	return fmt.Sprintf(
		"🏎 <b>БОРТОВОЙ ЖУРНАЛ DREAME X30 PRO</b>\n\n"+
			"🔋 <b>Силовая установка:</b>\n"+
			"• Заряд АКБ: <b>%s%%</b>\n"+
			"• Статус: <b>%s%s</b>\n"+
			"• Режим уборки: <b>%s</b>\n"+
			"• Аптайм Linux: <b>%s</b>\n\n"+
			"📈 <b>Сессии и налет:</b>\n"+
			"• Крайняя сессия: <b>%d мин %d с</b> | <b>%.1f м²</b>\n"+
			"• Всего выездов: <b>%d</b>\n"+
			"• Суммарный налет: <b>%d ч</b> (%.0f м²)\n\n"+
			"💧 <b>Резервуары станции:</b>\n"+
			"• Чистая вода: <b>%s</b>\n"+
			"• Грязная вода: <b>%s</b>\n"+
			"• Моющее средство: <b>%s</b>\n"+
			"• Пылесборник: <b>%s</b>\n\n"+
			"⚙️ <b>Остаточный ресурс узлов:</b>\n"+
			"%s\n\n"+
			"📊 <b>Статус:</b> Все системы в норме",
		batLevel, robotStatus, statusExtra, formatModeTitle(currentMode), uptimeStr,
		lastMin, lastSec, lastArea, totCount, totHours, totArea,
		formatDockSensor(cleanWater), formatDockSensor(dirtyWater), formatDockSensor(detergent), formatDockSensor(dustbag),
		strings.Join(consLines, "\n"),
	)
}
