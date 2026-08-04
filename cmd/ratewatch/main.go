package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"ratewatch/internal/telegram"
	"ratewatch/internal/telegram/bot"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	offset := 0

	token := mustToken()

	telegramUrl := "https://api.telegram.org/bot" + token + "/getUpdates"

	// input to the terminal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

loop:
	for {
		select {
		case <-ctx.Done(): // ctrl + c
			fmt.Println("shutting down")
			break loop
		default:
			var err error
			var updates []telegram.Update

			updates, err = telegram.GetUpdates(telegramUrl, offset)
			if err != nil {
				log.Printf("something went wrong while trying to get updates: %s", err)
			} else {
				offset, err = bot.Logic(updates, token, offset)
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
