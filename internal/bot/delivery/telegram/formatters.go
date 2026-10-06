package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"tgbot/internal/bot/domain"
	"tgbot/internal/bot/service/consumables"
	"tgbot/internal/i18n"
	"tgbot/internal/version"
)

func RenderProgressBar(percent int) string {
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

func FormatStatusDisplay(status, flag string, loc i18n.Locale) string {
	var icon, title string

	switch status {
	case "docked":
		icon = "🏠"
		title = i18n.T(loc, "statuses.docked")
	case "cleaning":
		switch flag {
		case "segment":
			icon = "🧹"
			title = i18n.T(loc, "statuses.cleaning_segment")
		case "zone":
			icon = "🧹"
			title = i18n.T(loc, "statuses.cleaning_zone")
		case "spot":
			icon = "🎯"
			title = i18n.T(loc, "statuses.cleaning_spot")
		case "mapping":
			icon = "🗺"
			title = i18n.T(loc, "statuses.cleaning_mapping")
		default:
			icon = "🧹"
			title = i18n.T(loc, "statuses.cleaning")
		}
	case "paused":
		icon = "⏸"
		title = i18n.T(loc, "statuses.paused")
	case "returning":
		icon = "🏠"
		title = i18n.T(loc, "statuses.returning")
	case "idle":
		icon = "💤"
		title = i18n.T(loc, "statuses.idle")
	case "moving":
		icon = "🚗"
		title = i18n.T(loc, "statuses.moving")
	case "manual_control":
		icon = "🎮"
		title = i18n.T(loc, "statuses.manual_control")
	case "error":
		icon = "🚨"
		title = i18n.T(loc, "statuses.error")
		if flag != "" && flag != "none" {
			title += " (" + flag + ")"
		}
	default:
		icon = "🤖"
		title = status
		if title == "" {
			title = i18n.T(loc, "statuses.unknown")
		}
	}

	return fmt.Sprintf("%s %s", icon, title)
}

func FormatModeTitle(mode string, loc i18n.Locale) string {
	switch mode {
	case "vacuum":
		return i18n.T(loc, "modes.vacuum")
	case "mop":
		return i18n.T(loc, "modes.mop")
	case "vacuum_and_mop":
		return i18n.T(loc, "modes.vacuum_and_mop")
	case "vacuum_then_mop":
		return i18n.T(loc, "modes.vacuum_then_mop")
	default:
		return mode
	}
}

func FormatDockSensor(val string, loc i18n.Locale) string {
	switch val {
	case "ok":
		return "🟢 " + i18n.T(loc, "telemetry.sensor_ok")
	case "missing":
		return "🟡 " + i18n.T(loc, "telemetry.sensor_missing")
	case "empty":
		return "🔴 " + i18n.T(loc, "telemetry.sensor_empty")
	case "full":
		return "🔴 " + i18n.T(loc, "telemetry.sensor_full")
	default:
		if val == "" {
			return "⚪ " + i18n.T(loc, "telemetry.status_unknown")
		}
		return "⚪ " + strings.ToUpper(val)
	}
}

func FormatReportAlert(r *domain.CleaningReport, loc i18n.Locale) string {
	if r == nil {
		return i18n.T(loc, "report.empty_alert")
	}

	rooms := r.Rooms
	runes := []rune(rooms)
	if len(runes) > 28 {
		rooms = string(runes[:25]) + "..."
	}

	return fmt.Sprintf(
		"🏁 %s\n⏱ %s: %d %s %d %s\n📐 %s: %.1f м²\n🔋 %s: %d%% ➔ %d%% (-%d%%)\n🧹 %s: %s",
		i18n.T(loc, "report.alert_title"),
		i18n.T(loc, "report.lbl_time"), r.DurationMin, i18n.T(loc, "report.min"), r.DurationSec, i18n.T(loc, "report.sec"),
		i18n.T(loc, "report.lbl_area"), r.AreaM2,
		i18n.T(loc, "report.lbl_battery"), r.StartBattery, r.EndBattery, r.BatteryUsed,
		i18n.T(loc, "report.lbl_rooms"), rooms,
	)
}

func FormatReportCaptionForChat(r *domain.CleaningReport, chatID int64, val domain.RobotClient, loc i18n.Locale) string {
	if r == nil {
		if val != nil {
			min, sec, areaM2 := val.GetCurrentSessionStats()
			if min > 0 || sec > 0 || areaM2 > 0 {
				return fmt.Sprintf(
					"🏁 <b>%s</b>\n%s\n\n⏱ <b>%s:</b> %d %s %d %s\n📐 <b>%s:</b> %.1f м²",
					i18n.T(loc, "watcher.cleaning_finished_title"),
					i18n.T(loc, "watcher.cleaning_finished_subtitle"),
					i18n.T(loc, "report.lbl_time"), min, i18n.T(loc, "report.min"), sec, i18n.T(loc, "report.sec"),
					i18n.T(loc, "report.lbl_area"), areaM2,
				)
			}
		}
		return fmt.Sprintf(
			"🏁 <b>%s</b>\n%s",
			i18n.T(loc, "watcher.cleaning_finished_title"),
			i18n.T(loc, "watcher.cleaning_finished_subtitle"),
		)
	}

	return fmt.Sprintf(
		"🏁 <b>%s</b>\n%s\n\n⏱ <b>%s:</b> %d %s %d %s\n📐 <b>%s:</b> %.1f м²\n🔋 <b>%s:</b> %d%% ➔ %d%% (-%d%%)\n🧹 <b>%s:</b> %s",
		i18n.T(loc, "watcher.cleaning_finished_title"),
		i18n.T(loc, "watcher.cleaning_finished_subtitle"),
		i18n.T(loc, "report.lbl_time"), r.DurationMin, i18n.T(loc, "report.min"), r.DurationSec, i18n.T(loc, "report.sec"),
		i18n.T(loc, "report.lbl_area"), r.AreaM2,
		i18n.T(loc, "report.lbl_battery"), r.StartBattery, r.EndBattery, r.BatteryUsed,
		i18n.T(loc, "report.lbl_rooms"), r.Rooms,
	)
}

func FormatBytes(bytes uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func FormatKB(kb uint64) string {
	return FormatBytes(kb * 1024)
}

func FormatDuration(d time.Duration, loc i18n.Locale) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60

	switch loc {
	case i18n.LocaleRU:
		if days > 0 {
			return fmt.Sprintf("%dд %dч %dм", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%dч %dм", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%dм %dс", mins, secs)
		}
		return fmt.Sprintf("%dс", secs)

	case i18n.LocaleDE:
		if days > 0 {
			return fmt.Sprintf("%dT %dStd %dMin", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%dStd %dMin", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%dMin %dSek", mins, secs)
		}
		return fmt.Sprintf("%dSek", secs)

	case i18n.LocaleZH:
		if days > 0 {
			return fmt.Sprintf("%d天 %d小时 %d分", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%d小时 %d分", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%d分 %d秒", mins, secs)
		}
		return fmt.Sprintf("%d秒", secs)

	default:
		if days > 0 {
			return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%dm %ds", mins, secs)
		}
		return fmt.Sprintf("%ds", secs)
	}
}

func FormatGCPause(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d µs", d.Microseconds())
	}
	return fmt.Sprintf("%.2f ms", float64(d.Microseconds())/1000.0)
}

