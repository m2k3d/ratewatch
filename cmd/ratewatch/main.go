package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"ratewatch/internal/bot"
	"ratewatch/internal/telegram"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	var client = http.Client{
		Timeout: 5 * time.Second,
	}
	offset := 0

	token := mustToken()

	// input to the terminal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	tClient, err := telegram.New(token, client, offset)
	if err != nil {
		log.Fatal(err)
	}

	app, err := bot.New(tClient)
	if err != nil {
		log.Fatal(err)
	}

Loop:
	for {
		select {
		case <-ctx.Done(): // ctrl + c
			fmt.Println("shutting down")
			break Loop
		default:
			var err error
			var updates []telegram.Update

			updates, err = tClient.GetUpdates()
			if err != nil {
				log.Printf("something went wrong while trying to get updates: %s", err)
			} else {
				err = app.HandleUpdates(updates)
				if err != nil {
					log.Printf("something went wrong in logic part: %s", err)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}

}

func mustToken() string {
	if err := godotenv.Load(); err != nil {
		log.Println("can't find \".env\" file")
	}

	t := os.Getenv("TELEGRAM_TOKEN")
	if t == "" {
		log.Fatal("there isn't TELEGRAM_TOKEN")
	}

	return t
}
