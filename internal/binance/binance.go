package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

const (
	streamURL = "wss://stream.binance.com:9443/ws/"

	// Binance pings every 20s and drops the connection if it gets no pong
	// within a minute, so a minute of silence means the connection is dead.
	readTimeout  = time.Minute
	writeTimeout = 10 * time.Second

	minBackoff = time.Second
	maxBackoff = time.Minute
)

type Client struct {
	symbol string
}

type Trade struct {
	EventType    string `json:"e"`
	EventTime    int64  `json:"E"`
	Symbol       string `json:"s"`
	TradeID      int64  `json:"t"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	TradeTime    int64  `json:"T"`
	IsBuyerMaker bool   `json:"m"`
}

func New(symbol string) *Client {
	return &Client{symbol: symbol}
}

// Run streams trades into binanceCh until ctx is cancelled, reconnecting
// with exponential backoff whenever the connection fails. It closes
// binanceCh on return.
func (c *Client) Run(ctx context.Context, binanceCh chan<- Trade) {
	defer close(binanceCh)

	backoff := minBackoff
	for {
		started := time.Now()
		err := c.stream(ctx, binanceCh)
		if ctx.Err() != nil {
			return
		}

		// The connection lived long enough to count as healthy,
		// so start the backoff over instead of growing it.
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

// stream runs a single websocket connection until it fails or ctx is cancelled.
func (c *Client) stream(ctx context.Context, binanceCh chan<- Trade) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, streamURL+c.symbol+"@trade", nil)
	if err != nil {
		return fmt.Errorf("can't connect: %w", err)
	}
	defer conn.Close()

	// Closing the connection is the only way to unblock ReadMessage.
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

	for {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var t Trade
		if err := json.Unmarshal(msg, &t); err != nil {
			return fmt.Errorf("can't unmarshal trade: %w", err)
		}

		select {
		case binanceCh <- t:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