func BuildResourcesReport(rStats domain.RuntimeStats, hStats domain.HostStats, loc i18n.Locale) string {
	var sb strings.Builder

	sb.WriteString(i18n.T(loc, "resources.title") + "\n\n")

	sb.WriteString(i18n.T(loc, "resources.sec_bot") + "\n")
	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b> (Sys: <b>%s</b>)\n",
		i18n.T(loc, "resources.lbl_heap"), FormatBytes(rStats.Alloc), FormatBytes(rStats.Sys)))
	sb.WriteString(fmt.Sprintf("• %s: <b>%d</b>\n", i18n.T(loc, "resources.lbl_goroutines"), rStats.NumGoroutine))
	sb.WriteString(fmt.Sprintf("• %s: <b>%d</b>\n", i18n.T(loc, "resources.lbl_objects"), rStats.HeapObjects))

	gcInfo := fmt.Sprintf(i18n.T(loc, "resources.gc_stat"), rStats.NumGC, FormatGCPause(rStats.GCPauseTotal))
	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b>\n", i18n.T(loc, "resources.lbl_gc"), gcInfo))

	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b>\n", i18n.T(loc, "resources.lbl_bot_uptime"), FormatDuration(rStats.BotUptime, loc)))
	sb.WriteString(fmt.Sprintf("• %s: <code>%s</code>\n", i18n.T(loc, "resources.lbl_bot_version"), version.Version))
	sb.WriteString(fmt.Sprintf("• %s: <code>%s (%s)</code>\n\n", i18n.T(loc, "resources.lbl_go_version"), rStats.GoVersion, rStats.Arch))

	sb.WriteString(i18n.T(loc, "resources.sec_host") + "\n")

	if hStats.HasMem {
		memBar := RenderProgressBar(hStats.MemPercent)
		sb.WriteString(fmt.Sprintf("• %s: <b>%s</b> / %s (%d%%)\n  <code>[%s] %d%%</code>\n",
			i18n.T(loc, "resources.lbl_host_mem"),
			FormatKB(hStats.MemUsedKB),
			FormatKB(hStats.MemTotalKB),
			hStats.MemPercent,
			memBar,
			hStats.MemPercent,
		))
	} else {
		sb.WriteString(fmt.Sprintf("• %s: <i>%s</i>\n", i18n.T(loc, "resources.lbl_host_mem"), i18n.T(loc, "resources.na")))
	}

	if hStats.HasLoadAvg {
		sb.WriteString(fmt.Sprintf("• %s: <b>%s, %s, %s</b>\n",
			i18n.T(loc, "resources.lbl_load_avg"),
			hStats.LoadAvg1, hStats.LoadAvg5, hStats.LoadAvg15,
		))
	} else {
		sb.WriteString(fmt.Sprintf("• %s: <i>%s</i>\n", i18n.T(loc, "resources.lbl_load_avg"), i18n.T(loc, "resources.na")))
	}

	if hStats.HasDisk {
		diskVal := fmt.Sprintf(i18n.T(loc, "resources.disk_usage"),
			FormatBytes(hStats.DiskFree),
			FormatBytes(hStats.DiskTotal),
			hStats.DiskPercent,
		)
		sb.WriteString(fmt.Sprintf("• %s: %s\n",
			fmt.Sprintf(i18n.T(loc, "resources.lbl_disk"), hStats.DiskPath),
			diskVal,
		))
	} else {
		sb.WriteString(fmt.Sprintf("• %s: <i>%s</i>\n",
			fmt.Sprintf(i18n.T(loc, "resources.lbl_disk"), "/data"),
			i18n.T(loc, "resources.na"),
		))
	}

	if hStats.HasTemp {
		sb.WriteString(fmt.Sprintf("• %s: <b>%.1f°C</b>\n", i18n.T(loc, "resources.lbl_temp"), hStats.TempCelsius))
	}

	uptimeStr := hStats.OSUptime
	if uptimeStr == "unknown" || uptimeStr == "" {
		uptimeStr = i18n.T(loc, "telemetry.status_unknown")
	}
	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b>", i18n.T(loc, "resources.lbl_os_uptime"), uptimeStr))

	return sb.String()
}

