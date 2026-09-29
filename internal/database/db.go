package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ChatID         int64     `json:"chat_id"`
	Username       string    `json:"username"`
	Role           Role      `json:"role"`
	Locale         string    `json:"locale"`
	NotifyErrors   bool      `json:"notify_errors"`
	NotifyReports  bool      `json:"notify_reports"`
	NotifyStation  bool      `json:"notify_station"`
	DashboardMsgID int       `json:"dashboard_msg_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type AuditLog struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

type DB struct {
	db *sql.DB
}

// Open открывает соединение с SQLite БД, настраивает WAL-режим и инициализирует схему.
func Open(dbPath string) (*DB, error) {
	if dbPath == "" {
		dbPath = "bot.db"
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	d := &DB{db: db}
	if err := d.InitSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to init database schema: %w", err)
	}

	return d, nil
}

// InitSchema создаёт таблицы users и audit_logs, а также накатывает безопасные миграции колонок.
func (d *DB) InitSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		chat_id INTEGER PRIMARY KEY,
		username TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL CHECK(role IN ('admin', 'user')),
		locale TEXT NOT NULL DEFAULT 'ru',
		notify_errors INTEGER NOT NULL DEFAULT 1,
		notify_reports INTEGER NOT NULL DEFAULT 1,
		notify_station INTEGER NOT NULL DEFAULT 1,
		dashboard_msg_id INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		username TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		details TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_id ON audit_logs(id DESC);

	CREATE TABLE IF NOT EXISTS bot_metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	if _, err := d.db.Exec(query); err != nil {
		return err
	}

	// Идемпотентная миграция колонок для ранее созданных баз
	addColumnIfNotExists(d.db, "users", "locale", "TEXT NOT NULL DEFAULT 'ru'")
	addColumnIfNotExists(d.db, "users", "notify_errors", "INTEGER NOT NULL DEFAULT 1")
	addColumnIfNotExists(d.db, "users", "notify_reports", "INTEGER NOT NULL DEFAULT 1")
	addColumnIfNotExists(d.db, "users", "notify_station", "INTEGER NOT NULL DEFAULT 1")
	addColumnIfNotExists(d.db, "users", "dashboard_msg_id", "INTEGER NOT NULL DEFAULT 0")

	return nil
}

func addColumnIfNotExists(db *sql.DB, table, column, colDef string) {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colDef)
	_, _ = db.Exec(query)
}

// BootstrapAdmin проверяет наличие пользователей. Если таблица пуста,
// добавляет начальный chat_id из ENV с ролью 'admin'.
func (d *DB) BootstrapAdmin(defaultAdminChatID int64, username string) error {
	if defaultAdminChatID <= 0 {
		return nil
	}

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count == 0 {
		return d.AddUser(defaultAdminChatID, username, RoleAdmin)
	}
	return nil
}

// GetUser возвращает пользователя по chat_id или ErrUserNotFound.
func (d *DB) GetUser(chatID int64) (*User, error) {
	row := d.db.QueryRow(`
		SELECT chat_id, username, role, locale, notify_errors, notify_reports, notify_station, dashboard_msg_id, created_at 
		FROM users WHERE chat_id = ?`, chatID)

	var u User
	var roleStr string
	var nErr, nRep, nSta int
	err := row.Scan(&u.ChatID, &u.Username, &roleStr, &u.Locale, &nErr, &nRep, &nSta, &u.DashboardMsgID, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	u.Role = Role(roleStr)
	u.NotifyErrors = nErr == 1
	u.NotifyReports = nRep == 1
	u.NotifyStation = nSta == 1
	return &u, nil
}

// IsAllowed проверяет, зарегистрирован ли пользователь в БД.
func (d *DB) IsAllowed(chatID int64) (bool, error) {
	var exists int
	err := d.db.QueryRow("SELECT 1 FROM users WHERE chat_id = ? LIMIT 1", chatID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsAdmin проверяет, является ли пользователь администратором.
func (d *DB) IsAdmin(chatID int64) (bool, error) {
	u, err := d.GetUser(chatID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return false, nil
		}
		return false, err
	}
	return u.Role == RoleAdmin, nil
}

// AddUser добавляет или обновляет роль и имя пользователя.
func (d *DB) AddUser(chatID int64, username string, role Role) error {
	query := `
	INSERT INTO users (chat_id, username, role, locale, notify_errors, notify_reports, notify_station, dashboard_msg_id, created_at)
	VALUES (?, ?, ?, 'ru', 1, 1, 1, 0, ?)
	ON CONFLICT(chat_id) DO UPDATE SET
		role = excluded.role,
		username = excluded.username;
	`
	_, err := d.db.Exec(query, chatID, username, string(role), time.Now().UTC())
	return err
}

// SetUserLocale сохраняет персональный язык интерфейса пользователя.
func (d *DB) SetUserLocale(chatID int64, locale string) error {
	_, err := d.db.Exec("UPDATE users SET locale = ? WHERE chat_id = ?", locale, chatID)
	return err
}

// SetUserDashboardMsgID сохраняет ID активного сообщения дашборда для пользователя.
func (d *DB) SetUserDashboardMsgID(chatID int64, msgID int) error {
	_, err := d.db.Exec("UPDATE users SET dashboard_msg_id = ? WHERE chat_id = ?", msgID, chatID)
	return err
}

// GetAllDashboardMsgIDs возвращает мапу сохраненных message_id дашбордов для всех пользователей.
func (d *DB) GetAllDashboardMsgIDs() (map[int64]int, error) {
	rows, err := d.db.Query("SELECT chat_id, dashboard_msg_id FROM users WHERE dashboard_msg_id > 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[int64]int)
	for rows.Next() {
		var chatID int64
		var msgID int
		if err := rows.Scan(&chatID, &msgID); err != nil {
			return nil, err
		}
		res[chatID] = msgID
	}
	return res, rows.Err()
}

// SetUserNotificationPref переключает подписку пользователя на категорию уведомлений (errors, reports, station).
func (d *DB) SetUserNotificationPref(chatID int64, prefType string, enabled bool) error {
	var col string
	switch prefType {
	case "errors":
		col = "notify_errors"
	case "reports":
		col = "notify_reports"
	case "station":
		col = "notify_station"
	default:
		return fmt.Errorf("unknown notification pref type: %s", prefType)
	}

	val := 0
	if enabled {
		val = 1
	}

	query := fmt.Sprintf("UPDATE users SET %s = ? WHERE chat_id = ?", col)
	_, err := d.db.Exec(query, val, chatID)
	return err
}

// GetSubscribedUsers возвращает список chat_id пользователей, подписанных на категорию уведомлений.
func (d *DB) GetSubscribedUsers(prefType string) ([]int64, error) {
	var col string
	switch prefType {
	case "errors":
		col = "notify_errors"
	case "reports":
		col = "notify_reports"
	case "station":
		col = "notify_station"
	default:
		return nil, fmt.Errorf("unknown notification pref type: %s", prefType)
	}

	query := fmt.Sprintf("SELECT chat_id FROM users WHERE %s = 1", col)
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LogAction записывает действие пользователя в журнал аудита и автоматически держит лимит до 100 записей.
func (d *DB) LogAction(chatID int64, username, action, details string) error {
	query := `INSERT INTO audit_logs (chat_id, username, action, details, created_at) VALUES (?, ?, ?, ?, ?)`
	if _, err := d.db.Exec(query, chatID, username, action, details, time.Now().UTC()); err != nil {
		return err
	}

	// Кольцевая очистка: удаляем всё за пределами последних 100 записей
	cleanupQuery := `DELETE FROM audit_logs WHERE id NOT IN (SELECT id FROM audit_logs ORDER BY id DESC LIMIT 100)`
	_, _ = d.db.Exec(cleanupQuery)
	return nil
}

// GetRecentAuditLogs возвращает последние N записей журнала аудита.
func (d *DB) GetRecentAuditLogs(limit int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 15
	}
	rows, err := d.db.Query(`SELECT id, chat_id, username, action, details, created_at FROM audit_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.ChatID, &a.Username, &a.Action, &a.Details, &a.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, a)
	}
	return logs, rows.Err()
}

