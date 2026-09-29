package consumables

import (
	"fmt"
	"math"
	"strings"

	"tgbot/internal/bot/domain"
	"tgbot/internal/i18n"
)

type Service struct {
	val domain.RobotClient
}

func NewService(val domain.RobotClient) *Service {
	return &Service{val: val}
}

func (s *Service) GetConsumableMeta(cType, subType string, loc i18n.Locale) (name string, shortName string, icon string) {
	switch cType {
	case "brush":
		switch subType {
		case "main":
			return i18n.T(loc, "consumables.main_brush"), i18n.T(loc, "consumables.main_brush_short"), "🌀"
		case "side_right":
			return i18n.T(loc, "consumables.side_brush"), i18n.T(loc, "consumables.side_brush_short"), "🪥"
		case "side_left":
			return i18n.T(loc, "consumables.side_brush_left"), i18n.T(loc, "consumables.side_brush_left_short"), "🪥"
		default:
			return i18n.T(loc, "consumables.brush_generic", subType), i18n.T(loc, "consumables.main_brush_short"), "🧹"
		}
	case "filter":
		switch subType {
		case "main":
			return i18n.T(loc, "consumables.hepa_filter"), i18n.T(loc, "consumables.hepa_filter_short"), "💨"
		default:
			return i18n.T(loc, "consumables.filter_generic", subType), i18n.T(loc, "consumables.hepa_filter_short"), "💨"
		}
	case "cleaning":
		switch subType {
		case "sensor":
			return i18n.T(loc, "consumables.sensor_cleaning"), i18n.T(loc, "consumables.sensor_cleaning_short"), "👁"
		default:
			return i18n.T(loc, "consumables.cleaning_generic", subType), i18n.T(loc, "consumables.sensor_cleaning_short"), "🧼"
		}
	case "mop":
		switch subType {
		case "main":
			return i18n.T(loc, "consumables.mop_pads"), i18n.T(loc, "consumables.mop_pads_short"), "💧"
		case "dock":
			return i18n.T(loc, "consumables.dock_tray"), i18n.T(loc, "consumables.dock_tray_short"), "🧼"
		default:
			return i18n.T(loc, "consumables.mop_generic", subType), i18n.T(loc, "consumables.mop_pads_short"), "💧"
		}
	case "detergent":
		return i18n.T(loc, "consumables.detergent"), i18n.T(loc, "consumables.detergent_short"), "🧴"
	case "bin":
		if subType == "dock" {
			return i18n.T(loc, "consumables.dustbag"), i18n.T(loc, "consumables.dustbag_short"), "🗑"
		}
		return i18n.T(loc, "consumables.bin_generic", subType), i18n.T(loc, "consumables.dustbag_short"), "🗑"
	default:
		title := strings.Title(cType)
		if subType != "" && subType != "none" && subType != "all" {
			return title + " (" + subType + ")", title, "🧹"
		}
		return title, title, "🧹"
	}
}

func (s *Service) FormatRemainingTime(remMin int, loc i18n.Locale) string {
	if remMin <= 0 {
		return i18n.T(loc, "consumables.depleted_time")
	}
	hours := remMin / 60
	mins := remMin % 60
	days := hours / 24
	hoursInDay := hours % 24

	if days > 0 {
		return i18n.T(loc, "consumables.time_days_hours", hours, days, hoursInDay)
	}
	if hours > 0 {
		if mins > 0 {
			return i18n.T(loc, "consumables.time_hours_mins", hours, mins)
		}
		return i18n.T(loc, "consumables.time_hours", hours)
	}
	return i18n.T(loc, "consumables.time_mins", mins)
}

func (s *Service) GetConsumablesDisplay(loc i18n.Locale) ([]domain.ConsumableDisplayInfo, error) {
	items, err := s.val.GetConsumables()
	if err != nil {
		return nil, err
	}

	maxMap := map[string]int{
		"brush/main":       18000,
		"brush/side_right":  12000,
		"brush/side_left":   12000,
		"filter/main":      9000,
		"cleaning/sensor":  1800,
		"mop/main":         6000,
	}

	if props, err := s.val.GetConsumableProperties(); err == nil && props != nil {
		for _, p := range props.AvailableConsumables {
			if p.MaxValue > 0 {
				key := p.Type + "/" + p.SubType
				maxMap[key] = p.MaxValue
			}
		}
	}

	var result []domain.ConsumableDisplayInfo
	for _, item := range items {
		name, shortName, icon := s.GetConsumableMeta(item.Type, item.SubType, loc)
		key := item.Type + "/" + item.SubType

		info := domain.ConsumableDisplayInfo{
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
				maxMin = 9000
			}

			info.RemainingH = remMin / 60
			info.MaxH = maxMin / 60
			info.RemainingFormatted = s.FormatRemainingTime(remMin, loc)
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

func (s *Service) ResetConsumable(cType, subType string) error {
	return s.val.ResetConsumable(cType, subType)
}
