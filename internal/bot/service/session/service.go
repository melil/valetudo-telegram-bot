package session

import (
	"strings"
	"sync"
	"time"

	"tgbot/internal/bot/domain"
	"tgbot/internal/i18n"
)

type Service struct {
	val domain.RobotClient

	mu         sync.RWMutex
	session    domain.CleaningSession
	lastReport *domain.CleaningReport
}

func NewService(val domain.RobotClient) *Service {
	return &Service{val: val}
}

func (s *Service) GetBatteryLevel() int {
	if s.val == nil {
		return 0
	}
	if attrs, err := s.val.GetAttributes(); err == nil {
		for _, attr := range attrs {
			if attr.Class == "BatteryStateAttribute" {
				return attr.Level
			}
		}
	}
	return 0
}

func (s *Service) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session.Active
}

func (s *Service) SetStartTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session.StartTime = t
}

func (s *Service) StartSession(rooms []string, startBattery int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session.Active {
		if len(rooms) > 0 {
			s.session.Rooms = rooms
		}
		if startBattery > 0 && s.session.StartBattery == 0 {
			s.session.StartBattery = startBattery
		}
		return
	}

	if startBattery <= 0 {
		startBattery = s.GetBatteryLevel()
	}

	var startTotalTime int
	var startTotalArea float64
	if s.val != nil {
		startTotalTime, _, startTotalArea = s.val.GetTotalStats()
	}

	s.session = domain.CleaningSession{
		Active:         true,
		StartTime:      time.Now(),
		StartBattery:   startBattery,
		StartTotalArea: startTotalArea,
		StartTotalTime: startTotalTime,
		Rooms:          rooms,
	}
}

func (s *Service) UpdateStats(min, sec int, areaM2 float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.session.Active {
		return
	}

	curSec := min*60 + sec
	peakSec := s.session.PeakMin*60 + s.session.PeakSec
	if curSec > peakSec {
		s.session.PeakMin = min
		s.session.PeakSec = sec
	}
	if areaM2 > s.session.PeakAreaM2 {
		s.session.PeakAreaM2 = areaM2
	}
}

func (s *Service) FinishSession(endBattery int, loc i18n.Locale) *domain.CleaningReport {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.session.Active && s.session.StartTime.IsZero() {
		return s.lastReport
	}

	if endBattery <= 0 {
		endBattery = s.GetBatteryLevel()
	}

	durMin := s.session.PeakMin
	durSec := s.session.PeakSec
	if durMin == 0 && durSec == 0 && !s.session.StartTime.IsZero() {
		elapsed := time.Since(s.session.StartTime)
		durMin = int(elapsed.Minutes())
		durSec = int(elapsed.Seconds()) % 60
	}

	areaM2 := s.session.PeakAreaM2
	if areaM2 <= 0.01 && s.session.StartTotalArea > 0 && s.val != nil {
		_, _, curTotalArea := s.val.GetTotalStats()
		if curTotalArea > s.session.StartTotalArea {
			areaM2 = curTotalArea - s.session.StartTotalArea
		}
	}

	startBat := s.session.StartBattery
	if startBat <= 0 {
		startBat = endBattery
	}
	batUsed := startBat - endBattery
	if batUsed < 0 {
		batUsed = 0
	}

	roomsStr := ""
	if len(s.session.Rooms) > 0 {
		roomsStr = strings.Join(s.session.Rooms, ", ")
	} else {
		roomsStr = i18n.T(loc, "report.all_house")
	}

	report := &domain.CleaningReport{
		DurationMin:  durMin,
		DurationSec:  durSec,
		AreaM2:       areaM2,
		StartBattery: startBat,
		EndBattery:   endBattery,
		BatteryUsed:  batUsed,
		Rooms:        roomsStr,
		Mode:         s.session.Mode,
		FinishedAt:   time.Now(),
	}

	s.lastReport = report
	s.session = domain.CleaningSession{}
	return report
}

func (s *Service) GetLastReport() *domain.CleaningReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastReport
}
