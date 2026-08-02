package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"ratewatch/internal/telegram"
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
		case <-ctx.Done():
			// ctrl + c
			fmt.Println("выход")
			break loop
		default:
			var err error
			offset, err = telegram.GetUpdates(telegramUrl, token, offset)
			if err != nil {
				log.Printf("something went wrong %s", err)
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
