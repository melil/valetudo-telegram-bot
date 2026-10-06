package auth

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"tgbot/internal/bot/domain"
	"tgbot/internal/database"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
)

type Service struct {
	allowedChatID   int64
	db              domain.UserRepository
	tg              domain.Messenger
	getUserLang     func(chatID int64) i18n.Locale
	authReqMu       sync.Mutex
	lastAuthReqTime map[int64]time.Time
}

func NewService(allowedChatID int64, db domain.UserRepository, tg domain.Messenger, getUserLang ...func(chatID int64) i18n.Locale) *Service {
	var langFn func(chatID int64) i18n.Locale
	if len(getUserLang) > 0 {
		langFn = getUserLang[0]
	}
	return &Service{
		allowedChatID:   allowedChatID,
		db:              db,
		tg:              tg,
		getUserLang:     langFn,
		lastAuthReqTime: make(map[int64]time.Time),
	}
}

// IsUserAllowed проверяет наличие пользователя в БД или соответствие AllowedChatID.
func (s *Service) IsUserAllowed(chatID int64) bool {
	if s.db != nil {
		allowed, err := s.db.IsAllowed(chatID)
		if err == nil && allowed {
			return true
		}
	}
	return s.allowedChatID != 0 && chatID == s.allowedChatID
}

// IsUserAdmin проверяет, является ли пользователь администратором.
func (s *Service) IsUserAdmin(chatID int64) bool {
	if s.db != nil {
		isAdmin, err := s.db.IsAdmin(chatID)
		if err == nil && isAdmin {
			return true
		}
	}
	return s.allowedChatID != 0 && chatID == s.allowedChatID
}

func (s *Service) getUserLangForChat(chatID int64) i18n.Locale {
	if s.getUserLang != nil {
		return s.getUserLang(chatID)
	}
	if s.db != nil && chatID != 0 {
		if u, err := s.db.GetUser(chatID); err == nil && u != nil && u.Locale != "" {
			return i18n.NormalizeLocale(u.Locale)
		}
	}
	return i18n.LocaleRU
}

func (s *Service) tUser(chatID int64, key string, args ...any) string {
	return i18n.T(s.getUserLangForChat(chatID), key, args...)
}

// HandleUnauthorizedAccess обрабатывает входящее сообщение от неавторизованного пользователя.
func (s *Service) HandleUnauthorizedAccess(msg *telegram.Message) {
	if msg == nil {
		return
	}
	chatID := msg.Chat.ID
	cleanText := strings.TrimSpace(msg.Text)
	if cleanText != "/start" && !strings.HasPrefix(cleanText, "/start ") {
		_ = s.tg.DeleteMessage(chatID, msg.MessageID)
	}

	// 1. Отвечаем неавторизованному пользователю
	replyText := fmt.Sprintf(s.tUser(chatID, "auth.no_access"), chatID)
	_, _ = s.tg.SendTextMessage(chatID, replyText, false, nil)

	// 2. Дедупликация: отправляем уведомление админам не чаще раза в минуту для одного ChatID
	s.authReqMu.Lock()
	lastTime, exists := s.lastAuthReqTime[chatID]
	if exists && time.Since(lastTime) < 1*time.Minute {
		s.authReqMu.Unlock()
		return
	}
	s.lastAuthReqTime[chatID] = time.Now()
	s.authReqMu.Unlock()

	// 3. Собираем информацию о пользователе
	var username, fullName string
	if msg.From != nil {
		username = msg.From.Username
		fullName = strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName)
	}

	userDisplay := ""
	if username != "" {
		userDisplay = "@" + username
	}

	// 4. Отправляем запрос администраторам
	adminIDs := s.GetAdminChatIDs()
	for _, adminID := range adminIDs {
		adminUserDisplay := userDisplay
		if adminUserDisplay == "" {
			adminUserDisplay = s.tUser(adminID, "auth.no_username")
		}
		adminFullName := fullName
		if adminFullName == "" {
			adminFullName = s.tUser(adminID, "auth.name_not_specified")
		}

		adminText := fmt.Sprintf(
			s.tUser(adminID, "auth.request_title"),
			adminUserDisplay, adminFullName, chatID,
		)

		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: s.tUser(adminID, "auth.btn_approve"), CallbackData: fmt.Sprintf("auth_approve:%d:%s", chatID, username)},
					{Text: s.tUser(adminID, "auth.btn_reject"), CallbackData: fmt.Sprintf("auth_reject:%d", chatID)},
				},
			},
		}

		_, _ = s.tg.SendTextMessage(adminID, adminText, false, markup)
	}
}

