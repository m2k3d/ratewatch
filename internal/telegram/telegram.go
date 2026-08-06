package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Response struct {
	Ok          bool     `json:"ok"`
	Result      []Update `json:"result"`
	Description string   `json:"description"`
	ErrorCode   int      `json:"error_code"`
}

type Update struct {
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
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

type SendMessageResponse struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
}

type Bot struct {
	Token  string
	Client http.Client
	Offset int
}

func New(token string, client http.Client, offset int) (*Bot, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("bot token must be specified")
	}
	return &Bot{
		Token:  token,
		Client: client,
		Offset: offset,
	}, nil
}

func (b *Bot) GetUpdates() ([]Update, error) {
	apiURL := "https://api.telegram.org/bot" + b.Token + "/getUpdates"

	q := url.Values{}
	q.Set("offset", strconv.Itoa(b.Offset))
	fullURL := apiURL + "?" + q.Encode()

	resp, err := b.Client.Get(fullURL)
	if err != nil {
		errWithNoToken := sanitize(string(err.Error()), b.Token)
		return nil, fmt.Errorf("%s", errWithNoToken)
	}
	defer resp.Body.Close()

	ans, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response: %w", err)
	}

	var r Response
	err = json.Unmarshal(ans, &r)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal: %w", err)
	}
	if !r.Ok {
		return nil, fmt.Errorf("error from the telegram: %s. Error code: %d", r.Description, r.ErrorCode)
	}

	return r.Result, nil
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	apiURL := "https://api.telegram.org/bot" + b.Token + "/sendMessage"

	q := url.Values{}
	q.Set("chat_id", strconv.FormatInt(chatID, 10))
	q.Set("text", text)

	fullURL := apiURL + "?" + q.Encode()

	resp, err := b.Client.Get(fullURL)
	if err != nil {
		errWithNoToken := sanitize(string(err.Error()), b.Token)
		return fmt.Errorf("%s", errWithNoToken)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var r SendMessageResponse
	err = json.Unmarshal(data, &r)
	if err != nil {
		return fmt.Errorf("can't unmarshal data: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("can't do the request, status code: %d. Description: %s", resp.StatusCode, r.Description)
	}

	return nil
}

func (b *Bot) Logic(result []Update) error {
	for _, u := range result {
		err := b.SendMessage(u.Message.Chat.ID, u.Message.Text)
		if err != nil {
			b.Offset = u.UpdateID + 1
			return fmt.Errorf("can't send message: %w", err)
		}

		b.Offset = u.UpdateID + 1
	}

	return nil
}

func sanitize(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "#####")
}
