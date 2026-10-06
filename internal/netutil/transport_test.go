package netutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTouchAliveFile(t *testing.T) {
	tmpDir := t.TempDir()
	alivePath := filepath.Join(tmpDir, "tgbot.alive")

	TouchAliveFile(alivePath)
	fi1, err := os.Stat(alivePath)
	if err != nil {
		t.Fatalf("alive file was not created: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	TouchAliveFile(alivePath)
	fi2, err := os.Stat(alivePath)
	if err != nil {
		t.Fatalf("alive file missing after second touch: %v", err)
	}

	if fi2.ModTime().Before(fi1.ModTime()) {
		t.Fatalf("mod time should be updated")
	}
}
