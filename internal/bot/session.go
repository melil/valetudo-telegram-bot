package bot

import (
	"fmt"
	"strings"
	"time"
)

type CleaningReport struct {
	DurationMin  int
	DurationSec  int
	AreaM2       float64
	StartBattery int
	EndBattery   int
	BatteryUsed  int
	Rooms        string
	Mode         string
	FinishedAt   time.Time
}

type CleaningSession struct {
	Active         bool
	StartTime      time.Time
	StartBattery   int
	StartTotalArea float64
	StartTotalTime int
	PeakMin        int
	PeakSec        int
	PeakAreaM2     float64
	Rooms          []string
	Mode           string
}

func (b *Bot) getBatteryLevel() int {
	if attrs, err := b.val.GetAttributes(); err == nil {
		for _, attr := range attrs {
			if attr.Class == "BatteryStateAttribute" {
				return attr.Level
			}
		}
	}
	return 0
}

func (b *Bot) IsSessionActive() bool {
	b.sessionMu.RLock()
	defer b.sessionMu.RUnlock()
	return b.session.Active
}

func (b *Bot) StartSession(rooms []string, startBattery int) {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	if b.session.Active {
		if len(rooms) > 0 {
			b.session.Rooms = rooms
		}
		if startBattery > 0 && b.session.StartBattery == 0 {
			b.session.StartBattery = startBattery
		}
		return
	}

	if startBattery <= 0 {
		startBattery = b.getBatteryLevel()
	}

	startTotalTime, _, startTotalArea := b.val.GetTotalStats()

	b.session = CleaningSession{
		Active:         true,
		StartTime:      time.Now(),
		StartBattery:   startBattery,
		StartTotalArea: startTotalArea,
		StartTotalTime: startTotalTime,
		Rooms:          rooms,
	}
}

func (b *Bot) UpdateSessionStats(min, sec int, areaM2 float64) {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	if !b.session.Active {
		return
	}

	curSec := min*60 + sec
	peakSec := b.session.PeakMin*60 + b.session.PeakSec
	if curSec > peakSec {
		b.session.PeakMin = min
		b.session.PeakSec = sec
	}
	if areaM2 > b.session.PeakAreaM2 {
		b.session.PeakAreaM2 = areaM2
	}
}

func (b *Bot) FinishSession(endBattery int) *CleaningReport {
	b.sessionMu.Lock()
	defer b.sessionMu.Unlock()

	if !b.session.Active && b.session.StartTime.IsZero() {
		return b.lastReport
	}

	if endBattery <= 0 {
		endBattery = b.getBatteryLevel()
	}

	// 1. Duration calculation
	durMin := b.session.PeakMin
	durSec := b.session.PeakSec
	if durMin == 0 && durSec == 0 && !b.session.StartTime.IsZero() {
		elapsed := time.Since(b.session.StartTime)
		durMin = int(elapsed.Minutes())
		durSec = int(elapsed.Seconds()) % 60
	}

	// 2. Area calculation with fallback to total statistics delta
	areaM2 := b.session.PeakAreaM2
	if areaM2 <= 0.01 && b.session.StartTotalArea > 0 {
		_, _, curTotalArea := b.val.GetTotalStats()
		if curTotalArea > b.session.StartTotalArea {
			areaM2 = curTotalArea - b.session.StartTotalArea
		}
	}

	// 3. Battery calculation
	startBat := b.session.StartBattery
	if startBat <= 0 {
		startBat = endBattery
	}
	batUsed := startBat - endBattery
	if batUsed < 0 {
		batUsed = 0
	}

	// 4. Cleaned rooms formatting
	roomsStr := ""
	if len(b.session.Rooms) > 0 {
		roomsStr = strings.Join(b.session.Rooms, ", ")
	} else {
		roomsStr = b.t("report.all_house")
	}

	report := &CleaningReport{
		DurationMin:  durMin,
		DurationSec:  durSec,
		AreaM2:       areaM2,
		StartBattery: startBat,
		EndBattery:   endBattery,
		BatteryUsed:  batUsed,
		Rooms:        roomsStr,
		Mode:         b.session.Mode,
		FinishedAt:   time.Now(),
	}

	b.lastReport = report
	b.session = CleaningSession{} // reset for next cleaning
	return report
}

func (b *Bot) GetLastReport() *CleaningReport {
	b.sessionMu.RLock()
	defer b.sessionMu.RUnlock()
	return b.lastReport
}

func (b *Bot) formatReportAlert(r *CleaningReport) string {
	if r == nil {
		return b.t("report.empty_alert")
	}

	rooms := r.Rooms
	// Trim long room names to respect Telegram answerCallbackQuery 200 rune limit
	runes := []rune(rooms)
	if len(runes) > 28 {
		rooms = string(runes[:25]) + "..."
	}

	return fmt.Sprintf(
		"🏁 %s\n⏱ %s: %d %s %d %s\n📐 %s: %.1f м²\n🔋 %s: %d%% ➔ %d%% (-%d%%)\n🧹 %s: %s",
		b.t("report.alert_title"),
		b.t("report.lbl_time"), r.DurationMin, b.t("report.min"), r.DurationSec, b.t("report.sec"),
		b.t("report.lbl_area"), r.AreaM2,
		b.t("report.lbl_battery"), r.StartBattery, r.EndBattery, r.BatteryUsed,
		b.t("report.lbl_rooms"), rooms,
	)
}

func (b *Bot) formatReportCaption(r *CleaningReport) string {
	return b.formatReportCaptionForChat(r, b.GetActiveChatID())
}

func (b *Bot) formatReportCaptionForChat(r *CleaningReport, chatID int64) string {
	if r == nil {
		min, sec, areaM2 := b.val.GetCurrentSessionStats()
		if min > 0 || sec > 0 || areaM2 > 0 {
			return fmt.Sprintf(
				"🏁 <b>%s</b>\n%s\n\n⏱ <b>%s:</b> %d %s %d %s\n📐 <b>%s:</b> %.1f м²",
				b.tUser(chatID, "watcher.cleaning_finished_title"),
				b.tUser(chatID, "watcher.cleaning_finished_subtitle"),
				b.tUser(chatID, "report.lbl_time"), min, b.tUser(chatID, "report.min"), sec, b.tUser(chatID, "report.sec"),
				b.tUser(chatID, "report.lbl_area"), areaM2,
			)
		}
		return fmt.Sprintf(
			"🏁 <b>%s</b>\n%s",
			b.tUser(chatID, "watcher.cleaning_finished_title"),
			b.tUser(chatID, "watcher.cleaning_finished_subtitle"),
		)
	}

	return fmt.Sprintf(
		"🏁 <b>%s</b>\n%s\n\n⏱ <b>%s:</b> %d %s %d %s\n📐 <b>%s:</b> %.1f м²\n🔋 <b>%s:</b> %d%% ➔ %d%% (-%d%%)\n🧹 <b>%s:</b> %s",
		b.tUser(chatID, "watcher.cleaning_finished_title"),
		b.tUser(chatID, "watcher.cleaning_finished_subtitle"),
		b.tUser(chatID, "report.lbl_time"), r.DurationMin, b.tUser(chatID, "report.min"), r.DurationSec, b.tUser(chatID, "report.sec"),
		b.tUser(chatID, "report.lbl_area"), r.AreaM2,
		b.tUser(chatID, "report.lbl_battery"), r.StartBattery, r.EndBattery, r.BatteryUsed,
		b.tUser(chatID, "report.lbl_rooms"), r.Rooms,
	)
}
