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
	"tgbot/internal/telegram"
	"tgbot/internal/valetudo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	tgClient := telegram.NewClient(cfg.BotToken, cfg.TgAPIBase, cfg.PollTimeoutSec)
	valetudoClient := valetudo.NewClient(cfg.ValetudoBaseURL, 10*time.Second)

	b := bot.New(cfg, tgClient, valetudoClient)

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
