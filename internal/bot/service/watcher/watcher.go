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
	var lastFlag string
	var lastCleanWater string
	var lastDirtyWater string
	var lastMidCleanDocked bool
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
			currentFlag := "none"
			currentCleanWater := "ok"
			currentDirtyWater := "ok"
			currentDockStatus := "idle"
			currentBattery := 0

			for _, attr := range attrs {
				switch attr.Class {
				case "StatusStateAttribute":
					if val, ok := attr.Value.(string); ok {
						currentStatus = val
					}
					if attr.Flag != "" {
						currentFlag = attr.Flag
					}
				case "BatteryStateAttribute":
					currentBattery = attr.Level
				case "DockStatusStateAttribute":
					if val, ok := attr.Value.(string); ok {
						currentDockStatus = val
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

			if s.cfg.OnStatusUpdate != nil {
				s.cfg.OnStatusUpdate(currentStatus, currentFlag)
			}

			// Автоматический запуск отслеживания сессии, если уборку включили вне визарда
			if (currentStatus == "cleaning" || currentStatus == "moving") && !s.sessionSvc.IsActive() {
				s.sessionSvc.StartSession(nil, currentBattery)
			}

			// Проверяем, находится ли робот на станции временно во время незавершенной уборки:
			// - флаг resumable (стирка мопов между проходами/комнатами, смена прохода в vacuum_then_mop, дозарядка и т.д.)
			// - док-станция в процессе очистки мопов (cleaning) или опустошения пылесборника (emptying)
			isMidCleanDocked := currentStatus == "docked" && (currentFlag == "resumable" || currentDockStatus == "cleaning" || currentDockStatus == "emptying")

			// Сбор пиковых метрик пока идет уборка, возврат на базу или промежуточная очистка на доке
			if s.sessionSvc.IsActive() && (currentStatus == "cleaning" || currentStatus == "moving" || currentStatus == "returning" || currentStatus == "paused" || isMidCleanDocked) {
				min, sec, areaM2 := s.val.GetCurrentSessionStats()
				s.sessionSvc.UpdateStats(min, sec, areaM2)
			}

			if firstRun {
				lastStatus = currentStatus
				lastFlag = currentFlag
				lastCleanWater = currentCleanWater
				lastDirtyWater = currentDirtyWater
				lastMidCleanDocked = isMidCleanDocked
				firstRun = false
				return
			}

			isDND := false
			if s.cfg.IsDNDActive != nil {
				isDND = s.cfg.IsDNDActive()
			}

			if currentStatus == "error" && (lastStatus != "error" || currentFlag != lastFlag) {
				for _, chatID := range s.authSvc.GetNotifyChatIDs("errors") {
					msg := s.cfg.TranslateUser(chatID, "watcher.err_robot", currentStatus, currentFlag)
					_, _ = s.tg.SendTextMessage(chatID, msg, isDND, nil)
				}
				if s.cfg.OnBroadcastDash != nil {
					s.cfg.OnBroadcastDash()
				}
			}

			justDocked := (lastStatus == "cleaning" || lastStatus == "returning") && currentStatus == "docked" && !isMidCleanDocked
			dockServiceFinished := lastMidCleanDocked && currentStatus == "docked" && !isMidCleanDocked && s.sessionSvc.IsActive()

			if justDocked || dockServiceFinished {
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
			lastFlag = currentFlag
			lastCleanWater = currentCleanWater
			lastDirtyWater = currentDirtyWater
			lastMidCleanDocked = isMidCleanDocked
		}()
	}
}
