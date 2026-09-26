package bot

import (
	"context"
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
		currentBattery := 0

		for _, attr := range attrs {
			switch attr.Class {
			case "StatusStateAttribute":
				if val, ok := attr.Value.(string); ok {
					currentStatus = val
				}
				if attr.Flag != "" {
					currentErrorFlag = attr.Flag
				}
			case "BatteryStateAttribute":
				currentBattery = attr.Level
			case "DockComponentStateAttribute":
				valStr, _ := attr.Value.(string)
				if attr.Type == "water_tank_clean" {
					currentCleanWater = valStr
				} else if attr.Type == "water_tank_dirty" {
					currentDirtyWater = valStr
				}
			}
		}

		b.SetRobotStatus(currentStatus, currentErrorFlag)

		// Автоматический запуск отслеживания сессии, если уборку включили вне визарда (кнопкой на роботе/в вебе)
		if (currentStatus == "cleaning" || currentStatus == "moving") && !b.IsSessionActive() {
			b.StartSession(nil, currentBattery)
		}

		// Сбор пиковых метрик пока идет уборка или возврат на базу
		if b.IsSessionActive() && (currentStatus == "cleaning" || currentStatus == "moving" || currentStatus == "returning" || currentStatus == "paused") {
			min, sec, areaM2 := b.val.GetCurrentSessionStats()
			b.UpdateSessionStats(min, sec, areaM2)
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
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("watcher.err_robot", currentStatus, currentErrorFlag), b.cfg.IsDNDActive(), nil)
			b.sendMainDashboard()
		}

		if (lastStatus == "cleaning" || lastStatus == "returning") && currentStatus == "docked" {
			report := b.FinishSession(currentBattery)
			msg := b.formatReportCaption(report)

			mapReader, err := b.val.GetMapReader()
			if err != nil {
				_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, msg, b.cfg.IsDNDActive(), nil)
			} else {
				_ = b.tg.SendPhoto(b.cfg.AllowedChatID, mapReader, msg, b.cfg.IsDNDActive())
				_ = mapReader.Close()
			}
			// Фоново обновляем дашборд на статус "На базе" с кнопкой отчета
			b.sendMainDashboard()
		}

		if currentCleanWater != "ok" && lastCleanWater == "ok" {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("watcher.clean_water_low"), b.cfg.IsDNDActive(), nil)
		}
		if currentDirtyWater != "ok" && lastDirtyWater == "ok" {
			_, _ = b.tg.SendTextMessage(b.cfg.AllowedChatID, b.t("watcher.dirty_water_full"), b.cfg.IsDNDActive(), nil)
		}

		lastStatus = currentStatus
		lastErrorFlag = currentErrorFlag
		lastCleanWater = currentCleanWater
		lastDirtyWater = currentDirtyWater
	}
}
