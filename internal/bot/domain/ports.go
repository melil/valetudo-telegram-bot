package domain

import (
	"io"
	"time"

	"tgbot/internal/database"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

// RobotClient abstracts communication with the robot controller (Valetudo).
type RobotClient interface {
	GetStatus() (valetudo.RobotStatus, error)
	GetAttributes() ([]valetudo.GenericAttribute, error)
	GetCapabilities() ([]string, error)
	TriggerAction(action string) error
	TriggerCapabilityAction(capability string, action string) error
	SetPreset(capability string, value string) error
	GetSegments() ([]valetudo.MapSegment, error)
	CleanSegments(segmentIDs []string, iterations int) error
	GetConsumables() ([]valetudo.ConsumableItem, error)
	GetConsumableProperties() (*valetudo.ConsumableProperties, error)
	ResetConsumable(cType, subType string) error
	GetMapReader() (io.ReadCloser, error)
	SetOperationMode(mode string) error
	SetMopWashTemperature(temp string) error
	SetMopDryingTime(duration string) error
	SetMopExtension(enable bool) error
	GetPresets(capability string) ([]string, error)
	GetMopWashTemperatureProperties() ([]string, error)
	GetMopDryingTimeProperties() ([]string, error)
	GetCurrentSessionStats() (int, int, float64)
	GetTotalStats() (int, int, float64)
}

// Messenger abstracts interactions with the Telegram Bot API.
type Messenger interface {
	GetUpdates(offset int) ([]telegram.Update, error)
	SendTextMessage(chatID int64, text string, disableNotification bool, markup any) (int, error)
	SendPayload(payload telegram.SendMessagePayload) (int, error)
	EditMessage(chatID int64, messageID int, text string, markup *telegram.InlineKeyboardMarkup) error
	DeleteMessage(chatID int64, messageID int) error
	SendPhoto(chatID int64, photo io.Reader, caption string, disableNotification bool) error
	AnswerCallbackQuery(callbackQueryID string) error
	AnswerCallbackQueryAlert(callbackQueryID string, text string, showAlert bool) error
}

// UserRepository abstracts database storage for users, roles, and preferences.
type UserRepository interface {
	IsAllowed(chatID int64) (bool, error)
	IsAdmin(chatID int64) (bool, error)
	GetUser(chatID int64) (*database.User, error)
	GetAllUsers() ([]database.User, error)
	GetAdmins() ([]database.User, error)
	GetSubscribedUsers(prefType string) ([]int64, error)
	AddUser(chatID int64, username string, role database.Role) error
	DeleteUser(chatID int64) error
	LogAction(chatID int64, username, action, details string) error
	SetUserLocale(chatID int64, locale string) error
	SetUserNotificationPref(chatID int64, prefType string, enabled bool) error
	SetUserDashboardMsgID(chatID int64, msgID int) error
	GetAllDashboardMsgIDs() (map[int64]int, error)
	GetRecentAuditLogs(limit int) ([]database.AuditLog, error)
	GetAuditLogsPaginated(offset, limit int) ([]database.AuditLog, int, error)
	GetAuditLogByID(id int64) (*database.AuditLog, error)
	BootstrapAdmin(defaultAdminChatID int64, username string) error
	GetMetadata(key string) (string, error)
	SetMetadata(key, value string) error
	DeleteMetadata(key string) error
}

// SystemCollector abstracts runtime and OS hardware statistics collection.
type SystemCollector interface {
	CollectRuntime(startTime time.Time) RuntimeStats
	CollectHost() HostStats
}
