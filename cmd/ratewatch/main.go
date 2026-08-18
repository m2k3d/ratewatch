package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"ratewatch/internal/binance"
	"ratewatch/internal/bot"
	"ratewatch/internal/telegram"

	"github.com/joho/godotenv"
)

func main() {
	var tgWG sync.WaitGroup
	var binanceWG sync.WaitGroup
	var transferWG sync.WaitGroup

	telegramCh := make(chan string)
	binanceCh := make(chan binance.Trade)

	//	mergedCh := make(chan string)

	client := http.Client{
		Timeout: 15 * time.Second,
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

	tgWG.Go(func() {
	Loop1:
		for {
			select {
			case <-ctx.Done(): // ctrl + c
				fmt.Println("shutting down telegram client")
				close(telegramCh)
				break Loop1
			default:
				var err error
				var updates []telegram.Update

				updates, err = tClient.GetUpdates()
				if err != nil {
					log.Printf("something went wrong while trying to get updates: %s", err)
				} else {
					err = app.HandleUpdates(updates, telegramCh)
					if err != nil {
						log.Printf("something went wrong in logic part: %s", err)
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	})

	c, err := binance.New("btcusdt") // just for test, it should be in telegram
	if err != nil {
		log.Fatal(err)
	}

	binanceWG.Go(
		func() {
			log.Println("connecting to binance")
			err = c.RequestHandler(binanceCh, ctx)
			if err != nil {
				fmt.Println(err)
			}
		})

	var ok bool
	var binanceStat binance.Trade
	transferWG.Go(func() {
	Loop2:
		for {

			select {
			case <-ctx.Done():
				fmt.Println("\nmerge function is stoping")
				break Loop2
			default:
			}

			select {
			case telegramMsg, ok := <-telegramCh:
				if !ok {
					break Loop2
				}
				fmt.Println(telegramMsg, binanceStat)
			default:
			}

			select {
			case binanceStat, ok = <-binanceCh:
				if !ok {
					break Loop2
				}
			default:
			}
		}
	})

	binanceWG.Wait()
	tgWG.Wait()
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
