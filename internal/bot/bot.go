package bot

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"ratewatch/internal/telegram"
)

type Bot struct {
	TelegramClient *telegram.Client
}

type Rule struct {
	ChatID   int64
	Currency string
	Op       string // operator
	Amount   float64
}

func New(tc *telegram.Client) (*Bot, error) {
	if tc == nil {
		return nil, errors.New("empty telegram client")
	}
	return &Bot{
		TelegramClient: tc,
	}, nil
}

func (b *Bot) HandleUpdates(result []telegram.Update, ruleCh chan<- Rule) error {
	var errs error

	for _, u := range result {
		switch u.Message.Text {
		case "/help":
			err := b.TelegramClient.SendMessage(u.Message.Chat.ID, "Формат: валюта оператор сумма\nВалюты: btc, eth, ton\nОператоры: <, =, >\nПример: btc > 50000")
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("can't send message when user sent /help: %w", err))
				continue
			}
		default:
			r, err := newRule(u.Message.Text)
			if err != nil && errors.Is(err, ErrMessageNotTemplated) {
				err = b.TelegramClient.SendMessage(u.Message.Chat.ID, "didn't understand, send /help")
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("failed to send message: %w", err))
					continue
				}
				continue
			} else if err != nil {
				err = b.TelegramClient.SendMessage(u.Message.Chat.ID, err.Error())
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("failed to send unknown command message: %w", err))
					continue
				}
				continue
			}

			r.ChatID = u.Message.Chat.ID
			ruleCh <- *r
		}
	}

	return errs
}

var (
	invalidCurrency               = errors.New("invalid currency")
	invalidOperator               = errors.New("invalid operator")
	wrongAmount                   = errors.New("wrong amount")
	lowAmount                     = errors.New("amount can't be less than 0")
	ErrMessageNotTemplated        = errors.New("non-templated message")
	invalidOperatorWithZeroAmount = errors.New("invalid operator \"<\" with 0 amount")
)

func newRule(m string) (*Rule, error) {
	var errs error

	currencies := []string{"btc", "eth", "ton"}
	operators := []string{"<", "=", ">"}

	wordsArray := strings.Fields(m)

	if len(wordsArray) != 3 {
		return nil, ErrMessageNotTemplated
	}

	if c := slices.Contains(currencies, wordsArray[0]); !c {
		errs = errors.Join(errs, invalidCurrency)
	}
	if c := slices.Contains(operators, wordsArray[1]); !c {
		errs = errors.Join(errs, invalidOperator)
	}

	amount, err := strconv.ParseFloat(wordsArray[2], 64)
	if err != nil {
		errs = errors.Join(errs, wrongAmount)
	} else if wordsArray[1] == "<" && amount == 0 {
		errs = errors.Join(errs, invalidOperatorWithZeroAmount)
	}

	if amount < 0.0 {
		errs = errors.Join(errs, lowAmount)
	}

	if errs != nil {
		return nil, errs
	}
	return &Rule{
		Currency: wordsArray[0],
		Op:       wordsArray[1],
		Amount:   amount,
	}, nil
}
