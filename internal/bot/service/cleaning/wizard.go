package cleaning

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"

	"tgbot/internal/bot/domain"
	"tgbot/internal/i18n"
)

type WizardService struct {
	val         domain.RobotClient
	roomAliases map[string]string

	mu       sync.Mutex
	sessions map[int64]*domain.WizardSession
}

func NewWizardService(val domain.RobotClient, roomAliases map[string]string) *WizardService {
	if roomAliases == nil {
		roomAliases = make(map[string]string)
	}
	return &WizardService{
		val:         val,
		roomAliases: roomAliases,
		sessions:    make(map[int64]*domain.WizardSession),
	}
}

// GetRooms возвращает отсортированный список комнат с учетом алиасов из конфигурации.
func (s *WizardService) GetRooms(loc i18n.Locale) ([]domain.RoomInfo, error) {
	segments, err := s.val.GetSegments()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T(loc, "rooms.err_get", ""), err)
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("%s", i18n.T(loc, "rooms.empty"))
	}

	var rooms []domain.RoomInfo
	for _, seg := range segments {
		name := strings.TrimSpace(seg.Name)
		if alias, ok := s.roomAliases[seg.ID]; ok && alias != "" {
			name = alias
		} else if alias, ok := s.roomAliases[seg.Name]; ok && alias != "" {
			name = alias
		}
		if name == "" {
			name = fmt.Sprintf(i18n.T(loc, "wizard.room_default"), seg.ID)
		}
		rooms = append(rooms, domain.RoomInfo{ID: seg.ID, Name: name})
	}

	sort.Slice(rooms, func(i, j int) bool {
		id1, err1 := strconv.Atoi(rooms[i].ID)
		id2, err2 := strconv.Atoi(rooms[j].ID)
		if err1 == nil && err2 == nil {
			return id1 < id2
		}
		return rooms[i].ID < rooms[j].ID
	})
	return rooms, nil
}

func (s *WizardService) StartSession(chatID int64, messageID int, rooms []domain.RoomInfo) *domain.WizardSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &domain.WizardSession{
		SelectedRooms: make(map[string]bool),
		Rooms:         rooms,
		Iterations:    1,
		MessageID:     messageID,
	}
	s.sessions[chatID] = session
	return session
}

func (s *WizardService) GetSession(chatID int64) (*domain.WizardSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[chatID]
	return sess, ok
}

func (s *WizardService) CancelSession(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, chatID)
}

func (s *WizardService) SetMode(chatID int64, mode string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[chatID]
	if !ok || sess == nil {
		return false
	}
	sess.Mode = mode
	return true
}

func (s *WizardService) ToggleRoom(chatID int64, roomID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[chatID]
	if !ok || sess == nil {
		return false
	}
	sess.SelectedRooms[roomID] = !sess.SelectedRooms[roomID]
	return true
}

func (s *WizardService) SelectAll(chatID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[chatID]
	if !ok || sess == nil {
		return false
	}
	for _, r := range sess.Rooms {
		sess.SelectedRooms[r.ID] = true
	}
	return true
}

func (s *WizardService) SelectNone(chatID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[chatID]
	if !ok || sess == nil {
		return false
	}
	sess.SelectedRooms = make(map[string]bool)
	return true
}

// ExecuteCleaning запускает сегментную уборку выбранных комнат.
func (s *WizardService) ExecuteCleaning(chatID int64, iterations int, hasModeCap bool) (mode string, targetNames []string, iterCount int, err error) {
	s.mu.Lock()
	sess, ok := s.sessions[chatID]
	if ok && sess != nil {
		delete(s.sessions, chatID)
	}
	s.mu.Unlock()

	if !ok || sess == nil {
		return "", nil, 0, fmt.Errorf("session expired")
	}

	if iterations <= 0 {
		iterations = 1
	}

	var targetIDs []string
	for _, r := range sess.Rooms {
		if sess.SelectedRooms[r.ID] {
			targetIDs = append(targetIDs, r.ID)
			targetNames = append(targetNames, r.Name)
		}
	}

	if sess.Mode != "" && hasModeCap {
		if err := s.val.SetOperationMode(sess.Mode); err != nil {
			log.Printf("[CleaningWizard] warning: failed to set operation mode %q: %v (continuing)", sess.Mode, err)
		}
	}

	if err := s.val.CleanSegments(targetIDs, iterations); err != nil {
		return sess.Mode, targetNames, iterations, fmt.Errorf("failed to clean segments: %w", err)
	}

	return sess.Mode, targetNames, iterations, nil
}
