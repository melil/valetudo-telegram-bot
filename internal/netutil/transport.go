package netutil

import (
	"net/http"
	"os"
	"time"
)

const (
	DefaultAliveFile = "/tmp/tgbot.alive"
)

// NewHTTPClient returns a standard http.Client with the specified timeout.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// TouchAliveFile touches the watchdog heartbeat file to notify supervisor of activity.
func TouchAliveFile(path string) {
	if path == "" {
		path = DefaultAliveFile
	}
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		f, createErr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
		if createErr == nil {
			_ = f.Close()
		}
	}
}
