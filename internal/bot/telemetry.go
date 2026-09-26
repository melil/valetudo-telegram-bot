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

func (b *Bot) formatModeTitle(mode string) string {
	switch mode {
	case "vacuum":
		return b.t("modes.vacuum")
	case "mop":
		return b.t("modes.mop")
	case "vacuum_and_mop":
		return b.t("modes.vacuum_and_mop")
	case "vacuum_then_mop":
		return b.t("modes.vacuum_then_mop")
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
		return "unknown"
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "unknown"
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "unknown"
	}
	d := time.Duration(sec) * time.Second
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func (b *Bot) buildTelemetryReport() string {
	batLevel := "?"
	robotStatus := b.t("telemetry.status_docked")
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
					robotStatus = b.formatStatusDisplay(val, attr.Flag)
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
		consLines = []string{b.t("telemetry.consumables_unavailable")}
	}

	uptimeStr := getSystemUptime()
	if uptimeStr == "unknown" {
		uptimeStr = b.t("telemetry.status_unknown")
	}

	statusExtra := ""
	if dockAction != "" {
		if dockAction == "drying" {
			statusExtra = b.t("telemetry.drying_mops")
		} else {
			statusExtra = fmt.Sprintf(" (%s)", dockAction)
		}
	}

	lastMin, lastSec, lastArea := b.val.GetCurrentSessionStats()
	totHours, totCount, totArea := b.val.GetTotalStats()

	lastTimeFormatted := fmt.Sprintf(b.t("telemetry.time_format"), lastMin, lastSec)

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
		b.t("telemetry.header"),
		b.t("telemetry.sec_powertrain"),
		b.t("telemetry.lbl_battery"), batLevel,
		b.t("telemetry.lbl_status"), robotStatus, statusExtra,
		b.t("telemetry.lbl_mode"), b.formatModeTitle(currentMode),
		b.t("telemetry.lbl_uptime"), uptimeStr,
		b.t("telemetry.sec_sessions"),
		b.t("telemetry.lbl_last_session"), lastTimeFormatted, lastArea,
		b.t("telemetry.lbl_total_runs"), totCount,
		b.t("telemetry.lbl_total_stats"), totHours, totArea,
		b.t("telemetry.sec_dock_tanks"),
		b.t("telemetry.lbl_clean_water"), formatDockSensor(cleanWater),
		b.t("telemetry.lbl_dirty_water"), formatDockSensor(dirtyWater),
		b.t("telemetry.lbl_detergent"), formatDockSensor(detergent),
		b.t("telemetry.lbl_dustbag"), formatDockSensor(dustbag),
		b.t("telemetry.sec_consumables"),
		strings.Join(consLines, "\n"),
		b.t("telemetry.sec_overall"),
	)
}
