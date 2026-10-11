package main

import (
	"context"
	"fmt"
	"log"
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

// ruleTriggered reports whether price matches the rule. For "=" it is enough
// that amount lies between the previous and the current price, since trades
// often jump over the exact target.
func ruleTriggered(op string, prev, price, amount float64) bool {
	switch op {
	case ">":
		return price > amount
	case "<":
		return price < amount
	case "=":
		return min(prev, price) <= amount && amount <= max(prev, price)
	default:
		return false
	}
}

func checkRules(tClient *telegram.Client, notifyWG *sync.WaitGroup, rules map[string][]bot.Rule, prevPrices map[string]float64, currency string, trade binance.Trade) {
	price, err := strconv.ParseFloat(trade.Price, 64)
	if err != nil {
		log.Printf("can't parse price %q for %s: %s", trade.Price, currency, err)
		return
	}

	prev, ok := prevPrices[currency]
	if !ok {
		prev = price
	}
	prevPrices[currency] = price

	remaining := rules[currency][:0]
	for _, r := range rules[currency] {
		if !ruleTriggered(r.Op, prev, price, r.Amount) {
			remaining = append(remaining, r)
			continue
		}
		msg := fmt.Sprintf("%s %s %s: current price is %s", currency, r.Op, strconv.FormatFloat(r.Amount, 'f', -1, 64), trade.Price)
		notifyWG.Go(func() {
			if err := tClient.SendMessage(r.ChatID, msg); err != nil {
				log.Printf("can't notify chat %d: %s", r.ChatID, err)
			}
		})
	}
	rules[currency] = remaining
}

func main() {
	var tgWG sync.WaitGroup
	var binanceWG sync.WaitGroup
	var transferWG sync.WaitGroup
	var notifyWG sync.WaitGroup

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
					err = app.HandleUpdates(ctx, updates, ruleCh)
					if err != nil {
						log.Printf("something went wrong in logic part: %s", err)
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	})

	log.Println("connecting to binance")
	binanceWG.Go(func() { binance.New("btcusdt").Run(ctx, btcusdtCh) })
	binanceWG.Go(func() { binance.New("ethusdt").Run(ctx, ethusdtCh) })
	binanceWG.Go(func() { binance.New("tonusdt").Run(ctx, tonusdtCh) })

	binanceStat := make(map[string]binance.Trade, 3)
	rules := make(map[string][]bot.Rule, 3)
	prevPrices := make(map[string]float64, 3)
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
				checkRules(tClient, &notifyWG, rules, prevPrices, "btc", trade)
			case trade, ok := <-ethusdtCh:
				if !ok {
					ethusdtCh = nil
					continue
				}
				binanceStat[trade.Symbol] = trade
				checkRules(tClient, &notifyWG, rules, prevPrices, "eth", trade)
			case trade, ok := <-tonusdtCh:
				if !ok {
					tonusdtCh = nil
					continue
				}
				binanceStat[trade.Symbol] = trade
				checkRules(tClient, &notifyWG, rules, prevPrices, "ton", trade)
			}
		}
	})

	binanceWG.Wait()
	tgWG.Wait()
	transferWG.Wait()
	notifyWG.Wait()
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