// HandleAuthCallback обрабатывает нажатия инлайн-кнопок одобрения/отклонения доступа.
func (s *Service) HandleAuthCallback(cb *telegram.CallbackQuery) bool {
	if !strings.HasPrefix(cb.Data, "auth_approve:") && !strings.HasPrefix(cb.Data, "auth_reject:") {
		return false
	}

	// Проверяем, что кнопку нажал администратор
	if !s.IsUserAdmin(cb.From.ID) {
		_ = s.tg.AnswerCallbackQueryAlert(cb.ID, s.tUser(cb.From.ID, "auth.admin_only"), true)
		return true
	}

	parts := strings.Split(cb.Data, ":")
	action := parts[0]
	targetID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		_ = s.tg.AnswerCallbackQueryAlert(cb.ID, s.tUser(cb.From.ID, "auth.err_parse_id"), false)
		return true
	}

	var targetUsername string
	if len(parts) > 2 {
		targetUsername = parts[2]
	}

	switch action {
	case "auth_approve":
		if s.db != nil {
			if err := s.db.AddUser(targetID, targetUsername, database.RoleUser); err != nil {
				log.Printf("Ошибка добавления пользователя %d: %v", targetID, err)
				_ = s.tg.AnswerCallbackQueryAlert(cb.ID, s.tUser(cb.From.ID, "auth.err_db"), false)
				return true
			}
		}

		_ = s.tg.AnswerCallbackQuery(cb.ID)

		// Обновляем сообщение у админа
		userDisplay := fmt.Sprintf("ID: <code>%d</code>", targetID)
		if targetUsername != "" {
			userDisplay = fmt.Sprintf("@%s (ID: <code>%d</code>)", targetUsername, targetID)
		}
		if s.db != nil {
			_ = s.db.LogAction(cb.From.ID, cb.From.Username, "user_approved", userDisplay)
		}

		updatedText := fmt.Sprintf(s.tUser(cb.From.ID, "auth.approved_admin"), userDisplay)
		if cb.Message != nil {
			_ = s.tg.EditMessage(cb.From.ID, cb.Message.MessageID, updatedText, nil)
		}

		// Отправляем уведомление новому пользователю
		welcomeMsg := s.tUser(targetID, "auth.approved_user")
		_, _ = s.tg.SendTextMessage(targetID, welcomeMsg, false, nil)

	case "auth_reject":
		_ = s.tg.AnswerCallbackQuery(cb.ID)

		userDisplay := fmt.Sprintf("ID: %d", targetID)
		if targetUsername != "" {
			userDisplay = fmt.Sprintf("@%s (ID: %d)", targetUsername, targetID)
		}
		if s.db != nil {
			_ = s.db.LogAction(cb.From.ID, cb.From.Username, "user_rejected", userDisplay)
		}

		// Обновляем сообщение у админа
		updatedText := fmt.Sprintf(s.tUser(cb.From.ID, "auth.rejected_admin"), targetID)
		if cb.Message != nil {
			_ = s.tg.EditMessage(cb.From.ID, cb.Message.MessageID, updatedText, nil)
		}

		// Уведомляем пользователя об отказе
		rejectMsg := s.tUser(targetID, "auth.rejected_user")
		_, _ = s.tg.SendTextMessage(targetID, rejectMsg, false, nil)
	}

	return true
}

// GetAdminChatIDs возвращает список всех ID чатов администраторов.
func (s *Service) GetAdminChatIDs() []int64 {
	idsMap := make(map[int64]bool)

	if s.db != nil {
		admins, err := s.db.GetAdmins()
		if err == nil {
			for _, admin := range admins {
				idsMap[admin.ChatID] = true
			}
		}
	}

	if s.allowedChatID != 0 {
		idsMap[s.allowedChatID] = true
	}

	var res []int64
	for id := range idsMap {
		res = append(res, id)
	}
	return res
}

// GetAllUserChatIDs возвращает список всех ID чатов авторизованных пользователей бота.
func (s *Service) GetAllUserChatIDs() []int64 {
	idsMap := make(map[int64]bool)

	if s.db != nil {
		users, err := s.db.GetAllUsers()
		if err == nil {
			for _, u := range users {
				idsMap[u.ChatID] = true
			}
		}
	}

	if s.allowedChatID != 0 {
		idsMap[s.allowedChatID] = true
	}

	var res []int64
	for id := range idsMap {
		res = append(res, id)
	}
	return res
}

// GetNotifyChatIDs возвращает список chat_id пользователей, подписанных на категорию уведомлений.
func (s *Service) GetNotifyChatIDs(prefType string) []int64 {
	if s.db != nil {
		ids, err := s.db.GetSubscribedUsers(prefType)
		if err == nil {
			return ids
		}
	}
	return s.GetAllUserChatIDs()
}
