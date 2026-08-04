package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var client = http.Client{
	Timeout: 5 * time.Second,
}

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

func GetUpdates(telegramUrl string, offset int) ([]Update, error) {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	fullURL := telegramUrl + "?" + q.Encode()

	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, errors.New("can't do GET request")
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

func SendMessage(chatID int64, text string, t string) error {
	apiURL := "https://api.telegram.org/bot" + t + "/sendMessage"

	q := url.Values{}
	q.Set("chat_id", strconv.FormatInt(chatID, 10))
	q.Set("text", text)

	fullURL := apiURL + "?" + q.Encode()

	resp, err := client.Get(fullURL)
	if err != nil {
		return errors.New("can't do GET request")
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
