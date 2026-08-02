package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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

var token = os.Getenv("BOT_TOKEN")

func main() {
	tUrl := "https://api.telegram.org/bot" + token + "/getUpdates"

	resp, err := http.Get(tUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	ans, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var r Response
	err = json.Unmarshal(ans, &r)
	if err != nil {
		log.Fatal(err)
	}

	chatIDText := make(map[int64]string)
	for _, u := range r.Result {
		chatIDText[u.Message.Chat.ID] = u.Message.Text
	}

	for u := range chatIDText {
		err = sendMessage(u, chatIDText[u])
		if err != nil {
			log.Fatal(err)
		}
	}
}

func sendMessage(chatID int64, text string) error {
	apiURL := "https://api.telegram.org/bot" + token + "/sendMessage"

	q := url.Values{}
	q.Set("chat_id", strconv.Itoa(int(chatID)))
	q.Set("text", text)

	fullURL := apiURL + "?" + q.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
