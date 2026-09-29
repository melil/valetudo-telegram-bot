package watcher

import (
	"bytes"
	"context"
	"io"
	"log"
	"time"

	"tgbot/internal/bot/domain"
	"tgbot/internal/bot/service/auth"
	"tgbot/internal/bot/service/session"
)

type Config struct {
	Interval           time.Duration
	IsDNDActive        func() bool
	OnStatusUpdate     func(status, flag string)
	OnLoadCapabilities func() error
	OnBroadcastDash    func()
	FormatCaption      func(report *domain.CleaningReport, chatID int64) string
	TranslateUser      func(chatID int64, key string, args ...any) string
}

type Service struct {
	val        domain.RobotClient
	tg         domain.Messenger
	authSvc    *auth.Service
	sessionSvc *session.Service
	cfg        Config
}

func NewService(
	val domain.RobotClient,
	tg domain.Messenger,
	authSvc *auth.Service,
	sessionSvc *session.Service,
	cfg Config,
) *Service {
	return &Service{
		val:        val,
		tg:         tg,
		authSvc:    authSvc,
		sessionSvc: sessionSvc,
		cfg:        cfg,
	}
}

func (s *Service) Start(ctx context.Context) {
	var lastStatus string
	var lastErrorFlag string
	var lastCleanWater string
	var lastDirtyWater string
	firstRun := true

	interval := s.cfg.Interval
	if interval <= 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
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
					log.Printf("watcher: перехвачена паника: %v", r)
				}
			}()

			if s.cfg.OnLoadCapabilities != nil {
				_ = s.cfg.OnLoadCapabilities()
			}

			attrs, err := s.val.GetAttributes()
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

			if s.cfg.OnStatusUpdate != nil {
				s.cfg.OnStatusUpdate(currentStatus, currentErrorFlag)
			}

			// Автоматический запуск отслеживания сессии, если уборку включили вне визарда
			if (currentStatus == "cleaning" || currentStatus == "moving") && !s.sessionSvc.IsActive() {
				s.sessionSvc.StartSession(nil, currentBattery)
			}

			// Сбор пиковых метрик пока идет уборка или возврат на базу
			if s.sessionSvc.IsActive() && (currentStatus == "cleaning" || currentStatus == "moving" || currentStatus == "returning" || currentStatus == "paused") {
				min, sec, areaM2 := s.val.GetCurrentSessionStats()
				s.sessionSvc.UpdateStats(min, sec, areaM2)
			}

			if firstRun {
				lastStatus = currentStatus
				lastErrorFlag = currentErrorFlag
				lastCleanWater = currentCleanWater
				lastDirtyWater = currentDirtyWater
				firstRun = false
				return
			}

			isDND := false
			if s.cfg.IsDNDActive != nil {
				isDND = s.cfg.IsDNDActive()
			}

			if currentStatus == "error" && (lastStatus != "error" || currentErrorFlag != lastErrorFlag) {
				for _, chatID := range s.authSvc.GetNotifyChatIDs("errors") {
					msg := s.cfg.TranslateUser(chatID, "watcher.err_robot", currentStatus, currentErrorFlag)
					_, _ = s.tg.SendTextMessage(chatID, msg, isDND, nil)
				}
				if s.cfg.OnBroadcastDash != nil {
					s.cfg.OnBroadcastDash()
				}
			}

			if (lastStatus == "cleaning" || lastStatus == "returning" || s.sessionSvc.IsActive()) && currentStatus == "docked" {
				report := s.sessionSvc.FinishSession(currentBattery, "ru")

				mapReader, err := s.val.GetMapReader()
				var mapData []byte
				if err == nil {
					mapData, _ = io.ReadAll(mapReader)
					_ = mapReader.Close()
				} else {
					log.Printf("watcher: карта недоступна как изображение (%v), отправляю текстовый отчет...", err)
				}

				reportChatIDs := s.authSvc.GetNotifyChatIDs("reports")
				for _, chatID := range reportChatIDs {
					userMsg := ""
					if s.cfg.FormatCaption != nil {
						userMsg = s.cfg.FormatCaption(report, chatID)
					}
					sent := false
					if len(mapData) > 0 {
						if errPhoto := s.tg.SendPhoto(chatID, bytes.NewReader(mapData), userMsg, isDND); errPhoto == nil {
							sent = true
						} else {
							log.Printf("watcher: не удалось отправить фото карты пользователю %d (%v)", chatID, errPhoto)
						}
					}
					if !sent {
						if _, errText := s.tg.SendTextMessage(chatID, userMsg, isDND, nil); errText != nil {
							log.Printf("watcher: не удалось отправить текстовый отчет об уборке пользователю %d: %v", chatID, errText)
						}
					}
				}

				if s.cfg.OnBroadcastDash != nil {
					s.cfg.OnBroadcastDash()
				}
			}

			if currentCleanWater != "ok" && lastCleanWater == "ok" {
				for _, chatID := range s.authSvc.GetNotifyChatIDs("station") {
					msg := s.cfg.TranslateUser(chatID, "watcher.clean_water_low")
					_, _ = s.tg.SendTextMessage(chatID, msg, isDND, nil)
				}
			}
			if currentDirtyWater != "ok" && lastDirtyWater == "ok" {
				for _, chatID := range s.authSvc.GetNotifyChatIDs("station") {
					msg := s.cfg.TranslateUser(chatID, "watcher.dirty_water_full")
					_, _ = s.tg.SendTextMessage(chatID, msg, isDND, nil)
				}
			}

			lastStatus = currentStatus
			lastErrorFlag = currentErrorFlag
			lastCleanWater = currentCleanWater
			lastDirtyWater = currentDirtyWater
		}()
	}
}
