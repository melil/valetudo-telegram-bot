package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"tgbot/internal/database"
	"tgbot/internal/telegram"
)

// isUserAllowed проверяет наличие пользователя в БД или соответствие AllowedChatID.
func (b *Bot) isUserAllowed(chatID int64) bool {
	if b.db != nil {
		allowed, err := b.db.IsAllowed(chatID)
		if err == nil && allowed {
			return true
		}
	}
	// Fallback на AllowedChatID из конфигурации
	return b.cfg.AllowedChatID != 0 && chatID == b.cfg.AllowedChatID
}

// isUserAdmin проверяет, является ли пользователь администратором.
func (b *Bot) isUserAdmin(chatID int64) bool {
	if b.db != nil {
		isAdmin, err := b.db.IsAdmin(chatID)
		if err == nil && isAdmin {
			return true
		}
	}
	return b.cfg.AllowedChatID != 0 && chatID == b.cfg.AllowedChatID
}

// handleUnauthorizedAccess обрабатывает входящее сообщение от неавторизованного пользователя.
func (b *Bot) handleUnauthorizedAccess(msg *telegram.Message) {
	if msg == nil {
		return
	}
	chatID := msg.Chat.ID
	_ = b.tg.DeleteMessage(chatID, msg.MessageID)

	// 1. Отвечаем неавторизованному пользователю
	replyText := fmt.Sprintf(b.tUser(chatID, "auth.no_access"), chatID)
	_, _ = b.tg.SendTextMessage(chatID, replyText, false, nil)

	// 2. Дедупликация: отправляем уведомление админам не чаще раза в минуту для одного ChatID
	b.authReqMu.Lock()
	lastTime, exists := b.lastAuthReqTime[chatID]
	if exists && time.Since(lastTime) < 1*time.Minute {
		b.authReqMu.Unlock()
		return
	}
	b.lastAuthReqTime[chatID] = time.Now()
	b.authReqMu.Unlock()

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
	adminIDs := b.getAdminChatIDs()
	for _, adminID := range adminIDs {
		adminUserDisplay := userDisplay
		if adminUserDisplay == "" {
			adminUserDisplay = b.tUser(adminID, "auth.no_username")
		}
		adminFullName := fullName
		if adminFullName == "" {
			adminFullName = b.tUser(adminID, "auth.name_not_specified")
		}

		adminText := fmt.Sprintf(
			b.tUser(adminID, "auth.request_title"),
			adminUserDisplay, adminFullName, chatID,
		)

		markup := &telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{
				{
					{Text: b.tUser(adminID, "auth.btn_approve"), CallbackData: fmt.Sprintf("auth_approve:%d:%s", chatID, username)},
					{Text: b.tUser(adminID, "auth.btn_reject"), CallbackData: fmt.Sprintf("auth_reject:%d", chatID)},
				},
			},
		}

		_, _ = b.tg.SendTextMessage(adminID, adminText, false, markup)
	}
}

