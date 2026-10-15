package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"ratewatch/internal/alert"
	"ratewatch/internal/market"
	"ratewatch/internal/telegram"
)

const retryDelay = 3 * time.Second

const (
	startText   = "Привет! Я пришлю сообщение, когда курс криптовалюты достигнет нужного значения.\nОтправь /help, чтобы узнать, как добавить правило."
	unknownText = "Не понял, отправь /help"
	failedText  = "Не удалось сохранить правило, попробуй позже"
)

var helpText = "Формат: валюта оператор сумма\nВалюты: " + strings.Join(market.Currencies, ", ") + "\nОператоры: <, =, >\nПример: btc > 50000"

type Client interface {
	GetUpdates(ctx context.Context) ([]telegram.Update, error)
	SendMessage(ctx context.Context, chatID int64, text string) error
}

type RuleAdder interface {
	AddRule(ctx context.Context, r alert.Rule) (alert.Rule, error)
}

type Bot struct {
	client Client
	rules  RuleAdder
}

func New(client Client, rules RuleAdder) *Bot {
	return &Bot{client: client, rules: rules}
}

func (b *Bot) Run(ctx context.Context) {
	for {
		updates, err := b.client.GetUpdates(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("can't get updates: %s", err)
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return
			}
			continue
		}

		for _, u := range updates {
			if u.Message == nil {
				continue
			}
			chatID := u.Message.Chat.ID
			if err := b.client.SendMessage(ctx, chatID, b.reply(ctx, chatID, u.Message.Text)); err != nil {
				log.Printf("can't reply to chat %d: %s", chatID, err)
			}
		}
	}
}

func (b *Bot) reply(ctx context.Context, chatID int64, text string) string {
	switch strings.TrimSpace(text) {
	case "/start":
		return startText
	case "/help":
		return helpText
	}

	r, err := parseRule(text)
	if errors.Is(err, errNotRule) {
		return unknownText
	}
	if err != nil {
		return err.Error()
	}

	r.ChatID = chatID
	r, err = b.rules.AddRule(ctx, r)
	if err != nil {
		log.Printf("can't add rule for chat %d: %s", chatID, err)
		return failedText
	}

	return fmt.Sprintf("Правило #%d сохранено: %s", r.ID, r)
}
