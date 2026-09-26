package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tgbot/internal/bot"
	"tgbot/internal/config"
	"tgbot/internal/database"
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	userDB, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Ошибка инициализации базы данных SQLite: %v", err)
	}
	defer userDB.Close()

	// Умный бутстрап (Migration/Init):
	// Если таблица пользователей пустая, берём CHAT_ID из ENV и записываем с ролью 'admin'
	if err := userDB.BootstrapAdmin(cfg.AllowedChatID, "admin"); err != nil {
		log.Printf("Предупреждение: ошибка начального бутстрапа администратора: %v", err)
	}

	tgClient := telegram.NewClient(cfg.BotToken, cfg.TgAPIBase, cfg.PollTimeoutSec)
	valetudoClient := valetudo.NewClient(cfg.ValetudoBaseURL, 10*time.Second)

	b := bot.New(cfg, tgClient, valetudoClient, userDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown по системным сигналам
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Получен сигнал %s, завершение работы...", sig)
		cancel()
	}()

	if err := b.Run(ctx); err != nil {
		log.Fatalf("Ошибка работы бота: %v", err)
	}
}
