package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"tgbot/internal/bot/domain"
	"tgbot/internal/bot/service/auth"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/version"
)

// Config contains settings for GitHub updates.
type Config struct {
	Repo          string
	GitHubToken   string
	CheckInterval time.Duration
	AutoNotify    bool
	AllowedChatID int64
}

// ReleaseInfo contains metadata about a release on GitHub.
type ReleaseInfo struct {
	TagName     string
	Version     string
	Name        string
	Body        string
	HTMLURL     string
	PublishedAt time.Time
	AssetURL    string
	AssetName   string
	AssetSize   int64
}

// Service manages checking, downloading and applying bot updates.
type Service struct {
	cfg         Config
	db          domain.UserRepository
	tg          domain.Messenger
	authSvc     *auth.Service
	apiClient   *http.Client
	downloadClient *http.Client
	getUserLang func(chatID int64) i18n.Locale

	isUpdating  bool
	mu          sync.Mutex

	// Test hooks
	exePathFunc func() (string, error)
	restartFunc func()
	apiBaseURL  string
}

// NewService creates a new update Service.
func NewService(
	cfg Config,
	db domain.UserRepository,
	tg domain.Messenger,
	authSvc *auth.Service,
	getUserLang func(chatID int64) i18n.Locale,
) *Service {
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 6 * time.Hour
	}
	if cfg.Repo == "" {
		cfg.Repo = "melil/valetudo-telegram-bot"
	}

	downloadClient := &http.Client{
		Timeout: 0, // No client-level hard timeout; controlled by context.WithTimeout
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			// When redirecting across hosts (e.g. github.com -> s3.amazonaws.com), drop Authorization header
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				req.Header.Del("Authorization")
			}
			return nil
		},
	}

	return &Service{
		cfg:            cfg,
		db:             db,
		tg:             tg,
		authSvc:        authSvc,
		apiClient:      &http.Client{Timeout: 30 * time.Second},
		downloadClient: downloadClient,
		getUserLang:    getUserLang,
		exePathFunc:    os.Executable,
		restartFunc: func() {
			log.Println("Restarting bot process after update...")
			os.Exit(0)
		},
		apiBaseURL: "https://api.github.com",
	}
}

type ghReleaseAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghReleaseResponse struct {
	TagName     string           `json:"tag_name"`
	Name        string           `json:"name"`
	Body        string           `json:"body"`
	HTMLURL     string           `json:"html_url"`
	PublishedAt time.Time        `json:"published_at"`
	Assets      []ghReleaseAsset `json:"assets"`
}

// CheckForUpdate checks GitHub Releases API for a newer version.
func (s *Service) CheckForUpdate(ctx context.Context) (*ReleaseInfo, bool, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", s.apiBaseURL, s.cfg.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Valetudo-Telegram-Bot")
	if s.cfg.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.GitHubToken)
	}

	resp, err := s.apiClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to query GitHub Releases API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, errors.New("no releases found on GitHub")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, false, fmt.Errorf("GitHub API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ghRel ghReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&ghRel); err != nil {
		return nil, false, fmt.Errorf("failed to parse GitHub release JSON: %w", err)
	}

	relVersion := version.Normalize(ghRel.TagName)
	curVersion := version.Normalize(version.Version)

	info := &ReleaseInfo{
		TagName:     ghRel.TagName,
		Version:     relVersion,
		Name:        ghRel.Name,
		Body:        strings.TrimSpace(ghRel.Body),
		HTMLURL:     ghRel.HTMLURL,
		PublishedAt: ghRel.PublishedAt,
	}

	// Find the Linux ARM64 binary asset
	for _, asset := range ghRel.Assets {
		nameLower := strings.ToLower(asset.Name)
		if nameLower == "tgbot" || strings.Contains(nameLower, "arm64") || strings.HasPrefix(nameLower, "tgbot") {
			info.AssetURL = asset.BrowserDownloadURL
			info.AssetName = asset.Name
			info.AssetSize = asset.Size
			break
		}
	}

	// Fallback to sole asset if only 1 exists
	if info.AssetURL == "" && len(ghRel.Assets) == 1 {
		info.AssetURL = ghRel.Assets[0].BrowserDownloadURL
		info.AssetName = ghRel.Assets[0].Name
		info.AssetSize = ghRel.Assets[0].Size
	}

	hasUpdate := version.IsNewer(relVersion, curVersion)
	return info, hasUpdate, nil
}

// DismissVersion marks a version as dismissed in DB so notifications are snoozed.
func (s *Service) DismissVersion(ver string) error {
	if s.db == nil {
		return nil
	}
	return s.db.SetMetadata("dismissed_update_version", version.Normalize(ver))
}

// IsVersionDismissed checks if a version was dismissed by admin.
func (s *Service) IsVersionDismissed(ver string) bool {
	if s.db == nil {
		return false
	}
	dismissed, err := s.db.GetMetadata("dismissed_update_version")
	if err != nil || dismissed == "" {
		return false
	}
	return dismissed == version.Normalize(ver)
}

