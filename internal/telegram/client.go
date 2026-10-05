package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tgbot/internal/netutil"
)

type Client struct {
	apiURL      string
	httpClient  *http.Client
	pollTimeout int
}

func NewClient(token, apiBase string, pollTimeoutSec int) *Client {
	if apiBase == "" {
		apiBase = "https://api.telegram.org"
	}
	apiURL := fmt.Sprintf("%s/bot%s", strings.TrimRight(apiBase, "/"), token)
	return &Client{
		apiURL:      apiURL,
		httpClient:  netutil.NewHTTPClient(time.Duration(pollTimeoutSec+10) * time.Second),
		pollTimeout: pollTimeoutSec,
	}
}

func (c *Client) GetUpdates(offset int) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=%d", c.apiURL, offset, c.pollTimeout)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data.Result, nil
}

func (c *Client) SendPayload(payload SendMessagePayload) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	resp, err := c.httpClient.Post(c.apiURL+"/sendMessage", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var res MsgResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	if !res.OK {
		return 0, fmt.Errorf("telegram sendMessage error: %s", res.Description)
	}
	return res.Result.MessageID, nil
}

func (c *Client) SendTextMessage(chatID int64, text string, disableNotification bool, markup any) (int, error) {
	payload := SendMessagePayload{
		ChatID:              chatID,
		Text:                text,
		ParseMode:           "HTML",
		DisableNotification: disableNotification,
		ReplyMarkup:         markup,
	}
	return c.SendPayload(payload)
}

func (c *Client) EditMessage(chatID int64, msgID int, text string, markup *InlineKeyboardMarkup) error {
	payload := EditMessagePayload{
		ChatID:      chatID,
		MessageID:   msgID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.apiURL+"/editMessageText", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if !res.OK {
		if strings.Contains(res.Description, "message is not modified") {
			return nil
		}
		return fmt.Errorf("telegram editMessageText error: %s", res.Description)
	}
	return nil
}

func (c *Client) DeleteMessage(chatID int64, msgID int) error {
	if msgID == 0 {
		return nil
	}
	url := fmt.Sprintf("%s/deleteMessage?chat_id=%d&message_id=%d", c.apiURL, chatID, msgID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if !res.OK {
		return fmt.Errorf("telegram deleteMessage error: %s", res.Description)
	}
	return nil
}

func (c *Client) RemoveReplyKeyboard(chatID int64, text string) error {
	payload := SendMessagePayload{
		ChatID:              chatID,
		Text:                text,
		ParseMode:           "HTML",
		ReplyMarkup:         ReplyKeyboardRemove{RemoveKeyboard: true},
		DisableNotification: true,
	}
	msgID, err := c.SendPayload(payload)
	if err == nil && msgID != 0 {
		_ = c.DeleteMessage(chatID, msgID)
	}
	return err
}

func (c *Client) AnswerCallbackQuery(callbackQueryID string) error {
	return c.AnswerCallbackQueryAlert(callbackQueryID, "", false)
}

func (c *Client) AnswerCallbackQueryAlert(callbackQueryID string, text string, showAlert bool) error {
	params := url.Values{}
	params.Set("callback_query_id", callbackQueryID)
	if text != "" {
		params.Set("text", text)
	}
	if showAlert {
		params.Set("show_alert", "true")
	}
	ackURL := fmt.Sprintf("%s/answerCallbackQuery?%s", c.apiURL, params.Encode())
	resp, err := c.httpClient.Get(ackURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *Client) SendPhoto(chatID int64, photoData io.Reader, caption string, disableNotification bool) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	_ = writer.WriteField("caption", caption)
	_ = writer.WriteField("parse_mode", "HTML")
	if disableNotification {
		_ = writer.WriteField("disable_notification", "true")
	}

	part, err := writer.CreateFormFile("photo", "map.png")
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, photoData); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	reqPost, err := http.NewRequest("POST", c.apiURL+"/sendPhoto", body)
	if err != nil {
		return err
	}
	reqPost.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(reqPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if !res.OK {
		return fmt.Errorf("telegram sendPhoto error: %s", res.Description)
	}
	return nil
}
