package main

import (
	"context"
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
		log.Fatalf("create app: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