// ApplyUpdate downloads the new binary, replaces the current executable and triggers restart.
func (s *Service) ApplyUpdate(ctx context.Context, rel *ReleaseInfo, chatID int64) error {
	s.mu.Lock()
	if s.isUpdating {
		s.mu.Unlock()
		return errors.New("update is already in progress")
	}
	s.isUpdating = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isUpdating = false
		s.mu.Unlock()
	}()

	if rel.AssetURL == "" {
		return errors.New("no deployable binary found in release assets")
	}

	targetPath, err := s.exePathFunc()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	targetPath, err = filepath.EvalSymlinks(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable symlinks: %w", err)
	}

	targetDir := filepath.Dir(targetPath)
	tempPath := filepath.Join(targetDir, fmt.Sprintf("tgbot_new_%d", time.Now().UnixNano()))

	// Download binary with timeout (up to 15 minutes)
	downloadCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, rel.AssetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "Valetudo-Telegram-Bot")
	req.Header.Set("Accept", "application/octet-stream")
	if s.cfg.GitHubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.GitHubToken)
	}

	resp, err := s.downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download release asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download asset failed with HTTP %d", resp.StatusCode)
	}

	outFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temporary binary file: %w", err)
	}

	buf := make([]byte, 128*1024)
	written, err := io.CopyBuffer(outFile, resp.Body, buf)
	_ = outFile.Sync()
	_ = outFile.Close()

	if err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to write binary: %w", err)
	}

	log.Printf("Update binary downloaded successfully: %d bytes", written)

	if written == 0 {
		_ = os.Remove(tempPath)
		return errors.New("downloaded binary is empty")
	}

	// Make executable
	if err := os.Chmod(tempPath, 0755); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Replace existing binary atomically
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to replace running binary: %w", err)
	}

	// Save update state in DB
	if s.db != nil {
		_ = s.db.SetMetadata("pending_update_version", rel.Version)
		_ = s.db.SetMetadata("pending_update_chat_id", strconv.FormatInt(chatID, 10))
		_ = s.db.LogAction(chatID, "admin", "update_installed", "Installed version "+rel.Version)
	}

	// Schedule process exit to allow run.sh supervisor to restart tgbot
	go func() {
		time.Sleep(1500 * time.Millisecond)
		s.restartFunc()
	}()

	return nil
}

// CheckAndNotifyPostUpdate checks on bot startup if an update was just completed.
func (s *Service) CheckAndNotifyPostUpdate(ctx context.Context) {
	if s.db == nil {
		return
	}

	pendingVer, err := s.db.GetMetadata("pending_update_version")
	if err != nil || pendingVer == "" {
		return
	}

	chatIDStr, _ := s.db.GetMetadata("pending_update_chat_id")
	_ = s.db.DeleteMetadata("pending_update_version")
	_ = s.db.DeleteMetadata("pending_update_chat_id")

	var chatID int64
	if chatIDStr != "" {
		if parsed, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			chatID = parsed
		}
	}
	if chatID == 0 {
		chatID = s.cfg.AllowedChatID
	}

	curVer := version.Normalize(version.Version)
	pVer := version.Normalize(pendingVer)

	if curVer == pVer && chatID != 0 {
		var loc i18n.Locale
		if s.getUserLang != nil {
			loc = s.getUserLang(chatID)
		} else {
			loc = i18n.LocaleRU
		}

		msg := fmt.Sprintf(i18n.T(loc, "updates.post_update_success"), version.Version)
		_, _ = s.tg.SendTextMessage(chatID, msg, false, nil)
		_ = s.db.LogAction(chatID, "system", "update_applied", "Successfully running version "+version.Version)
	}
}

// Start runs background update checking every CheckInterval.
func (s *Service) Start(ctx context.Context) {
	if !s.cfg.AutoNotify {
		log.Println("UpdateService: automatic background update checking disabled")
		return
	}

	log.Printf("UpdateService: starting background checker (interval: %v, repo: %s)", s.cfg.CheckInterval, s.cfg.Repo)

	// Initial delay before first check
	select {
	case <-ctx.Done():
		return
	case <-time.After(1 * time.Minute):
		s.runBackgroundCheck(ctx)
	}

	ticker := time.NewTicker(s.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("UpdateService: background check stopped")
			return
		case <-ticker.C:
			s.runBackgroundCheck(ctx)
		}
	}
}

func (s *Service) runBackgroundCheck(ctx context.Context) {
	rel, hasUpdate, err := s.CheckForUpdate(ctx)
	if err != nil {
		log.Printf("UpdateService: error checking for updates: %v", err)
		return
	}

	if !hasUpdate {
		return
	}

	// Check if this version was already dismissed
	if s.IsVersionDismissed(rel.Version) {
		log.Printf("UpdateService: version %s is dismissed by admin, skipping auto-notification", rel.Version)
		return
	}

	adminIDs := s.getAdminChatIDs()
	if len(adminIDs) == 0 && s.cfg.AllowedChatID != 0 {
		adminIDs = []int64{s.cfg.AllowedChatID}
	}

	log.Printf("UpdateService: found new version %s! Notifying %d admin(s)...", rel.Version, len(adminIDs))

	for _, adminID := range adminIDs {
		var loc i18n.Locale
		if s.getUserLang != nil {
			loc = s.getUserLang(adminID)
		} else {
			loc = i18n.LocaleRU
		}

		bodySnippet := rel.Body
		if len(bodySnippet) > 400 {
			bodySnippet = bodySnippet[:400] + "..."
		}
		if bodySnippet == "" {
			bodySnippet = i18n.T(loc, "updates.no_changelog")
		}

		text := fmt.Sprintf(i18n.T(loc, "updates.notification_text"), version.Version, rel.Version, bodySnippet)
		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: i18n.T(loc, "updates.btn_update_now"), CallbackData: "action_update_bot:" + rel.Version},
					{Text: i18n.T(loc, "updates.btn_later"), CallbackData: "action_update_later:" + rel.Version},
				},
			},
		}

		_, _ = s.tg.SendTextMessage(adminID, text, false, markup)
	}
}

func (s *Service) getAdminChatIDs() []int64 {
	if s.authSvc != nil {
		return s.authSvc.GetAdminChatIDs()
	}
	if s.db != nil {
		admins, err := s.db.GetAdmins()
		if err == nil && len(admins) > 0 {
			ids := make([]int64, 0, len(admins))
			for _, a := range admins {
				ids = append(ids, a.ChatID)
			}
			return ids
		}
	}
	return nil
}
