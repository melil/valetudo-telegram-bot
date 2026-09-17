package bot

import (
	"context"
	"fmt"
	"time"
)

func (b *Bot) statusWatcher(ctx context.Context) {
	var lastStatus string
	var lastErrorFlag string
	var lastCleanWater string
	var lastDirtyWater string
	firstRun := true

	ticker := time.NewTicker(b.cfg.WatcherInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		attrs, err := b.val.GetAttributes()
		if err != nil {
			continue
		}

		currentStatus := ""
		currentErrorFlag := "none"
		currentCleanWater := "ok"
		currentDirtyWater := "ok"

		for _, attr := range attrs {
			switch attr.Class {
			case "StatusStateAttribute":
				if val, ok := attr.Value.(string); ok {
					currentStatus = val
				}
				if attr.Flag != "" {
					currentErrorFlag = attr.Flag
				}
			case "DockComponentStateAttribute":
				valStr, _ := attr.Value.(string)
				if attr.Type == "water_tank_clean" {
					currentCleanWater = valStr
				} else if attr.Type == "water_tank_dirty" {
					currentDirtyWater = valStr
				}
			}
		}

		if firstRun {
			lastStatus = currentStatus
			lastErrorFlag = currentErrorFlag
			lastCleanWater = currentCleanWater
			lastDirtyWater = currentDirtyWater
			firstRun = false
			continue
		}

		if currentStatus == "error" && (lastStatus != "error" || currentErrorFlag != lastErrorFlag) {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, fmt.Sprintf("🚨 <b>Внимание! Ошибка робота:</b>\nСтатус: <code>%s</code> | Код: <code>%s</code>", currentStatus, currentErrorFlag), b.cfg.IsDNDActive(), nil)
		}

		if (lastStatus == "cleaning" || lastStatus == "returning") && currentStatus == "docked" {
			lastMin, lastSec, lastArea := b.val.GetCurrentSessionStats()
			msg := fmt.Sprintf(
				"🏁 <b>Уборка завершена!</b>\n"+
					"Робот успешно вернулся на станцию.\n\n"+
					"⏱ <b>Время:</b> %d мин %d сек\n"+
					"📐 <b>Площадь:</b> %.1f м²",
				lastMin, lastSec, lastArea,
			)

			mapReader, err := b.val.GetMapReader()
			if err != nil {
				_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, msg, b.cfg.IsDNDActive(), nil)
			} else {
				_ = b.tg.SendPhoto(b.cfg.AllowedChatID, mapReader, msg, b.cfg.IsDNDActive())
				_ = mapReader.Close()
			}
		}

		if currentCleanWater != "ok" && lastCleanWater == "ok" {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "🚰 <b>Док-станция: закончилась чистая вода!</b>\nПожалуйста, заполните бак.", b.cfg.IsDNDActive(), nil)
		}
		if currentDirtyWater != "ok" && lastDirtyWater == "ok" {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, "☣️ <b>Док-станция: бак грязной воды полон!</b>\nПожалуйста, слейте сточную воду.", b.cfg.IsDNDActive(), nil)
		}

		lastStatus = currentStatus
		lastErrorFlag = currentErrorFlag
		lastCleanWater = currentCleanWater
		lastDirtyWater = currentDirtyWater
	}
}
