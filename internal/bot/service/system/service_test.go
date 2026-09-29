package system

import (
	"testing"
)

func TestParseMeminfo(t *testing.T) {
	sampleMeminfo := `MemTotal:         506720 kB
MemFree:           45180 kB
MemAvailable:     184920 kB
Buffers:           12400 kB
Cached:           127340 kB
SwapTotal:             0 kB
SwapFree:              0 kB`

	total, avail, used, ok := parseMeminfo(sampleMeminfo)
	if !ok {
		t.Fatalf("expected parseMeminfo to succeed")
	}
	if total != 506720 {
		t.Errorf("expected total=506720, got %d", total)
	}
	if avail != 184920 {
		t.Errorf("expected avail=184920, got %d", avail)
	}
	expectedUsed := uint64(506720 - 184920)
	if used != expectedUsed {
		t.Errorf("expected used=%d, got %d", expectedUsed, used)
	}

	sampleWithoutAvail := `MemTotal:         500000 kB
MemFree:           50000 kB
Buffers:           20000 kB
Cached:            30000 kB`

	total2, avail2, used2, ok2 := parseMeminfo(sampleWithoutAvail)
	if !ok2 {
		t.Fatalf("expected parseMeminfo without MemAvailable to succeed")
	}
	if total2 != 500000 {
		t.Errorf("expected total=500000, got %d", total2)
	}
	if avail2 != 100000 {
		t.Errorf("expected avail=100000, got %d", avail2)
	}
	if used2 != 400000 {
		t.Errorf("expected used=400000, got %d", used2)
	}

	_, _, _, okBad := parseMeminfo("")
	if okBad {
		t.Errorf("expected parseMeminfo on empty string to return ok=false")
	}
}

func TestParseLoadAvg(t *testing.T) {
	sample := "0.45 0.32 0.28 1/120 12345"
	l1, l5, l15, ok := parseLoadAvg(sample)
	if !ok {
		t.Fatalf("expected parseLoadAvg to succeed")
	}
	if l1 != "0.45" || l5 != "0.32" || l15 != "0.28" {
		t.Errorf("unexpected load avg values: %s, %s, %s", l1, l5, l15)
	}

	_, _, _, okBad := parseLoadAvg("bad")
	if okBad {
		t.Errorf("expected parseLoadAvg to fail on invalid input")
	}
}

func TestParseThermalTemp(t *testing.T) {
	temp, ok := parseThermalTemp("48500\n")
	if !ok || temp != 48.5 {
		t.Errorf("expected 48.5°C, got %f (ok=%v)", temp, ok)
	}

	temp2, ok2 := parseThermalTemp("52.3")
	if !ok2 || temp2 != 52.3 {
		t.Errorf("expected 52.3°C, got %f (ok=%v)", temp2, ok2)
	}

	_, okBad := parseThermalTemp("999999")
	if okBad {
		t.Errorf("expected invalid high temp to fail")
	}
}
