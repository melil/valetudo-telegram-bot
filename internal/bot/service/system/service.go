package system

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"tgbot/internal/bot/domain"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CollectRuntime(startTime time.Time) domain.RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Duration(0)
	if !startTime.IsZero() {
		uptime = time.Since(startTime)
	}

	return domain.RuntimeStats{
		Alloc:        m.Alloc,
		Sys:          m.Sys,
		HeapObjects:  m.HeapObjects,
		NumGoroutine: runtime.NumGoroutine(),
		NumGC:        m.NumGC,
		GCPauseTotal: time.Duration(m.PauseTotalNs),
		BotUptime:    uptime,
		GoVersion:    runtime.Version(),
		Arch:         fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func (s *Service) CollectHost() domain.HostStats {
	var stats domain.HostStats

	// 1. RAM from /proc/meminfo
	if memData, err := os.ReadFile("/proc/meminfo"); err == nil {
		if total, avail, used, ok := parseMeminfo(string(memData)); ok {
			stats.MemTotalKB = total
			stats.MemAvailableKB = avail
			stats.MemUsedKB = used
			if total > 0 {
				stats.MemPercent = int((used * 100) / total)
			}
			stats.HasMem = true
		}
	}

	// 2. CPU Load Avg from /proc/loadavg
	if laData, err := os.ReadFile("/proc/loadavg"); err == nil {
		if l1, l5, l15, ok := parseLoadAvg(string(laData)); ok {
			stats.LoadAvg1 = l1
			stats.LoadAvg5 = l5
			stats.LoadAvg15 = l15
			stats.HasLoadAvg = true
		}
	}

	// 3. Disk space (check /data first, then /)
	for _, path := range []string{"/data", "/"} {
		total, free, used, err := getDiskSpaceInfo(path)
		if err == nil && total > 0 {
			stats.DiskPath = path
			stats.DiskTotal = total
			stats.DiskFree = free
			stats.DiskUsed = used
			stats.DiskPercent = int((free * 100) / total)
			stats.HasDisk = true
			break
		}
	}

	// 4. SoC Temperature
	for _, tPath := range []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
	} {
		if tData, err := os.ReadFile(tPath); err == nil {
			if temp, ok := parseThermalTemp(string(tData)); ok {
				stats.TempCelsius = temp
				stats.HasTemp = true
				break
			}
		}
	}

	// 5. System Uptime
	stats.OSUptime = getSystemUptime()

	return stats
}

func parseMeminfo(content string) (totalKB, availKB, usedKB uint64, ok bool) {
	var total, free, avail, buffers, cached uint64
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemFree:":
			free = val
		case "MemAvailable:":
			avail = val
		case "Buffers:":
			buffers = val
		case "Cached:":
			cached = val
		}
	}
	if total == 0 {
		return 0, 0, 0, false
	}
	if avail == 0 && free > 0 {
		avail = free + buffers + cached
	}
	if total >= avail {
		usedKB = total - avail
	}
	return total, avail, usedKB, true
}

func parseLoadAvg(content string) (l1, l5, l15 string, ok bool) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return "", "", "", false
	}
	return fields[0], fields[1], fields[2], true
}

func parseThermalTemp(content string) (float64, bool) {
	str := strings.TrimSpace(content)
	if str == "" {
		return 0, false
	}
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, false
	}
	if val > 1000 {
		val = val / 1000.0
	}
	if val < 0 || val > 150 {
		return 0, false
	}
	return val, true
}

func getSystemUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "unknown"
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "unknown"
	}
	d := time.Duration(sec) * time.Second
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
