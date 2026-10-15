package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	apiURL      = "https://api.telegram.org"
	PollTimeout = 30 * time.Second
)

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type response struct {
	Ok          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
}

type Client struct {
	token  string
	http   *http.Client
	url    string
	offset int
}

func New(token string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("bot token must be specified")
	}
	return &Client{token: token, http: httpClient, url: apiURL}, nil
}

func (c *Client) GetUpdates(ctx context.Context) ([]Update, error) {
	params := map[string]int{
		"offset":  c.offset,
		"timeout": int(PollTimeout.Seconds()),
	}

	var updates []Update
	if err := c.call(ctx, "getUpdates", params, &updates); err != nil {
		return nil, err
	}

	if len(updates) > 0 {
		c.offset = updates[len(updates)-1].UpdateID + 1
	}
	return updates, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	params := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	return c.call(ctx, "sendMessage", params, nil)
}

func (c *Client) call(ctx context.Context, method string, params, result any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("%s: can't marshal params: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/bot"+c.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: %s", method, c.sanitize(err))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s: %s", method, c.sanitize(err))
	}
	defer resp.Body.Close()

	var r response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("%s: can't decode response, status %d: %w", method, resp.StatusCode, err)
	}
	if !r.Ok {
		return fmt.Errorf("%s: error from telegram %d: %s", method, r.ErrorCode, r.Description)
	}

	if result == nil {
		return nil
	}
	if err := json.Unmarshal(r.Result, result); err != nil {
		return fmt.Errorf("%s: can't unmarshal result: %w", method, err)
	}
	return nil
}

func (c *Client) sanitize(err error) string {
	return strings.ReplaceAll(err.Error(), c.token, "#####")
}