// handleAuthCallback обрабатывает нажатия инлайн-кнопок одобрения/отклонения доступа.
func (b *Bot) handleAuthCallback(cb *telegram.CallbackQuery) bool {
	if !strings.HasPrefix(cb.Data, "auth_approve:") && !strings.HasPrefix(cb.Data, "auth_reject:") {
		return false
	}

	// Проверяем, что кнопку нажал администратор
	if !b.isUserAdmin(cb.From.ID) {
		_ = b.tg.AnswerCallbackQueryAlert(cb.ID, b.tUser(cb.From.ID, "auth.admin_only"), true)
		return true
	}

	parts := strings.Split(cb.Data, ":")
	action := parts[0]
	targetID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		_ = b.tg.AnswerCallbackQueryAlert(cb.ID, b.tUser(cb.From.ID, "auth.err_parse_id"), false)
		return true
	}

	var targetUsername string
	if len(parts) > 2 {
		targetUsername = parts[2]
	}

	switch action {
	case "auth_approve":
		if b.db != nil {
			if err := b.db.AddUser(targetID, targetUsername, database.RoleUser); err != nil {
				log.Printf("Ошибка добавления пользователя %d: %v", targetID, err)
				_ = b.tg.AnswerCallbackQueryAlert(cb.ID, b.tUser(cb.From.ID, "auth.err_db"), false)
				return true
			}
		}

		_ = b.tg.AnswerCallbackQuery(cb.ID)

		// Обновляем сообщение у админа
		userDisplay := fmt.Sprintf("ID: <code>%d</code>", targetID)
		if targetUsername != "" {
			userDisplay = fmt.Sprintf("@%s (ID: <code>%d</code>)", targetUsername, targetID)
		}
		b.LogAction(cb.From.ID, "user_approved", userDisplay)

		updatedText := fmt.Sprintf(b.tUser(cb.From.ID, "auth.approved_admin"), userDisplay)
		if cb.Message != nil {
			_ = b.tg.EditMessage(cb.From.ID, cb.Message.MessageID, updatedText, nil)
		}

		// Отправляем уведомление новому пользователю
		welcomeMsg := b.tUser(targetID, "auth.approved_user")
		_, _ = b.tg.SendTextMessage(targetID, welcomeMsg, false, nil)

	case "auth_reject":
		_ = b.tg.AnswerCallbackQuery(cb.ID)

		userDisplay := fmt.Sprintf("ID: %d", targetID)
		if targetUsername != "" {
			userDisplay = fmt.Sprintf("@%s (ID: %d)", targetUsername, targetID)
		}
		b.LogAction(cb.From.ID, "user_rejected", userDisplay)

		// Обновляем сообщение у админа
		updatedText := fmt.Sprintf(b.tUser(cb.From.ID, "auth.rejected_admin"), targetID)
		if cb.Message != nil {
			_ = b.tg.EditMessage(cb.From.ID, cb.Message.MessageID, updatedText, nil)
		}

		// Уведомляем пользователя об отказе
		rejectMsg := b.tUser(targetID, "auth.rejected_user")
		_, _ = b.tg.SendTextMessage(targetID, rejectMsg, false, nil)
	}

	return true
}

// getAdminChatIDs возвращает список всех ID чатов администраторов.
func (b *Bot) getAdminChatIDs() []int64 {
	idsMap := make(map[int64]bool)

	if b.db != nil {
		admins, err := b.db.GetAdmins()
		if err == nil {
			for _, admin := range admins {
				idsMap[admin.ChatID] = true
			}
		}
	}

	if b.cfg.AllowedChatID != 0 {
		idsMap[b.cfg.AllowedChatID] = true
	}

	var res []int64
	for id := range idsMap {
		res = append(res, id)
	}
	return res
}

// getAllUserChatIDs возвращает список всех ID чатов авторизованных пользователей бота.
func (b *Bot) getAllUserChatIDs() []int64 {
	idsMap := make(map[int64]bool)

	if b.db != nil {
		users, err := b.db.GetAllUsers()
		if err == nil {
			for _, u := range users {
				idsMap[u.ChatID] = true
			}
		}
	}

	if b.cfg.AllowedChatID != 0 {
		idsMap[b.cfg.AllowedChatID] = true
	}

	var res []int64
	for id := range idsMap {
		res = append(res, id)
	}
	return res
}

// getNotifyChatIDs возвращает список chat_id пользователей, подписанных на категорию уведомлений.
func (b *Bot) getNotifyChatIDs(prefType string) []int64 {
	if b.db != nil {
		ids, err := b.db.GetSubscribedUsers(prefType)
		if err == nil {
			return ids
		}
	}
	return b.getAllUserChatIDs()
}

// getMainDashboardForChat возвращает главное меню с учетом персонального языка чата.
func (b *Bot) getMainDashboardForChat(chatID int64) (string, *telegram.InlineKeyboardMarkup) {
	b.SetActiveChatID(chatID)
	return b.getMainDashboard()
}

// broadcastTextMessage отправляет текстовое сообщение всем авторизованным пользователям.
func (b *Bot) broadcastTextMessage(text string, disableNotification bool) {
	for _, chatID := range b.getAllUserChatIDs() {
		_, _ = b.tg.SendTextMessage(chatID, text, disableNotification, nil)
	}
}

// broadcastMainDashboard обновляет главное меню у всех пользователей с учетом персонального языка каждого.
func (b *Bot) broadcastMainDashboard() {
	for _, chatID := range b.getAllUserChatIDs() {
		text, markup := b.getMainDashboardForChat(chatID)
		_ = b.renderDashboardForChat(chatID, text, markup)
	}
}
