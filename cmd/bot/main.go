package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"telegram-bot/internal/bot"
)

func main() {
	cfg, err := bot.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	app, err := bot.NewApp(cfg)
	if err != nil {
		if errors.Is(err, bot.ErrAlreadyRunning) {
			log.Fatalf("create app: another world_cup_poll_bot instance is already running: %v", err)
		}
		log.Fatalf("create app: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("release app lock: %v", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
