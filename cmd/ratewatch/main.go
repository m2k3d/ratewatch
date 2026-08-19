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
	btcusdtCh := make(chan binance.Trade)
	ethusdtCh := make(chan binance.Trade)
	tonusdtCh := make(chan binance.Trade)

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

	btcusdt, err := binance.New("btcusdt")
	if err != nil {
		log.Fatal(err)
	}
	ethusdt, err := binance.New("ethusdt")
	if err != nil {
		log.Fatal(err)
	}
	tonusdt, err := binance.New("tonusdt")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("connecting to binance")
	binanceWG.Go(
		func() {
			log.Println("connecting to websocket (btcusdt)")
			err = btcusdt.RequestHandler(btcusdtCh, ctx)
			if err != nil {
				fmt.Println(err)
			}
			btcusdt.Conn.Close()
		})
	binanceWG.Go(
		func() {
			log.Println("connecting to websocket (ethusdt)")
			err = ethusdt.RequestHandler(ethusdtCh, ctx)
			if err != nil {
				fmt.Println(err)
			}
			ethusdt.Conn.Close()
		})

	binanceWG.Go(
		func() {
			log.Println("connecting to websocket (tonusdt)")
			err = tonusdt.RequestHandler(tonusdtCh, ctx)
			if err != nil {
				fmt.Println(err)
			}
			tonusdt.Conn.Close()
		})

	binanceStat := make(map[string]binance.Trade, 3)
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
			case trade, ok := <-btcusdtCh:
				if !ok {
					btcusdtCh = nil
					break
				}
				binanceStat[trade.Symbol] = trade
			default:
			}

			select {
			case trade, ok := <-ethusdtCh:
				if !ok {
					ethusdtCh = nil
					break
				}
				binanceStat[trade.Symbol] = trade
			default:
			}

			select {
			case trade, ok := <-tonusdtCh:
				if !ok {
					tonusdtCh = nil
					break
				}
				binanceStat[trade.Symbol] = trade
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
