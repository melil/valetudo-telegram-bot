package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ValetudoBaseURL = "http://127.0.0.1/api/v2/robot"
	PollTimeoutSec  = 25
	WatcherInterval = 12 * time.Second
)

var (
	BotToken      = os.Getenv("BOT_TOKEN")
	AllowedChatID int64
	TgAPI         string

	// Настройки DND (Не беспокоить)
	DNDEnabled   = true
	DNDStartHour = 23
	DNDEndHour   = 8

	// Маппинг комнат Dreame
	RoomAliases = map[string]string{
		"1": "Кухня",
		"2": "Диван",
		"3": "Коридор",
		"4": "Воркспейс",
	}

	// Сессии мастера уборки
	wizardMu      sync.Mutex
	activeWizards = make(map[int64]*WizardSession)
)

type WizardSession struct {
	Mode          string
	SelectedRooms map[string]bool
	Iterations    int
	MessageID     int
}

// --- Telegram Models ---

type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID int `json:"message_id"`
	Chat      struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text string `json:"text"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    struct{ ID int64 `json:"id"` } `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

type TgResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type TgMsgResponse struct {
	OK     bool    `json:"ok"`
	Result Message `json:"result"`
}

type ReplyKeyboardMarkup struct {
	Keyboard       [][]string `json:"keyboard"`
	ResizeKeyboard bool       `json:"resize_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessagePayload struct {
	ChatID              int64  `json:"chat_id"`
	Text                string `json:"text"`
	ParseMode           string `json:"parse_mode,omitempty"`
	ReplyMarkup         any    `json:"reply_markup,omitempty"`
	DisableNotification bool   `json:"disable_notification,omitempty"`
}

type EditMessagePayload struct {
	ChatID      int64                 `json:"chat_id"`
	MessageID   int                   `json:"message_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// --- Valetudo Models ---

type CleanSegmentPayload struct {
	Action     string   `json:"action"`
	SegmentIDs []string `json:"segment_ids"`
	Iterations int      `json:"iterations"`
}

type PresetActionPayload struct {
	Name string `json:"name"`
}

type GenericAttribute struct {
	Class string `json:"__class"`
	Type  string `json:"type,omitempty"`
	Value any    `json:"value,omitempty"`
	Level int    `json:"level,omitempty"`
	Flag  string `json:"flag,omitempty"`
}

type ConsumableRemaining struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type ConsumableItem struct {
	Type      string              `json:"type"`
	SubType   string              `json:"subType"`
	Remaining ConsumableRemaining `json:"remaining"`
}

type ValetudoDataPoint struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

// --- Utilities ---

func isDNDActive() bool {
	if !DNDEnabled {
		return false
	}
	h := time.Now().Hour()
	if DNDStartHour > DNDEndHour {
		return h >= DNDStartHour || h < DNDEndHour
	}
	return h >= DNDStartHour && h < DNDEndHour
}

func renderProgressBar(percent int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := percent / 10
	empty := 10 - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func formatModeTitle(mode string) string {
	switch mode {
	case "vacuum":
		return "Только сухая"
	case "mop":
		return "Только влажная"
	case "vacuum_and_mop":
		return "Вместе (сухая + влажная)"
	case "vacuum_then_mop":
		return "Сначала сухая, затем влажная"
	default:
		return mode
	}
}

func main() {
	if BotToken == "" {
		log.Fatal("ОШИБКА: Задайте переменную окружения BOT_TOKEN")
	}

	chatIDStr := os.Getenv("CHAT_ID")
	var err error
	AllowedChatID, err = strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || AllowedChatID == 0 {
		log.Fatal("ОШИБКА: Задайте корректный числовой CHAT_ID в переменных окружения")
	}

	apiBase := os.Getenv("TG_API_BASE")
	if apiBase == "" {
		apiBase = "https://api.telegram.org"
	}
	TgAPI = fmt.Sprintf("%s/bot%s", strings.TrimRight(apiBase, "/"), BotToken)
	log.Printf("Бот запущен. Слушаю сообщения для ChatID: %d\n", AllowedChatID)

	client := &http.Client{Timeout: (PollTimeoutSec + 5) * time.Second}

	go statusWatcher(client)

	offset := 0
	for {
		updates, err := getUpdates(client, offset)
		if err != nil {
			log.Printf("Ошибка poll-запроса: %v. Повтор через 3с...", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.Message != nil && update.Message.Chat.ID == AllowedChatID {
				handleTextCommand(client, update.Message.Text)
			}

			if update.CallbackQuery != nil && update.CallbackQuery.From.ID == AllowedChatID {
				handleCallback(client, update.CallbackQuery)
			}
		}
	}
}

// --- Watcher (Push-уведомления) ---

func statusWatcher(client *http.Client) {
	var lastStatus string
	var lastErrorFlag string
	var lastCleanWater string
	var lastDirtyWater string
	firstRun := true

	for {
		time.Sleep(WatcherInterval)

		resp, err := http.Get(ValetudoBaseURL + "/state/attributes")
		if err != nil {
			continue
		}

		var attrs []GenericAttribute
		if err := json.NewDecoder(resp.Body).Decode(&attrs); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		currentStatus := ""
		currentErrorFlag := "none"
		currentCleanWater := "ok"
		currentDirtyWater := "ok"

		for _, attr := range attrs {
			switch attr.Class {
			case "StatusStateAttribute":
				if val, ok := attr.Value.(string); ok {
					currentStatus = val
				}
				if attr.Flag != "" {
					currentErrorFlag = attr.Flag
				}
			case "DockComponentStateAttribute":
				valStr, _ := attr.Value.(string)
				if attr.Type == "water_tank_clean" {
					currentCleanWater = valStr
				} else if attr.Type == "water_tank_dirty" {
					currentDirtyWater = valStr
				}
			}
		}

		if firstRun {
			lastStatus = currentStatus
			lastErrorFlag = currentErrorFlag
			lastCleanWater = currentCleanWater
			lastDirtyWater = currentDirtyWater
			firstRun = false
			continue
		}

		if currentStatus == "error" && (lastStatus != "error" || currentErrorFlag != lastErrorFlag) {
			sendTextMessage(client, fmt.Sprintf("🚨 <b>Внимание! Ошибка робота:</b>\nСтатус: <code>%s</code> | Код: <code>%s</code>", currentStatus, currentErrorFlag), true)
		}

		if (lastStatus == "cleaning" || lastStatus == "returning") && currentStatus == "docked" {
			lastMin, lastSec, lastArea := getCurrentSessionStats()
			msg := fmt.Sprintf(
				"🏁 <b>Уборка завершена!</b>\n"+
					"Робот успешно вернулся на станцию.\n\n"+
					"⏱ <b>Время:</b> %d мин %d сек\n"+
					"📐 <b>Площадь:</b> %.1f м²",
				lastMin, lastSec, lastArea,
			)
			sendCleaningMap(client, msg)
		}

		if currentCleanWater != "ok" && lastCleanWater == "ok" {
			sendTextMessage(client, "🚰 <b>Док-станция: закончилась чистая вода!</b>\nПожалуйста, заполните бак.", true)
		}
		if currentDirtyWater != "ok" && lastDirtyWater == "ok" {
			sendTextMessage(client, "☣️ <b>Док-станция: бак грязной воды полон!</b>\nПожалуйста, слейте сточную воду.", true)
		}

		lastStatus = currentStatus
		lastErrorFlag = currentErrorFlag
		lastCleanWater = currentCleanWater
		lastDirtyWater = currentDirtyWater
	}
}

// --- Меню Робота, Станции и Настроек ---

func sendRobotMenu(client *http.Client, msgID int) {
	text := "🤖 <b>Управление роботом</b>\nВыберите действие:"
	rows := [][]InlineKeyboardButton{
		{{Text: "🚀 Старт (Вся уборка)", CallbackData: "cmd_start"}},
		{{Text: "⏸ Пауза", CallbackData: "cmd_pause"}, {Text: "🏠 Домой", CallbackData: "cmd_home"}},
		{{Text: "🏎 Телеметрия", CallbackData: "cmd_telemetry"}, {Text: "🧹 Расходники", CallbackData: "cmd_consumables"}},
		{{Text: "⚙️ Настройки", CallbackData: "menu_settings"}},
	}
	markup := &InlineKeyboardMarkup{InlineKeyboard: rows}

	if msgID == 0 {
		payload := SendMessagePayload{
			ChatID:              AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: isDNDActive(),
		}
		body, _ := json.Marshal(payload)
		client.Post(TgAPI+"/sendMessage", "application/json", bytes.NewReader(body))
	} else {
		editMessage(client, msgID, text, markup)
	}
}

func sendStationMenu(client *http.Client, msgID int) {
	text := "🏠 <b>Управление док-станцией</b>\nВыберите действие:"
	rows := [][]InlineKeyboardButton{
		{{Text: "🏠 Вернуть робота на базу", CallbackData: "cmd_station_home"}},
		{{Text: "💨 Вытряхнуть пыль", CallbackData: "dock_empty"}},
		{{Text: "🧼 Постирать швабры", CallbackData: "dock_wash"}},
		{{Text: "♨️ Старт сушки", CallbackData: "dock_dry_start"}, {Text: "❄️ Стоп сушки", CallbackData: "dock_dry_stop"}},
	}
	markup := &InlineKeyboardMarkup{InlineKeyboard: rows}

	if msgID == 0 {
		payload := SendMessagePayload{
			ChatID:              AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: isDNDActive(),
		}
		body, _ := json.Marshal(payload)
		client.Post(TgAPI+"/sendMessage", "application/json", bytes.NewReader(body))
	} else {
		editMessage(client, msgID, text, markup)
	}
}

func getSettingsMainMenu() (string, *InlineKeyboardMarkup) {
	text := "⚙️ <b>Настройки параметров по умолчанию</b>\nЭти параметры применяются при запуске <b>«🚀 Вся уборка»</b>. Выберите категорию:"
	rows := [][]InlineKeyboardButton{
		{{Text: "🛠 Тип уборки", CallbackData: "sub_mode"}},
		{{Text: "💨 Мощность всасывания", CallbackData: "sub_fan"}},
		{{Text: "💧 Влажность швабр", CallbackData: "sub_water"}},
		{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}},
	}
	return text, &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func triggerValetudoAction(capability string, action string) {
	url := ValetudoBaseURL + "/capabilities/" + capability
	payload := fmt.Sprintf(`{"action":"%s"}`, action)
	req, _ := http.NewRequest("PUT", url, bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)
}

func setValetudoPreset(capability string, value string) {
	url := ValetudoBaseURL + "/capabilities/" + capability + "/preset"
	payload := fmt.Sprintf(`{"name":"%s"}`, value)
	req, _ := http.NewRequest("PUT", url, bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)
}

// --- Дополнительные функции (Map, Locate, Consumables) ---

func sendCleaningMap(client *http.Client, caption string) {
	req, _ := http.NewRequest("GET", ValetudoBaseURL+"/state/map", nil)
	req.Header.Set("Accept", "image/png")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		sendTextMessage(client, caption, true)
		if err == nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("chat_id", strconv.FormatInt(AllowedChatID, 10))
	_ = writer.WriteField("caption", caption)
	_ = writer.WriteField("parse_mode", "HTML")
	if isDNDActive() {
		_ = writer.WriteField("disable_notification", "true")
	}

	part, _ := writer.CreateFormFile("photo", "map.png")
	io.Copy(part, resp.Body)
	writer.Close()

	reqPost, _ := http.NewRequest("POST", TgAPI+"/sendPhoto", body)
	reqPost.Header.Set("Content-Type", writer.FormDataContentType())
	client.Do(reqPost)
}

func triggerLocate() error {
	payload := map[string]string{"action": "locate"}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPut, ValetudoBaseURL+"/capabilities/LocateCapability", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func sendConsumablesMenu(client *http.Client, msgID int) {
	resp, err := http.Get(ValetudoBaseURL + "/capabilities/ConsumableMonitoringCapability")
	if err != nil {
		if msgID == 0 {
			sendTextMessage(client, "❌ Ошибка связи с Valetudo API", true)
		} else {
			editMessage(client, msgID, "❌ Ошибка связи с Valetudo API", nil)
		}
		return
	}
	defer resp.Body.Close()

	var items []ConsumableItem
	_ = json.NewDecoder(resp.Body).Decode(&items)

	text := "🧹 <b>Состояние расходников:</b>\n\n"
	var rows [][]InlineKeyboardButton

	for _, item := range items {
		valMin := item.Remaining.Value / 60
		name := item.SubType
		var maxMin int

		switch {
		case item.Type == "brush" && item.SubType == "main":
			name = "Турбощетка"
			maxMin = 240 * 60
		case item.Type == "brush" && item.SubType == "side_right":
			name = "Боковая щетка"
			maxMin = 150 * 60
		case item.Type == "filter" && item.SubType == "main":
			name = "HEPA-фильтр"
			maxMin = 90 * 60
		case item.Type == "cleaning" && item.SubType == "sensor":
			name = "Сенсоры"
			maxMin = 30 * 60
		default:
			name = item.Type + "_" + item.SubType
			maxMin = 150 * 60
		}

		pct := (valMin * 100) / maxMin
		if pct > 100 {
			pct = 100
		}

		bar := renderProgressBar(pct)
		text += fmt.Sprintf("• <b>%s</b>: %d ч\n<code>[%s] %d%%</code>\n\n", name, valMin/60, bar, pct)

		rows = append(rows, []InlineKeyboardButton{{
			Text:         fmt.Sprintf("🔄 Сбросить: %s", name),
			CallbackData: fmt.Sprintf("reset_cons:%s:%s", item.Type, item.SubType),
		}})
	}

	rows = append(rows, []InlineKeyboardButton{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}})
	markup := &InlineKeyboardMarkup{InlineKeyboard: rows}

	if msgID == 0 {
		payload := SendMessagePayload{
			ChatID:              AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: isDNDActive(),
		}
		body, _ := json.Marshal(payload)
		client.Post(TgAPI+"/sendMessage", "application/json", bytes.NewReader(body))
	} else {
		editMessage(client, msgID, text, markup)
	}
}

// --- Статистика ---

func getCurrentSessionStats() (min int, sec int, areaM2 float64) {
	resp, err := http.Get(ValetudoBaseURL + "/capabilities/CurrentStatisticsCapability")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var points []ValetudoDataPoint
	if err := json.NewDecoder(resp.Body).Decode(&points); err == nil {
		for _, p := range points {
			switch p.Type {
			case "time":
				min = p.Value / 60
				sec = p.Value % 60
			case "area":
				areaM2 = float64(p.Value) / 10000.0
			}
		}
	}
	return
}

func getTotalStats() (totalHours int, totalCount int, totalAreaM2 float64) {
	resp, err := http.Get(ValetudoBaseURL + "/capabilities/TotalStatisticsCapability")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var points []ValetudoDataPoint
	if err := json.NewDecoder(resp.Body).Decode(&points); err == nil {
		for _, p := range points {
			switch p.Type {
			case "time":
				totalHours = p.Value / 3600
			case "count":
				totalCount = p.Value
			case "area":
				totalAreaM2 = float64(p.Value) / 10000.0
			}
		}
	}
	return
}

// --- API Робота ---

func setOperationMode(mode string) error {
	url := ValetudoBaseURL + "/capabilities/OperationModeControlCapability/preset"
	payload := PresetActionPayload{Name: mode}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func cleanSegments(segmentIDs []string, iterations int) error {
	url := ValetudoBaseURL + "/capabilities/MapSegmentationCapability"
	payload := CleanSegmentPayload{
		Action:     "start_segment_action",
		SegmentIDs: segmentIDs,
		Iterations: iterations,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func triggerAction(action string) error {
	url := ValetudoBaseURL + "/capabilities/BasicControlCapability"
	payload := map[string]string{"action": action}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// --- Обработка Telegram ---

func getUpdates(client *http.Client, offset int) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=%d", TgAPI, offset, PollTimeoutSec)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data TgResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data.Result, nil
}

func sendTextMessage(client *http.Client, text string, showMainMenu bool) int {
	payload := SendMessagePayload{
		ChatID:              AllowedChatID,
		Text:                text,
		ParseMode:           "HTML",
		DisableNotification: isDNDActive(),
	}

	if showMainMenu {
		payload.ReplyMarkup = &ReplyKeyboardMarkup{
			Keyboard: [][]string{
				{"🪄 Старт уборки", "🛑 Закончить уборку"},
				{"🤖 Робот", "🏠 Станция"},
				{"📢 Найти робота"},
			},
			ResizeKeyboard: true,
		}
	}

	body, _ := json.Marshal(payload)
	resp, err := client.Post(TgAPI+"/sendMessage", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var res TgMsgResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return res.Result.MessageID
}

func editMessage(client *http.Client, msgID int, text string, markup *InlineKeyboardMarkup) {
	payload := EditMessagePayload{
		ChatID:      AllowedChatID,
		MessageID:   msgID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}

	body, _ := json.Marshal(payload)
	_, _ = client.Post(TgAPI+"/editMessageText", "application/json", bytes.NewReader(body))
}

func handleTextCommand(client *http.Client, text string) {
	switch strings.TrimSpace(text) {
	case "/start", "Меню":
		sendTextMessage(client, "🕹 <b>Управление Dreame X30 Pro готово:</b>", true)

	case "/wizard", "🪄 Старт уборки":
		startCleaningWizard(client)

	case "/stop", "🛑 Закончить уборку":
		if err := triggerAction("home"); err != nil {
			sendTextMessage(client, "❌ Ошибка: "+err.Error(), true)
		} else {
			sendTextMessage(client, "🛑 Уборка прервана. Робот возвращается на базу.", true)
		}

	case "/robot", "🤖 Робот":
		sendRobotMenu(client, 0)

	case "/station", "🏠 Станция":
		sendStationMenu(client, 0)

	case "/locate", "📢 Найти робота":
		_ = triggerLocate()
		sendTextMessage(client, "🔊 Подаю звуковой сигнал!", true)

	// Оставлено для обратной совместимости
	case "/settings", "⚙️ Настройки":
		text, markup := getSettingsMainMenu()
		payload := SendMessagePayload{
			ChatID:              AllowedChatID,
			Text:                text,
			ParseMode:           "HTML",
			ReplyMarkup:         markup,
			DisableNotification: isDNDActive(),
		}
		body, _ := json.Marshal(payload)
		client.Post(TgAPI+"/sendMessage", "application/json", bytes.NewReader(body))

	case "/start_clean", "🚀 Вся уборка":
		if err := triggerAction("start"); err != nil {
			sendTextMessage(client, "❌ Ошибка старта: "+err.Error(), true)
		} else {
			sendTextMessage(client, "🚀 Запустил генеральную уборку с настройками по умолчанию.", true)
		}

	case "/pause", "⏸ Пауза":
		if err := triggerAction("pause"); err != nil {
			sendTextMessage(client, "❌ Ошибка: "+err.Error(), true)
		} else {
			sendTextMessage(client, "⏸ Робот на паузе.", true)
		}

	case "/home", "🏠 На базу":
		if err := triggerAction("home"); err != nil {
			sendTextMessage(client, "❌ Ошибка: "+err.Error(), true)
		} else {
			sendTextMessage(client, "🏠 Еду на док-станцию.", true)
		}

	case "/telemetry", "🏎 Телеметрия":
		sendTextMessage(client, buildTelemetryReport(), true)

	case "/consumables", "🧹 Расходники":
		sendConsumablesMenu(client, 0)

	default:
		sendTextMessage(client, "Команда не распознана. Воспользуйтесь меню.", true)
	}
}

// --- Мастер Уборки (Wizard) ---

func startCleaningWizard(client *http.Client) {
	wizardMu.Lock()
	activeWizards[AllowedChatID] = &WizardSession{
		SelectedRooms: make(map[string]bool),
		Iterations:    1,
	}
	wizardMu.Unlock()

	text, markup := renderWizardStep1()
	payload := SendMessagePayload{
		ChatID:              AllowedChatID,
		Text:                text,
		ParseMode:           "HTML",
		ReplyMarkup:         markup,
		DisableNotification: isDNDActive(),
	}

	body, _ := json.Marshal(payload)
	resp, err := client.Post(TgAPI+"/sendMessage", "application/json", bytes.NewReader(body))
	if err == nil {
		defer resp.Body.Close()
		var res TgMsgResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
			wizardMu.Lock()
			if ws, ok := activeWizards[AllowedChatID]; ok {
				ws.MessageID = res.Result.MessageID
			}
			wizardMu.Unlock()
		}
	}
}

func renderWizardStep1() (string, *InlineKeyboardMarkup) {
	text := "🪄 <b>Шаг 1 из 3: Выберите тип уборки</b>\nКак будем убирать выбранные зоны?"
	modes := []struct {
		ID   string
		Name string
	}{
		{"vacuum_and_mop", "🌪 Сухая + Влажная (одновременно)"},
		{"vacuum_then_mop", "🔄 Сначала сухая, затем влажная"},
		{"vacuum", "💨 Только сухая (пылесос)"},
		{"mop", "💧 Только влажная (швабры)"},
	}

	var rows [][]InlineKeyboardButton
	for _, m := range modes {
		btn := InlineKeyboardButton{
			Text:         m.Name,
			CallbackData: "wiz_mode:" + m.ID,
		}
		rows = append(rows, []InlineKeyboardButton{btn})
	}
	rows = append(rows, []InlineKeyboardButton{{Text: "❌ Отмена", CallbackData: "wiz_cancel"}})

	return text, &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func renderWizardStep2(ws *WizardSession) (string, *InlineKeyboardMarkup) {
	text := fmt.Sprintf("🪄 <b>Шаг 2 из 3: Выберите комнаты</b>\nРежим: <code>%s</code>\n\n<i>Отметьте одну или несколько комнат и нажмите «Далее»:</i>", formatModeTitle(ws.Mode))

	order := []string{"1", "2", "3", "4"}
	var rows [][]InlineKeyboardButton

	hasSelected := false
	for _, id := range order {
		name := RoomAliases[id]
		icon := "◻️"
		if ws.SelectedRooms[id] {
			icon = "✅"
			hasSelected = true
		}
		btn := InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", icon, name),
			CallbackData: "wiz_toggle_room:" + id,
		}
		rows = append(rows, []InlineKeyboardButton{btn})
	}

	var controlRow []InlineKeyboardButton
	if hasSelected {
		controlRow = append(controlRow, InlineKeyboardButton{Text: "Далее ➡️", CallbackData: "wiz_to_step3"})
	}
	controlRow = append(controlRow, InlineKeyboardButton{Text: "❌ Отмена", CallbackData: "wiz_cancel"})
	rows = append(rows, controlRow)

	return text, &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func renderWizardStep3(ws *WizardSession) (string, *InlineKeyboardMarkup) {
	var roomNames []string
	for id, ok := range ws.SelectedRooms {
		if ok {
			roomNames = append(roomNames, RoomAliases[id])
		}
	}

	text := fmt.Sprintf(
		"🪄 <b>Шаг 3 из 3: Количество проходов</b>\n\n"+
			"• <b>Режим:</b> <code>%s</code>\n"+
			"• <b>Зоны:</b> %s\n\n"+
			"Сколько раз повторить уборку выбранных зон?",
		formatModeTitle(ws.Mode), strings.Join(roomNames, ", "),
	)

	rows := [][]InlineKeyboardButton{
		{
			{Text: "1️⃣ Один проход (1x)", CallbackData: "wiz_iter:1"},
			{Text: "2️⃣ Двойной проход (2x)", CallbackData: "wiz_iter:2"},
		},
		{
			{Text: "⬅️ Назад к комнатам", CallbackData: "wiz_back_to_step2"},
			{Text: "❌ Отмена", CallbackData: "wiz_cancel"},
		},
	}

	return text, &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func handleCallback(client *http.Client, cb *CallbackQuery) {
	ackURL := fmt.Sprintf("%s/answerCallbackQuery?callback_query_id=%s", TgAPI, cb.ID)
	_, _ = client.Get(ackURL)

	data := cb.Data

	if data == "noop" {
		return
	}

	// --- Действия меню Робот ---
	switch data {
	case "menu_robot":
		sendRobotMenu(client, cb.Message.MessageID)
		return
	case "cmd_start":
		triggerAction("start")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
		editMessage(client, cb.Message.MessageID, "🚀 <b>Запущена генеральная уборка.</b>", markup)
		return
	case "cmd_pause":
		triggerAction("pause")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
		editMessage(client, cb.Message.MessageID, "⏸ <b>Робот на паузе.</b>", markup)
		return
	case "cmd_home":
		triggerAction("home")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
		editMessage(client, cb.Message.MessageID, "🏠 <b>Робот возвращается на базу.</b>", markup)
		return
	case "cmd_telemetry":
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к роботу", CallbackData: "menu_robot"}}}}
		editMessage(client, cb.Message.MessageID, buildTelemetryReport(), markup)
		return
	case "cmd_consumables":
		sendConsumablesMenu(client, cb.Message.MessageID)
		return
	}

	// --- Управление Станцией ---
	switch data {
	case "menu_station":
		sendStationMenu(client, cb.Message.MessageID)
		return
	case "cmd_station_home":
		triggerAction("home")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		editMessage(client, cb.Message.MessageID, "🏠 <b>Робот возвращается на базу.</b>", markup)
		return
	case "dock_empty":
		triggerValetudoAction("AutoEmptyDockManualTriggerCapability", "trigger")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		editMessage(client, cb.Message.MessageID, "💨 <b>Запущена выгрузка пыли в док-станцию.</b>", markup)
		return
	case "dock_wash":
		triggerValetudoAction("MopWashingManualTriggerCapability", "start")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		editMessage(client, cb.Message.MessageID, "🧼 <b>Запущена стирка швабр.</b>", markup)
		return
	case "dock_dry_start":
		triggerValetudoAction("MopDryingManualTriggerCapability", "start")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		editMessage(client, cb.Message.MessageID, "♨️ <b>Запущена сушка швабр.</b>", markup)
		return
	case "dock_dry_stop":
		triggerValetudoAction("MopDryingManualTriggerCapability", "stop")
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "⬅️ Назад к станции", CallbackData: "menu_station"}}}}
		editMessage(client, cb.Message.MessageID, "❄️ <b>Сушка швабр остановлена.</b>", markup)
		return
	}

	// --- Навигация по подменю настроек ---
	switch data {
	case "menu_settings":
		text, markup := getSettingsMainMenu()
		editMessage(client, cb.Message.MessageID, text, markup)
		return

	case "sub_mode":
		text := "🛠 <b>Выберите тип уборки по умолчанию:</b>"
		rows := [][]InlineKeyboardButton{
			{{Text: "Только сухая", CallbackData: "set_mode:vacuum"}, {Text: "Только влажная", CallbackData: "set_mode:mop"}},
			{{Text: "Сухая + Влажная", CallbackData: "set_mode:vacuum_and_mop"}},
			{{Text: "Сначала сухая ➡️ затем влажная", CallbackData: "set_mode:vacuum_then_mop"}},
			{{Text: "⬅️ Назад", CallbackData: "menu_settings"}},
		}
		editMessage(client, cb.Message.MessageID, text, &InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_fan":
		text := "💨 <b>Выберите мощность всасывания по умолчанию:</b>"
		rows := [][]InlineKeyboardButton{
			{{Text: "Тихо", CallbackData: "set_fan:min"}, {Text: "Стандарт", CallbackData: "set_fan:low"}},
			{{Text: "Турбо", CallbackData: "set_fan:high"}, {Text: "Макс", CallbackData: "set_fan:max"}},
			{{Text: "⬅️ Назад", CallbackData: "menu_settings"}},
		}
		editMessage(client, cb.Message.MessageID, text, &InlineKeyboardMarkup{InlineKeyboard: rows})
		return

	case "sub_water":
		text := "💧 <b>Выберите влажность швабр по умолчанию:</b>"
		rows := [][]InlineKeyboardButton{
			{{Text: "Мин", CallbackData: "set_water:min"}, {Text: "Средне", CallbackData: "set_water:medium"}, {Text: "Макс", CallbackData: "set_water:max"}},
			{{Text: "⬅️ Назад", CallbackData: "menu_settings"}},
		}
		editMessage(client, cb.Message.MessageID, text, &InlineKeyboardMarkup{InlineKeyboard: rows})
		return
	}

	// --- Установка самих настроек ---
	if strings.HasPrefix(data, "set_mode:") {
		val := strings.TrimPrefix(data, "set_mode:")
		setValetudoPreset("OperationModeControlCapability", val)
		text, markup := getSettingsMainMenu()
		editMessage(client, cb.Message.MessageID, "✅ Установлено: <b>"+formatModeTitle(val)+"</b>\n\n"+text, markup)
		return
	}
	if strings.HasPrefix(data, "set_fan:") {
		val := strings.TrimPrefix(data, "set_fan:")
		setValetudoPreset("FanSpeedControlCapability", val)
		text, markup := getSettingsMainMenu()
		editMessage(client, cb.Message.MessageID, "✅ Мощность установлена на: <b>"+val+"</b>\n\n"+text, markup)
		return
	}
	if strings.HasPrefix(data, "set_water:") {
		val := strings.TrimPrefix(data, "set_water:")
		setValetudoPreset("WaterUsageControlCapability", val)
		text, markup := getSettingsMainMenu()
		editMessage(client, cb.Message.MessageID, "✅ Влажность установлена на: <b>"+val+"</b>\n\n"+text, markup)
		return
	}

	// Сброс расходников
	if strings.HasPrefix(data, "reset_cons:") {
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			cType, cSubType := parts[1], parts[2]
			payload := fmt.Sprintf(`{"action":"reset", "type":"%s", "subType":"%s"}`, cType, cSubType)
			req, _ := http.NewRequest(http.MethodPut, ValetudoBaseURL+"/capabilities/ConsumableMonitoringCapability", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			client.Do(req)

			// Даем Valetudo 300мс на обновление атрибутов перед отрисовкой меню
			time.Sleep(300 * time.Millisecond)
			sendConsumablesMenu(client, cb.Message.MessageID)
		}
		return
	}

	// --- Обработка Визарда ---
	wizardMu.Lock()
	ws, exists := activeWizards[AllowedChatID]
	wizardMu.Unlock()

	if strings.HasPrefix(data, "wiz_") && (!exists || ws == nil) {
		editMessage(client, cb.Message.MessageID, "⚠️ Сессия настройки устарела. Нажмите кнопку <b>🪄 Старт уборки</b> заново.", nil)
		return
	}

	switch {
	case data == "wiz_cancel":
		wizardMu.Lock()
		delete(activeWizards, AllowedChatID)
		wizardMu.Unlock()
		editMessage(client, cb.Message.MessageID, "❌ Настройка уборки отменена.", nil)

	case strings.HasPrefix(data, "wiz_mode:"):
		mode := strings.TrimPrefix(data, "wiz_mode:")
		ws.Mode = mode
		text, markup := renderWizardStep2(ws)
		editMessage(client, cb.Message.MessageID, text, markup)

	case strings.HasPrefix(data, "wiz_toggle_room:"):
		roomID := strings.TrimPrefix(data, "wiz_toggle_room:")
		ws.SelectedRooms[roomID] = !ws.SelectedRooms[roomID]
		text, markup := renderWizardStep2(ws)
		editMessage(client, cb.Message.MessageID, text, markup)

	case data == "wiz_to_step3":
		text, markup := renderWizardStep3(ws)
		editMessage(client, cb.Message.MessageID, text, markup)

	case data == "wiz_back_to_step2":
		text, markup := renderWizardStep2(ws)
		editMessage(client, cb.Message.MessageID, text, markup)

	case strings.HasPrefix(data, "wiz_iter:"):
		iterStr := strings.TrimPrefix(data, "wiz_iter:")
		iterations, _ := strconv.Atoi(iterStr)
		if iterations <= 0 {
			iterations = 1
		}
		ws.Iterations = iterations

		var targetIDs []string
		var targetNames []string
		for id, active := range ws.SelectedRooms {
			if active {
				targetIDs = append(targetIDs, id)
				targetNames = append(targetNames, RoomAliases[id])
			}
		}

		wizardMu.Lock()
		delete(activeWizards, AllowedChatID)
		wizardMu.Unlock()

		if err := setOperationMode(ws.Mode); err != nil {
			editMessage(client, cb.Message.MessageID, "❌ Ошибка установки режима: "+err.Error(), nil)
			return
		}

		if err := cleanSegments(targetIDs, ws.Iterations); err != nil {
			editMessage(client, cb.Message.MessageID, "❌ Ошибка старта сегментов: "+err.Error(), nil)
			return
		}

		successMsg := fmt.Sprintf(
			"🚀 <b>Уборка запущена!</b>\n\n"+
				"• <b>Режим:</b> <code>%s</code>\n"+
				"• <b>Комнаты:</b> %s\n"+
				"• <b>Проходов:</b> %d",
			formatModeTitle(ws.Mode), strings.Join(targetNames, ", "), ws.Iterations,
		)
		markup := &InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{{{Text: "🤖 Открыть меню робота", CallbackData: "menu_robot"}}}}
		editMessage(client, cb.Message.MessageID, successMsg, markup)
	}
}

// --- Телеметрия ---

func formatDockSensor(val string) string {
	if val == "ok" {
		return "🟢 OK"
	}
	return "🔴 " + strings.ToUpper(val)
}

func buildTelemetryReport() string {
	batLevel := "?"
	robotStatus := "на базе"
	dockAction := ""
	currentMode := "vacuum_and_mop"

	cleanWater := "ok"
	dirtyWater := "ok"
	detergent := "ok"
	dustbag := "ok"

	if resp, err := http.Get(ValetudoBaseURL + "/state/attributes"); err == nil {
		defer resp.Body.Close()
		var attrs []GenericAttribute
		if err := json.NewDecoder(resp.Body).Decode(&attrs); err == nil {
			for _, attr := range attrs {
				switch attr.Class {
				case "BatteryStateAttribute":
					batLevel = strconv.Itoa(attr.Level)
				case "StatusStateAttribute":
					if val, ok := attr.Value.(string); ok {
						robotStatus = val
					}
				case "DockStatusStateAttribute":
					if val, ok := attr.Value.(string); ok && val != "idle" && val != "none" {
						dockAction = val
					}
				case "PresetSelectionStateAttribute":
					if attr.Type == "operation_mode" {
						if val, ok := attr.Value.(string); ok {
							currentMode = val
						}
					}
				case "DockComponentStateAttribute":
					valStr, _ := attr.Value.(string)
					switch attr.Type {
					case "water_tank_clean":
						cleanWater = valStr
					case "water_tank_dirty":
						dirtyWater = valStr
					case "detergent":
						detergent = valStr
					case "dustbag":
						dustbag = valStr
					}
				}
			}
		}
	}

	mainH, sideH, filterH, sensorH := "?", "?", "?", "?"
	if resp, err := http.Get(ValetudoBaseURL + "/capabilities/ConsumableMonitoringCapability"); err == nil {
		defer resp.Body.Close()
		var list []ConsumableItem
		if err := json.NewDecoder(resp.Body).Decode(&list); err == nil {
			for _, item := range list {
				hours := strconv.Itoa(item.Remaining.Value / 60)
				switch {
				case item.Type == "brush" && item.SubType == "main":
					mainH = hours
				case item.Type == "brush" && item.SubType == "side_right":
					sideH = hours
				case item.Type == "filter" && item.SubType == "main":
					filterH = hours
				case item.Type == "cleaning" && item.SubType == "sensor":
					sensorH = hours
				}
			}
		}
	}

	uptimeStr := "неизвестно"
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if sec, err := strconv.ParseFloat(fields[0], 64); err == nil {
				d := time.Duration(sec) * time.Second
				uptimeStr = fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
			}
		}
	}

	statusExtra := ""
	if dockAction != "" {
		if dockAction == "drying" {
			statusExtra = " (сушка швабр)"
		} else {
			statusExtra = fmt.Sprintf(" (%s)", dockAction)
		}
	}

	lastMin, lastSec, lastArea := getCurrentSessionStats()
	totHours, totCount, totArea := getTotalStats()

	return fmt.Sprintf(
		"🏎 <b>БОРТОВОЙ ЖУРНАЛ DREAME X30 PRO</b>\n\n"+
			"🔋 <b>Силовая установка:</b>\n"+
			"• Заряд АКБ: <b>%s%%</b>\n"+
			"• Статус: <b>%s%s</b>\n"+
			"• Режим уборки: <b>%s</b>\n"+
			"• Аптайм Linux: <b>%s</b>\n\n"+
			"📈 <b>Сессии и налет:</b>\n"+
			"• Крайняя сессия: <b>%d мин %d с</b> | <b>%.1f м²</b>\n"+
			"• Всего выездов: <b>%d</b>\n"+
			"• Суммарный налет: <b>%d ч</b> (%.0f м²)\n\n"+
			"💧 <b>Резервуары станции:</b>\n"+
			"• Чистая вода: <b>%s</b>\n"+
			"• Грязная вода: <b>%s</b>\n"+
			"• Моющее средство: <b>%s</b>\n"+
			"• Пылесборник: <b>%s</b>\n\n"+
			"⚙️ <b>Остаточный ресурс узлов:</b>\n"+
			"• Основная щетка: <b>%s ч</b> (~240 ч макс)\n"+
			"• Боковая щетка: <b>%s ч</b> (~150 ч макс)\n"+
			"• HEPA-фильтр: <b>%s ч</b> (~90 ч макс)\n"+
			"• Очистка датчиков: <b>%s ч</b> (~30 ч макс)\n\n"+
			"📊 <b>Статус:</b> Все системы в норме",
		batLevel, robotStatus, statusExtra, formatModeTitle(currentMode), uptimeStr,
		lastMin, lastSec, lastArea, totCount, totHours, totArea,
		formatDockSensor(cleanWater), formatDockSensor(dirtyWater), formatDockSensor(detergent), formatDockSensor(dustbag),
		mainH, sideH, filterH, sensorH,
	)
}