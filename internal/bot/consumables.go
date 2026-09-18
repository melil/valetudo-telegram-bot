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

func getConsumableMeta(cType, subType string) (name string, shortName string, icon string) {
	switch cType {
	case "brush":
		switch subType {
		case "main":
			return "Основная щетка", "Осн. щетка", "🌀"
		case "side_right":
			return "Боковая щетка", "Бок. щетка", "🪥"
		case "side_left":
			return "Левая боковая щетка", "Лев. щетка", "🪥"
		default:
			return "Щетка (" + subType + ")", "Щетка", "🧹"
		}
	case "filter":
		switch subType {
		case "main":
			return "HEPA-фильтр", "HEPA-фильтр", "💨"
		default:
			return "Фильтр (" + subType + ")", "Фильтр", "💨"
		}
	case "cleaning":
		switch subType {
		case "sensor":
			return "Очистка датчиков", "Датчики", "👁"
		default:
			return "Очистка (" + subType + ")", "Очистка", "🧼"
		}
	case "mop":
		switch subType {
		case "main":
			return "Тряпки швабры", "Швабры", "💧"
		case "dock":
			return "Очистка поддона станции", "Поддон", "🧼"
		default:
			return "Швабра (" + subType + ")", "Швабра", "💧"
		}
	case "detergent":
		return "Моющее средство", "Моющее", "🧴"
	case "bin":
		if subType == "dock" {
			return "Мешок станции", "Мешок", "🗑"
		}
		return "Контейнер (" + subType + ")", "Контейнер", "🗑"
	default:
		title := strings.Title(cType)
		if subType != "" && subType != "none" && subType != "all" {
			return title + " (" + subType + ")", title, "🧹"
		}
		return title, title, "🧹"
	}
}

func formatRemainingTime(remMin int) string {
	if remMin <= 0 {
		return "исчерпан (0 мин)"
	}
	hours := remMin / 60
	mins := remMin % 60
	days := hours / 24
	hoursInDay := hours % 24

	if days > 0 {
		return fmt.Sprintf("%d ч (%d д %d ч)", hours, days, hoursInDay)
	}
	if hours > 0 {
		if mins > 0 {
			return fmt.Sprintf("%d ч %d мин", hours, mins)
		}
		return fmt.Sprintf("%d ч", hours)
	}
	return fmt.Sprintf("%d мин", mins)
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
		name, shortName, icon := getConsumableMeta(item.Type, item.SubType)
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
			info.RemainingFormatted = formatRemainingTime(remMin)
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
