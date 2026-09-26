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
	ChatID    int64     `json:"chat_id"`
	Username  string    `json:"username"`
	Role      Role      `json:"role"`
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

	// modernc.org/sqlite использует имя драйвера "sqlite"
	// _pragma=busy_timeout=5000&_pragma=journal_mode=WAL для надежности и параллельного чтения
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite в одном файле лучше всего работает с ограничением на параллельную запись
	db.SetMaxOpenConns(1)

	d := &DB{db: db}
	if err := d.InitSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to init database schema: %w", err)
	}

	return d, nil
}

// InitSchema создаёт таблицу users, если она ещё не существует.
func (d *DB) InitSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		chat_id INTEGER PRIMARY KEY,
		username TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL CHECK(role IN ('admin', 'user')),
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
	`
	_, err := d.db.Exec(query)
	return err
}

// BootstrapAdmin проверяет, есть ли записи в таблице users.
// Если таблица пуста и передан начальный defaultAdminChatID > 0, добавляет его как 'admin'.
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
	row := d.db.QueryRow("SELECT chat_id, username, role, created_at FROM users WHERE chat_id = ?", chatID)

	var u User
	var roleStr string
	err := row.Scan(&u.ChatID, &u.Username, &roleStr, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	u.Role = Role(roleStr)
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
	INSERT INTO users (chat_id, username, role, created_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(chat_id) DO UPDATE SET
		role = excluded.role,
		username = excluded.username;
	`
	_, err := d.db.Exec(query, chatID, username, string(role), time.Now().UTC())
	return err
}

// GetAdmins возвращает список всех администраторов (для рассылки уведомлений).
func (d *DB) GetAdmins() ([]User, error) {
	rows, err := d.db.Query("SELECT chat_id, username, role, created_at FROM users WHERE role = 'admin'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []User
	for rows.Next() {
		var u User
		var roleStr string
		if err := rows.Scan(&u.ChatID, &u.Username, &roleStr, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Role = Role(roleStr)
		admins = append(admins, u)
	}
	return admins, rows.Err()
}

// GetAllUsers возвращает список всех пользователей бота.
func (d *DB) GetAllUsers() ([]User, error) {
	rows, err := d.db.Query("SELECT chat_id, username, role, created_at FROM users ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var roleStr string
		if err := rows.Scan(&u.ChatID, &u.Username, &roleStr, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Role = Role(roleStr)
		users = append(users, u)
	}
	return users, rows.Err()
}

// DeleteUser удаляет пользователя по chat_id.
func (d *DB) DeleteUser(chatID int64) error {
	_, err := d.db.Exec("DELETE FROM users WHERE chat_id = ?", chatID)
	return err
}

// Close закрывает соединение с БД.
func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
