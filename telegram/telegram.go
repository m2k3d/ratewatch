package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Response struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
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

func SendMessage(chatID int64, text string, t string) error {
	apiURL := "https://api.telegram.org/bot" + t + "/sendMessage"

	q := url.Values{}
	q.Set("chat_id", strconv.Itoa(int(chatID)))
	q.Set("text", text)

	fullURL := apiURL + "?" + q.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("can't do the request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func GetUpdates(telegramUrl, token string, offset int) (int, error) {
	apiURL := "https://api.telegram.org/bot" + token + "/getupdates"
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	fullURL := apiURL + "?" + q.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return offset, fmt.Errorf("can't do the request: %w", err)
	}
	defer resp.Body.Close()

	ans, err := io.ReadAll(resp.Body)
	if err != nil {
		return offset, fmt.Errorf("can't read response: %w", err)
	}

	var r Response
	err = json.Unmarshal(ans, &r)
	if err != nil {
		return offset, fmt.Errorf("can't unmarshal: %w", err)
	}

	for _, u := range r.Result {
		err = SendMessage(u.Message.Chat.ID, u.Message.Text, token)
		if err != nil {
			return offset, fmt.Errorf("can't send message: %w", err)
		}
		offset = u.UpdateID
	}

	if len(r.Result) > 0 {
		return offset + 1, nil
	} else {
		return offset, nil
	}
}