// GetAdmins возвращает список всех администраторов (для рассылки запросов доступа).
func (d *DB) GetAdmins() ([]User, error) {
	rows, err := d.db.Query(`
		SELECT chat_id, username, role, locale, notify_errors, notify_reports, notify_station, dashboard_msg_id, created_at 
		FROM users WHERE role = 'admin'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []User
	for rows.Next() {
		var u User
		var roleStr string
		var nErr, nRep, nSta int
		if err := rows.Scan(&u.ChatID, &u.Username, &roleStr, &u.Locale, &nErr, &nRep, &nSta, &u.DashboardMsgID, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Role = Role(roleStr)
		u.NotifyErrors = nErr == 1
		u.NotifyReports = nRep == 1
		u.NotifyStation = nSta == 1
		admins = append(admins, u)
	}
	return admins, rows.Err()
}

// GetAllUsers возвращает список всех пользователей бота.
func (d *DB) GetAllUsers() ([]User, error) {
	rows, err := d.db.Query(`
		SELECT chat_id, username, role, locale, notify_errors, notify_reports, notify_station, dashboard_msg_id, created_at 
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var roleStr string
		var nErr, nRep, nSta int
		if err := rows.Scan(&u.ChatID, &u.Username, &roleStr, &u.Locale, &nErr, &nRep, &nSta, &u.DashboardMsgID, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Role = Role(roleStr)
		u.NotifyErrors = nErr == 1
		u.NotifyReports = nRep == 1
		u.NotifyStation = nSta == 1
		users = append(users, u)
	}
	return users, rows.Err()
}

// DeleteUser удаляет пользователя по chat_id.
func (d *DB) DeleteUser(chatID int64) error {
	_, err := d.db.Exec("DELETE FROM users WHERE chat_id = ?", chatID)
	return err
}

// GetMetadata получает значение ключа из bot_metadata.
func (d *DB) GetMetadata(key string) (string, error) {
	var val string
	err := d.db.QueryRow("SELECT value FROM bot_metadata WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetMetadata устанавливает или обновляет значение ключа в bot_metadata.
func (d *DB) SetMetadata(key, value string) error {
	_, err := d.db.Exec(`
		INSERT INTO bot_metadata (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// DeleteMetadata удаляет ключ из bot_metadata.
func (d *DB) DeleteMetadata(key string) error {
	_, err := d.db.Exec("DELETE FROM bot_metadata WHERE key = ?", key)
	return err
}

// Close закрывает соединение с БД.
func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
