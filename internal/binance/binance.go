package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
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

func New(symbol string) (*Client, error) {
	not200 := errors.New("status code is not 200")

	url := "wss://stream.binance.com:9443/ws/" + symbol + "@trade"

	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 101 {
		return nil, not200
	}

	return &Client{Conn: conn}, nil
}

func (c *Client) RequestHandler(binanceCh chan<- Trade, ctx context.Context) error {
	defer fmt.Println("RequestHandler is finishing its work")

	var UnmarshalMsg Trade

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			close(binanceCh)
			return err
		}
		if err := json.Unmarshal(msg, &UnmarshalMsg); err != nil {
			close(binanceCh)
			return err
		}

		select {
		case binanceCh <- UnmarshalMsg:
		case <-ctx.Done():
			close(binanceCh)
			return nil
		}
	}
}
