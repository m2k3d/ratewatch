package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"ratewatch/internal/alert"
	"ratewatch/internal/binance"
	"ratewatch/internal/bot"
	"ratewatch/internal/config"
	"ratewatch/internal/market"
	"ratewatch/internal/storage/postgres"
	"ratewatch/internal/telegram"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tg, err := telegram.New(cfg.TelegramToken, &http.Client{Timeout: telegram.PollTimeout + 10*time.Second})
	if err != nil {
		return err
	}

	store, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	watcher := alert.NewWatcher(store, tg)
	b := bot.New(tg, watcher)

	quotes := make(chan market.Quote)
	var feeds sync.WaitGroup
	for _, currency := range market.Currencies {
		feeds.Go(func() { binance.New(currency).Run(ctx, quotes) })
	}

	var wg sync.WaitGroup
	var watchErr error
	wg.Go(func() { b.Run(ctx) })
	wg.Go(func() {
		watchErr = watcher.Run(ctx, quotes)
		stop()
	})

	feeds.Wait()
	close(quotes)
	wg.Wait()

	log.Println("ratewatch stopped")
	return watchErr
}