func BuildTelemetryReport(
	val domain.RobotClient,
	consSvc *consumables.Service,
	osUptime string,
	loc i18n.Locale,
	statusDisplay string,
) string {
	batLevel := "?"
	robotStatus := statusDisplay
	dockAction := ""
	currentMode := "vacuum_and_mop"

	cleanWater := "ok"
	dirtyWater := "ok"
	detergent := "ok"
	dustbag := "ok"

	if val != nil {
		if attrs, err := val.GetAttributes(); err == nil {
			for _, attr := range attrs {
				switch attr.Class {
				case "BatteryStateAttribute":
					batLevel = strconv.Itoa(attr.Level)
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
	}

	var consLines []string
	if consSvc != nil {
		if displays, err := consSvc.GetConsumablesDisplay(loc); err == nil && len(displays) > 0 {
			for _, d := range displays {
				if d.IsMinutes {
					consLines = append(consLines, fmt.Sprintf("• %s: <b>%d ч</b> из %d ч (%d%%)", d.Name, d.RemainingH, d.MaxH, d.Percent))
				} else {
					consLines = append(consLines, fmt.Sprintf("• %s: <b>%d%%</b>", d.Name, d.Percent))
				}
			}
		}
	}
	if len(consLines) == 0 {
		consLines = []string{i18n.T(loc, "telemetry.consumables_unavailable")}
	}

	uptimeStr := osUptime
	if uptimeStr == "unknown" || uptimeStr == "" {
		uptimeStr = i18n.T(loc, "telemetry.status_unknown")
	}

	statusExtra := ""
	if dockAction != "" {
		if dockAction == "drying" {
			statusExtra = i18n.T(loc, "telemetry.drying_mops")
		} else {
			statusExtra = fmt.Sprintf(" (%s)", dockAction)
		}
	}

	var lastMin, lastSec int
	var lastArea, totArea float64
	var totHours, totCount int
	if val != nil {
		lastMin, lastSec, lastArea = val.GetCurrentSessionStats()
		totHours, totCount, totArea = val.GetTotalStats()
	}

	lastTimeFormatted := fmt.Sprintf(i18n.T(loc, "telemetry.time_format"), lastMin, lastSec)

	return fmt.Sprintf(
		"%s\n\n"+
			"%s\n"+
			"• %s: <b>%s%%</b>\n"+
			"• %s: <b>%s%s</b>\n"+
			"• %s: <b>%s</b>\n"+
			"• %s: <b>%s</b>\n\n"+
			"%s\n"+
			"• %s: <b>%s</b> | <b>%.1f м²</b>\n"+
			"• %s: <b>%d</b>\n"+
			"• %s: <b>%d ч</b> (%.0f м²)\n\n"+
			"%s\n"+
			"• %s: <b>%s</b>\n"+
			"• %s: <b>%s</b>\n"+
			"• %s: <b>%s</b>\n"+
			"• %s: <b>%s</b>\n\n"+
			"%s\n"+
			"%s\n\n"+
			"%s",
		i18n.T(loc, "telemetry.header"),
		i18n.T(loc, "telemetry.sec_powertrain"),
		i18n.T(loc, "telemetry.lbl_battery"), batLevel,
		i18n.T(loc, "telemetry.lbl_status"), robotStatus, statusExtra,
		i18n.T(loc, "telemetry.lbl_mode"), FormatModeTitle(currentMode, loc),
		i18n.T(loc, "telemetry.lbl_uptime"), uptimeStr,
		i18n.T(loc, "telemetry.sec_sessions"),
		i18n.T(loc, "telemetry.lbl_last_session"), lastTimeFormatted, lastArea,
		i18n.T(loc, "telemetry.lbl_total_runs"), totCount,
		i18n.T(loc, "telemetry.lbl_total_stats"), totHours, totArea,
		i18n.T(loc, "telemetry.sec_dock_tanks"),
		i18n.T(loc, "telemetry.lbl_clean_water"), FormatDockSensor(cleanWater, loc),
		i18n.T(loc, "telemetry.lbl_dirty_water"), FormatDockSensor(dirtyWater, loc),
		i18n.T(loc, "telemetry.lbl_detergent"), FormatDockSensor(detergent, loc),
		i18n.T(loc, "telemetry.lbl_dustbag"), FormatDockSensor(dustbag, loc),
		i18n.T(loc, "telemetry.sec_consumables"),
		strings.Join(consLines, "\n"),
		i18n.T(loc, "telemetry.sec_overall"),
	)
}

func BuildHelpText(isAdmin bool, loc i18n.Locale) string {
	var sb strings.Builder
	sb.WriteString(i18n.T(loc, "help.title") + "\n\n")

	sb.WriteString(i18n.T(loc, "help.section_cleaning") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_start") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_help") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_wizard") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_clean") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_pause") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_resume") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_stop") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_home") + "\n\n")

	sb.WriteString(i18n.T(loc, "help.section_info") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_robot") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_station") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_rooms") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_map") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_telemetry") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_consumables") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_settings") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_botsettings") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_resources") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_lang") + "\n")
	sb.WriteString(i18n.T(loc, "help.cmd_locate") + "\n")

	if isAdmin {
		sb.WriteString("\n" + i18n.T(loc, "help.section_admin") + "\n")
		sb.WriteString(i18n.T(loc, "help.cmd_users") + "\n")
		sb.WriteString(i18n.T(loc, "help.cmd_audit") + "\n")
		sb.WriteString(i18n.T(loc, "help.cmd_update") + "\n")
	}

	return sb.String()
}
