package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"ratewatch/internal/market"

	"github.com/gorilla/websocket"
)

const (
	streamURL = "wss://stream.binance.com:9443/ws/"

	readTimeout  = time.Minute
	writeTimeout = 10 * time.Second

	minBackoff = time.Second
	maxBackoff = time.Minute
)

type Client struct {
	currency string
	symbol   string
}

type trade struct {
	Price string `json:"p"`
}

func New(currency string) *Client {
	return &Client{currency: currency, symbol: currency + "usdt"}
}

func (c *Client) Run(ctx context.Context, quotes chan<- market.Quote) {
	backoff := minBackoff
	for {
		started := time.Now()
		err := c.stream(ctx, quotes)
		if ctx.Err() != nil {
			return
		}

		if time.Since(started) > maxBackoff {
			backoff = minBackoff
		}
		log.Printf("binance %s: %s, reconnecting in %s", c.symbol, err, backoff)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func (c *Client) stream(ctx context.Context, quotes chan<- market.Quote) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, streamURL+c.symbol+"@trade", nil)
	if err != nil {
		return fmt.Errorf("can't connect: %w", err)
	}
	defer conn.Close()

	log.Printf("binance %s: handshake was successful", c.symbol)

	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	conn.SetPingHandler(func(data string) error {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		err := conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(writeTimeout))
		var netErr net.Error
		if errors.Is(err, websocket.ErrCloseSent) || errors.As(err, &netErr) && netErr.Timeout() {
			return nil
		}
		return err
	})

	first := true

	for {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var t trade
		if err := json.Unmarshal(msg, &t); err != nil {
			log.Printf("binance %s: can't unmarshal trade: %s", c.symbol, err)
			continue
		}

		price, err := strconv.ParseFloat(t.Price, 64)
		if err != nil {
			log.Printf("binance %s: can't parse price %q: %s", c.symbol, t.Price, err)
			continue
		}

		if first {
			log.Printf("binance %s: receiving trades, price %s", c.symbol, t.Price)
			first = false
		}

		select {
		case quotes <- market.Quote{Currency: c.currency, Price: price}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
