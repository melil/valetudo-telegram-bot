package domain

import (
	"time"

	"tgbot/internal/valetudo"
)

// CleaningReport contains metrics from a completed cleaning session.
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

// CleaningSession tracks the state of an in-progress cleaning session.
type CleaningSession struct {
	Active          bool
	StartTime       time.Time
	StartBattery    int
	StartTotalArea  float64
	StartTotalTime  int
	PeakMin         int
	PeakSec         int
	PeakAreaM2      float64
	AccumulatedSec  int
	AccumulatedArea float64
	CurrentPassSec  int
	CurrentPassArea float64
	Rooms           []string
	Mode            string
}

// ConsumableDisplayInfo provides formatted presentation data for a consumable item.
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

// RoomInfo represents a segmented room on the map.
type RoomInfo struct {
	ID   string
	Name string
}

// WizardSession holds transient state during the multi-step cleaning wizard.
type WizardSession struct {
	Mode          string
	SelectedRooms map[string]bool
	Rooms         []RoomInfo
	Iterations    int
	MessageID     int
}

// RuntimeStats holds memory, GC, and goroutine statistics for the bot process.
type RuntimeStats struct {
	Alloc        uint64
	Sys          uint64
	HeapObjects  uint64
	NumGoroutine int
	NumGC        uint32
	GCPauseTotal time.Duration
	BotUptime    time.Duration
	GoVersion    string
	Arch         string
}

// HostStats holds system-level host metrics (RAM, CPU Load, Disk, Temp, Uptime).
type HostStats struct {
	MemTotalKB     uint64
	MemAvailableKB uint64
	MemUsedKB      uint64
	MemPercent     int
	HasMem         bool

	LoadAvg1   string
	LoadAvg5   string
	LoadAvg15  string
	HasLoadAvg bool

	DiskPath    string
	DiskTotal   uint64
	DiskFree    uint64
	DiskUsed    uint64
	DiskPercent int
	HasDisk     bool

	TempCelsius float64
	HasTemp     bool

	OSUptime string
}
