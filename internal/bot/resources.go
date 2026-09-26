package bot

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
)

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

func collectRuntimeStats(startTime time.Time) RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Duration(0)
	if !startTime.IsZero() {
		uptime = time.Since(startTime)
	}

	return RuntimeStats{
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

func formatBytes(bytes uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatKB(kb uint64) string {
	return formatBytes(kb * 1024)
}

func formatDuration(d time.Duration, loc i18n.Locale) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60

	switch loc {
	case i18n.LocaleRU:
		if days > 0 {
			return fmt.Sprintf("%dд %dч %dм", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%dч %dм", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%dм %dс", mins, secs)
		}
		return fmt.Sprintf("%dс", secs)

	case i18n.LocaleDE:
		if days > 0 {
			return fmt.Sprintf("%dT %dStd %dMin", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%dStd %dMin", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%dMin %dSek", mins, secs)
		}
		return fmt.Sprintf("%dSek", secs)

	case i18n.LocaleZH:
		if days > 0 {
			return fmt.Sprintf("%d天 %d小时 %d分", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%d小时 %d分", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%d分 %d秒", mins, secs)
		}
		return fmt.Sprintf("%d秒", secs)

	default: // LocaleEN and others
		if days > 0 {
			return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
		}
		if hours > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		if mins > 0 {
			return fmt.Sprintf("%dm %ds", mins, secs)
		}
		return fmt.Sprintf("%ds", secs)
	}
}

func formatGCPause(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d µs", d.Microseconds())
	}
	return fmt.Sprintf("%.2f ms", float64(d.Microseconds())/1000.0)
}

func collectHostStats() HostStats {
	var stats HostStats

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

func (b *Bot) buildResourcesReport() string {
	rStats := collectRuntimeStats(b.startTime)
	hStats := collectHostStats()
	loc := b.GetLang()

	var sb strings.Builder

	sb.WriteString(b.t("resources.title") + "\n\n")

	// Section 1: Bot Process (Go runtime)
	sb.WriteString(b.t("resources.sec_bot") + "\n")
	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b> (Sys: <b>%s</b>)\n",
		b.t("resources.lbl_heap"), formatBytes(rStats.Alloc), formatBytes(rStats.Sys)))
	sb.WriteString(fmt.Sprintf("• %s: <b>%d</b>\n", b.t("resources.lbl_goroutines"), rStats.NumGoroutine))
	sb.WriteString(fmt.Sprintf("• %s: <b>%d</b>\n", b.t("resources.lbl_objects"), rStats.HeapObjects))

	gcInfo := fmt.Sprintf(b.t("resources.gc_stat"), rStats.NumGC, formatGCPause(rStats.GCPauseTotal))
	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b>\n", b.t("resources.lbl_gc"), gcInfo))

	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b>\n", b.t("resources.lbl_bot_uptime"), formatDuration(rStats.BotUptime, loc)))
	sb.WriteString(fmt.Sprintf("• %s: <code>%s (%s)</code>\n\n", b.t("resources.lbl_go_version"), rStats.GoVersion, rStats.Arch))

	// Section 2: Host System (Linux)
	sb.WriteString(b.t("resources.sec_host") + "\n")

	if hStats.HasMem {
		memBar := renderProgressBar(hStats.MemPercent)
		sb.WriteString(fmt.Sprintf("• %s: <b>%s</b> / %s (%d%%)\n  <code>[%s] %d%%</code>\n",
			b.t("resources.lbl_host_mem"),
			formatKB(hStats.MemUsedKB),
			formatKB(hStats.MemTotalKB),
			hStats.MemPercent,
			memBar,
			hStats.MemPercent,
		))
	} else {
		sb.WriteString(fmt.Sprintf("• %s: <i>%s</i>\n", b.t("resources.lbl_host_mem"), b.t("resources.na")))
	}

	if hStats.HasLoadAvg {
		sb.WriteString(fmt.Sprintf("• %s: <b>%s, %s, %s</b>\n",
			b.t("resources.lbl_load_avg"),
			hStats.LoadAvg1, hStats.LoadAvg5, hStats.LoadAvg15,
		))
	} else {
		sb.WriteString(fmt.Sprintf("• %s: <i>%s</i>\n", b.t("resources.lbl_load_avg"), b.t("resources.na")))
	}

	if hStats.HasDisk {
		diskVal := fmt.Sprintf(b.t("resources.disk_usage"),
			formatBytes(hStats.DiskFree),
			formatBytes(hStats.DiskTotal),
			hStats.DiskPercent,
		)
		sb.WriteString(fmt.Sprintf("• %s: %s\n",
			fmt.Sprintf(b.t("resources.lbl_disk"), hStats.DiskPath),
			diskVal,
		))
	} else {
		sb.WriteString(fmt.Sprintf("• %s: <i>%s</i>\n",
			fmt.Sprintf(b.t("resources.lbl_disk"), "/data"),
			b.t("resources.na"),
		))
	}

	if hStats.HasTemp {
		sb.WriteString(fmt.Sprintf("• %s: <b>%.1f°C</b>\n", b.t("resources.lbl_temp"), hStats.TempCelsius))
	}

	uptimeStr := hStats.OSUptime
	if uptimeStr == "unknown" || uptimeStr == "" {
		uptimeStr = b.t("telemetry.status_unknown")
	}
	sb.WriteString(fmt.Sprintf("• %s: <b>%s</b>", b.t("resources.lbl_os_uptime"), uptimeStr))

	return sb.String()
}

func (b *Bot) sendResourcesMenu() {
	text := b.buildResourcesReport()
	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: b.t("resources.btn_refresh"), CallbackData: "cmd_resources_refresh"},
			},
			{
				{Text: b.t("robot_menu.btn_back"), CallbackData: "menu_robot"},
			},
		},
	}
	_ = b.renderDashboard(text, markup)
}
