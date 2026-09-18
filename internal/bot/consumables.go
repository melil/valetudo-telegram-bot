package bot

import (
	"fmt"
	"math"
	"strings"

	"tgbot/internal/valetudo"
)

type ConsumableDisplayInfo struct {
	Item               valetudo.ConsumableItem
	Name               string
	ShortName          string
	Icon               string
	RemainingH         int
	RemainingFormatted string
	MaxH               int
	Percent            int
	IsDepleted         bool
	IsMinutes          bool
}

func (b *Bot) getConsumableMeta(cType, subType string) (name string, shortName string, icon string) {
	switch cType {
	case "brush":
		switch subType {
		case "main":
			return b.t("consumables.main_brush"), b.t("consumables.main_brush_short"), "🌀"
		case "side_right":
			return b.t("consumables.side_brush"), b.t("consumables.side_brush_short"), "🪥"
		case "side_left":
			return b.t("consumables.side_brush_left"), b.t("consumables.side_brush_left_short"), "🪥"
		default:
			return b.t("consumables.brush_generic", subType), b.t("consumables.main_brush_short"), "🧹"
		}
	case "filter":
		switch subType {
		case "main":
			return b.t("consumables.hepa_filter"), b.t("consumables.hepa_filter_short"), "💨"
		default:
			return b.t("consumables.filter_generic", subType), b.t("consumables.hepa_filter_short"), "💨"
		}
	case "cleaning":
		switch subType {
		case "sensor":
			return b.t("consumables.sensor_cleaning"), b.t("consumables.sensor_cleaning_short"), "👁"
		default:
			return b.t("consumables.cleaning_generic", subType), b.t("consumables.sensor_cleaning_short"), "🧼"
		}
	case "mop":
		switch subType {
		case "main":
			return b.t("consumables.mop_pads"), b.t("consumables.mop_pads_short"), "💧"
		case "dock":
			return b.t("consumables.dock_tray"), b.t("consumables.dock_tray_short"), "🧼"
		default:
			return b.t("consumables.mop_generic", subType), b.t("consumables.mop_pads_short"), "💧"
		}
	case "detergent":
		return b.t("consumables.detergent"), b.t("consumables.detergent_short"), "🧴"
	case "bin":
		if subType == "dock" {
			return b.t("consumables.dustbag"), b.t("consumables.dustbag_short"), "🗑"
		}
		return b.t("consumables.bin_generic", subType), b.t("consumables.dustbag_short"), "🗑"
	default:
		title := strings.Title(cType)
		if subType != "" && subType != "none" && subType != "all" {
			return title + " (" + subType + ")", title, "🧹"
		}
		return title, title, "🧹"
	}
}

func (b *Bot) formatRemainingTime(remMin int) string {
	if remMin <= 0 {
		return b.t("consumables.depleted_time")
	}
	hours := remMin / 60
	mins := remMin % 60
	days := hours / 24
	hoursInDay := hours % 24

	if days > 0 {
		return b.t("consumables.time_days_hours", hours, days, hoursInDay)
	}
	if hours > 0 {
		if mins > 0 {
			return b.t("consumables.time_hours_mins", hours, mins)
		}
		return b.t("consumables.time_hours", hours)
	}
	return b.t("consumables.time_mins", mins)
}

func (b *Bot) getConsumablesDisplay() ([]ConsumableDisplayInfo, error) {
	items, err := b.val.GetConsumables()
	if err != nil {
		return nil, err
	}

	// Запрашиваем официальные maxValue из свойств робота
	maxMap := map[string]int{
		"brush/main":        18000,
		"brush/side_right":   12000,
		"brush/side_left":    12000,
		"filter/main":       9000,
		"cleaning/sensor":   1800,
		"mop/main":          6000,
	}

	if props, err := b.val.GetConsumableProperties(); err == nil && props != nil {
		for _, p := range props.AvailableConsumables {
			if p.MaxValue > 0 {
				key := p.Type + "/" + p.SubType
				maxMap[key] = p.MaxValue
			}
		}
	}

	var result []ConsumableDisplayInfo
	for _, item := range items {
		name, shortName, icon := b.getConsumableMeta(item.Type, item.SubType)
		key := item.Type + "/" + item.SubType

		info := ConsumableDisplayInfo{
			Item:      item,
			Name:      name,
			ShortName: shortName,
			Icon:      icon,
		}

		if item.Remaining.Unit == "percent" {
			info.IsMinutes = false
			info.Percent = item.Remaining.Value
			if info.Percent < 0 {
				info.Percent = 0
			}
			if info.Percent > 100 {
				info.Percent = 100
			}
			info.IsDepleted = (info.Percent <= 0)
			info.RemainingFormatted = fmt.Sprintf("%d%%", info.Percent)
		} else {
			// unit: minutes
			info.IsMinutes = true
			remMin := item.Remaining.Value
			maxMin := maxMap[key]
			if maxMin <= 0 {
				maxMin = 9000 // 150 часов по умолчанию
			}

			info.RemainingH = remMin / 60
			info.MaxH = maxMin / 60
			info.RemainingFormatted = b.formatRemainingTime(remMin)
			info.IsDepleted = (remMin <= 0)

			if remMin <= 0 {
				info.Percent = 0
			} else {
				pct := int(math.Round(float64(remMin) * 100.0 / float64(maxMin)))
				if pct > 100 {
					pct = 100
				}
				if pct < 0 {
					pct = 0
				}
				info.Percent = pct
			}
		}

		result = append(result, info)
	}

	return result, nil
}
