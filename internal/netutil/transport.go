package netutil

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// FallbackDNSServers contains reliable public DNS servers used when local DNS fails.
var FallbackDNSServers = []string{
	"8.8.8.8:53",
	"1.1.1.1:53",
	"77.88.8.8:53",
}

const (
	DefaultAliveFile  = "/tmp/tgbot.alive"
	DefaultCacheTTL   = 5 * time.Minute
	DefaultDNSWait    = 3 * time.Second
	DefaultDialWait   = 10 * time.Second
	DefaultSingleDial = 5 * time.Second
)

type dnsCacheEntry struct {
	ips       []net.IP
	expiresAt time.Time
}

// ResilientDialer provides DNS fallback and IPv4 preference for embedded systems.
type ResilientDialer struct {
	mu         sync.RWMutex
	cache      map[string]dnsCacheEntry
	cacheTTL   time.Duration
	dialer     *net.Dialer
	dnsServers []string
}

// NewResilientDialer creates a new dialer with DNS fallback capability.
func NewResilientDialer(dnsServers ...string) *ResilientDialer {
	servers := FallbackDNSServers
	if len(dnsServers) > 0 {
		servers = dnsServers
	}
	return &ResilientDialer{
		cache:      make(map[string]dnsCacheEntry),
		cacheTTL:   DefaultCacheTTL,
		dialer:     &net.Dialer{Timeout: DefaultDialWait, KeepAlive: 30 * time.Second},
		dnsServers: servers,
	}
}

func (d *ResilientDialer) updateCache(host string, ips []net.IP) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cache[host] = dnsCacheEntry{
		ips:       ips,
		expiresAt: time.Now().Add(d.cacheTTL),
	}
}

// Resolve resolves the host to IPv4 addresses using system DNS first, then fallback public DNS.
func (d *ResilientDialer) Resolve(ctx context.Context, host string) ([]net.IP, error) {
	// 1. IP literal check
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() != nil {
			return []net.IP{ip}, nil
		}
		return []net.IP{ip}, nil
	}

	// 2. Check internal cache
	d.mu.RLock()
	entry, found := d.cache[host]
	d.mu.RUnlock()
	if found && time.Now().Before(entry.expiresAt) && len(entry.ips) > 0 {
		return entry.ips, nil
	}

	// 3. Try system resolver first (requesting IPv4)
	ctxSys, cancelSys := context.WithTimeout(ctx, DefaultDNSWait)
	sysIPs, sysErr := net.DefaultResolver.LookupIP(ctxSys, "ip4", host)
	cancelSys()
	if sysErr == nil && len(sysIPs) > 0 {
		d.updateCache(host, sysIPs)
		return sysIPs, nil
	}

	// 4. Local DNS failed; try fallback public DNS servers
	for _, server := range d.dnsServers {
		ctxDns, cancelDns := context.WithTimeout(ctx, DefaultDNSWait)
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				dNet := net.Dialer{Timeout: 2 * time.Second}
				return dNet.DialContext(ctx, "udp", server)
			},
		}
		ips, err := r.LookupIP(ctxDns, "ip4", host)
		cancelDns()
		if err == nil && len(ips) > 0 {
			log.Printf("[netutil] Resolved %s via fallback DNS %s -> %v", host, server, ips)
			d.updateCache(host, ips)
			return ips, nil
		}
	}

	if sysErr != nil {
		return nil, sysErr
	}
	return nil, errors.New("netutil: no IPv4 addresses resolved")
}

// DialContext connects to the destination address, preferring IPv4 and utilizing DNS fallback.
func (d *ResilientDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return d.dialer.DialContext(ctx, "tcp4", addr)
	}

	ips, err := d.Resolve(ctx, host)
	if err != nil || len(ips) == 0 {
		// If custom resolution failed, attempt direct dial as final effort
		return d.dialer.DialContext(ctx, "tcp4", addr)
	}

	var lastErr error
	for _, ip := range ips {
		target := net.JoinHostPort(ip.String(), port)
		singleCtx, singleCancel := context.WithTimeout(ctx, DefaultSingleDial)
		conn, dialErr := d.dialer.DialContext(singleCtx, "tcp4", target)
		singleCancel()
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("netutil: connection failed to all resolved IPs")
}

// NewResilientTransport returns an http.Transport equipped with DNS fallback and IPv4 preference.
func NewResilientTransport() *http.Transport {
	dialer := NewResilientDialer()
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     false,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 35 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// NewHTTPClient creates an http.Client with resilient transport and the specified timeout.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: NewResilientTransport(),
		Timeout:   timeout,
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
