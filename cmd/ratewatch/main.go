package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"ratewatch/internal/binance"
	"ratewatch/internal/bot"
	"ratewatch/internal/telegram"

	"github.com/joho/godotenv"
)

// priceEqualEpsilon is how close price must get to a rule's amount
// for the "=" operator to count as a match.
const priceEqualEpsilon = 0.01

func ruleTriggered(op string, price, amount float64) bool {
	switch op {
	case ">":
		return price > amount
	case "<":
		return price < amount
	case "=":
		return math.Abs(price-amount) <= priceEqualEpsilon
	default:
		return false
	}
}

// checkRules evaluates every rule registered for currency against trade's
// price, notifies the owning chat for each match, and drops matched rules
// so they fire only once.
func checkRules(tClient *telegram.Client, rules map[string][]bot.Rule, currency string, trade binance.Trade) {
	price, err := strconv.ParseFloat(trade.Price, 64)
	if err != nil {
		log.Printf("can't parse price %q for %s: %s", trade.Price, currency, err)
		return
	}

	remaining := rules[currency][:0]
	for _, r := range rules[currency] {
		if !ruleTriggered(r.Op, price, r.Amount) {
			remaining = append(remaining, r)
			continue
		}
		msg := fmt.Sprintf("%s %s %s: current price is %s", currency, r.Op, strconv.FormatFloat(r.Amount, 'f', -1, 64), trade.Price)
		if err := tClient.SendMessage(r.ChatID, msg); err != nil {
			log.Printf("can't notify chat %d: %s", r.ChatID, err)
		}
	}
	rules[currency] = remaining
}

func main() {
	var tgWG sync.WaitGroup
	var binanceWG sync.WaitGroup
	var transferWG sync.WaitGroup

	ruleCh := make(chan bot.Rule)
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
				close(ruleCh)
				break Loop1
			default:
				var err error
				var updates []telegram.Update

				updates, err = tClient.GetUpdates()
				if err != nil {
					log.Printf("something went wrong while trying to get updates: %s", err)
				} else {
					err = app.HandleUpdates(updates, ruleCh)
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
	rules := make(map[string][]bot.Rule, 3)
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
			case <-ctx.Done():
				fmt.Println("\nmerge function is stopping")
				break Loop2
			case r, ok := <-ruleCh:
				if !ok {
					break Loop2
				}
				rules[r.Currency] = append(rules[r.Currency], r)
			case trade, ok := <-btcusdtCh:
				if !ok {
					btcusdtCh = nil
					continue
				}
				binanceStat[trade.Symbol] = trade
				checkRules(tClient, rules, "btc", trade)
			case trade, ok := <-ethusdtCh:
				if !ok {
					ethusdtCh = nil
					continue
				}
				binanceStat[trade.Symbol] = trade
				checkRules(tClient, rules, "eth", trade)
			case trade, ok := <-tonusdtCh:
				if !ok {
					tonusdtCh = nil
					continue
				}
				binanceStat[trade.Symbol] = trade
				checkRules(tClient, rules, "ton", trade)
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
