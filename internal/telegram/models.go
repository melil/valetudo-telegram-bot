package telegram

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
	ID      string `json:"id"`
	From    struct {
		ID int64 `json:"id"`
	} `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

type Response struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type MsgResponse struct {
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
