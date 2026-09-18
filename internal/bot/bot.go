package bot

import (
	"context"
	"log"
	"sync"
	"time"

	"tgbot/internal/config"
	"tgbot/internal/i18n"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

type Bot struct {
	cfg *config.Config
	tg  *telegram.Client
	val *valetudo.Client

	langMu sync.RWMutex
	lang   i18n.Locale

	wizardMu      sync.Mutex
	activeWizards map[int64]*WizardSession
}

func New(cfg *config.Config, tg *telegram.Client, val *valetudo.Client) *Bot {
	return &Bot{
		cfg:           cfg,
		tg:            tg,
		val:           val,
		lang:          i18n.NormalizeLocale(cfg.DefaultLang),
		activeWizards: make(map[int64]*WizardSession),
	}
}

func (b *Bot) GetLang() i18n.Locale {
	b.langMu.RLock()
	defer b.langMu.RUnlock()
	return b.lang
}

func (b *Bot) SetLang(loc i18n.Locale) {
	b.langMu.Lock()
	defer b.langMu.Unlock()
	b.lang = loc
}

func (b *Bot) t(key string, args ...any) string {
	return i18n.T(b.GetLang(), key, args...)
}

func (b *Bot) Run(ctx context.Context) error {
	log.Printf("Бот запущен. Слушаю сообщения для ChatID: %d\n", b.cfg.AllowedChatID)

	// Запуск фонового мониторинга
	go b.statusWatcher(ctx)

	offset := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("Остановка цикла long polling...")
			return nil
		default:
		}

		updates, err := b.tg.GetUpdates(offset)
		if err != nil {
			log.Printf("Ошибка poll-запроса: %v. Повтор через 3с...", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.Message != nil && update.Message.Chat.ID == b.cfg.AllowedChatID {
				b.handleTextCommand(update.Message.Text)
			}

			if update.CallbackQuery != nil && update.CallbackQuery.From.ID == b.cfg.AllowedChatID {
				b.handleCallback(update.CallbackQuery)
			}
		}
	}
}
