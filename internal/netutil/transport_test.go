package netutil

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResilientDialer_ResolveLiteralIP(t *testing.T) {
	d := NewResilientDialer()
	ips, err := d.Resolve(context.Background(), "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) == 0 || !ips[0].Equal(net.ParseIP("127.0.0.1")) {
		t.Fatalf("expected 127.0.0.1, got %v", ips)
	}
}

func TestResilientDialer_ResolveCache(t *testing.T) {
	d := NewResilientDialer()
	d.cache["fake.example.com"] = dnsCacheEntry{
		ips:       []net.IP{net.ParseIP("10.0.0.1")},
		expiresAt: time.Now().Add(10 * time.Minute),
	}

	ips, err := d.Resolve(context.Background(), "fake.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) == 0 || !ips[0].Equal(net.ParseIP("10.0.0.1")) {
		t.Fatalf("expected 10.0.0.1 from cache, got %v", ips)
	}
}

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
