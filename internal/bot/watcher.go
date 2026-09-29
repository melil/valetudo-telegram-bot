package bot

import (
	"bytes"
	"context"
	"io"
	"log"
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

		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("statusWatcher: перехвачена паника: %v", r)
				}
			}()

			// Автоматическая загрузка возможностей, если Valetudo был недоступен при старте
			if len(b.Caps().List()) == 0 {
				_ = b.LoadCapabilities()
			}

			attrs, err := b.val.GetAttributes()
			if err != nil {
				return
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
			return
		}

		if currentStatus == "error" && (lastStatus != "error" || currentErrorFlag != lastErrorFlag) {
			for _, chatID := range b.getNotifyChatIDs("errors") {
				msg := b.tUser(chatID, "watcher.err_robot", currentStatus, currentErrorFlag)
				_, _ = b.tg.SendTextMessage(chatID, msg, b.cfg.IsDNDActive(), nil)
			}
			b.broadcastMainDashboard()
		}

		if (lastStatus == "cleaning" || lastStatus == "returning" || b.IsSessionActive()) && currentStatus == "docked" {
			report := b.FinishSession(currentBattery)

			mapReader, err := b.val.GetMapReader()
			var mapData []byte
			if err == nil {
				mapData, _ = io.ReadAll(mapReader)
				_ = mapReader.Close()
			} else {
				log.Printf("statusWatcher: карта недоступна как изображение (%v), отправляю текстовый отчет...", err)
			}

			reportChatIDs := b.getNotifyChatIDs("reports")
			for _, chatID := range reportChatIDs {
				userMsg := b.formatReportCaptionForChat(report, chatID)
				sent := false
				if len(mapData) > 0 {
					if errPhoto := b.tg.SendPhoto(chatID, bytes.NewReader(mapData), userMsg, b.cfg.IsDNDActive()); errPhoto == nil {
						sent = true
					} else {
						log.Printf("statusWatcher: не удалось отправить фото карты пользователю %d (%v)", chatID, errPhoto)
					}
				}
				if !sent {
					if _, errText := b.tg.SendTextMessage(chatID, userMsg, b.cfg.IsDNDActive(), nil); errText != nil {
						log.Printf("statusWatcher: не удалось отправить текстовый отчет об уборке пользователю %d: %v", chatID, errText)
					}
				}
			}

			// Фоново обновляем дашборд на статус "На базе" с кнопкой отчета для всех пользователей
			b.broadcastMainDashboard()
		}

		if currentCleanWater != "ok" && lastCleanWater == "ok" {
			for _, chatID := range b.getNotifyChatIDs("station") {
				msg := b.tUser(chatID, "watcher.clean_water_low")
				_, _ = b.tg.SendTextMessage(chatID, msg, b.cfg.IsDNDActive(), nil)
			}
		}
		if currentDirtyWater != "ok" && lastDirtyWater == "ok" {
			for _, chatID := range b.getNotifyChatIDs("station") {
				msg := b.tUser(chatID, "watcher.dirty_water_full")
				_, _ = b.tg.SendTextMessage(chatID, msg, b.cfg.IsDNDActive(), nil)
			}
		}

		lastStatus = currentStatus
		lastErrorFlag = currentErrorFlag
		lastCleanWater = currentCleanWater
		lastDirtyWater = currentDirtyWater
		}()
	}
}
