package bot

import (
	"errors"
	"fmt"
	"ratewatch/internal/telegram"
	"slices"
	"strconv"
	"strings"
)

var ErrMessageNotTemplated = errors.New("non-templated message")

type Bot struct {
	TelegramClient *telegram.Client
}

type Rule struct {
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

func (b *Bot) HandleUpdates(result []telegram.Update) error {
	var errs error

	for _, u := range result {
		switch u.Message.Text {
		case "/help":
			err := b.TelegramClient.SendMessage(u.Message.Chat.ID, "there will be help here")
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("can't send message when user sent /help: %w", err))
				continue
			}
		default:
			r, err := newRule(u.Message.Text)
			if err != nil && errors.Is(err, ErrMessageNotTemplated) {
				err = b.TelegramClient.SendMessage(u.Message.Chat.ID, "didn't understand, send \\help")
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("failed to send unknown command message: %w", err))
					continue
				}
				continue
			} else if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to process non template message: %w", err))
				continue
			}

			switch {
			case r != nil:
				s := strconv.Itoa(int(r.Amount)) + r.Currency + r.Op + "idk"
				err := b.TelegramClient.SendMessage(u.Message.Chat.ID, s)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("can't send message in Rule branch: %w", err))
					continue
				}
			default:
				err := b.TelegramClient.SendMessage(u.Message.Chat.ID, "didn't understand, send \\help")
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("can't send message, \"default\" branch: %w", err))
					continue
				}
			}
		}

	}

	return errs
}

func newRule(m string) (*Rule, error) {
	currencies := []string{"btc", "eth", "ton"}
	operators := []string{"<", "=", ">"}

	wordsArray := strings.Split(m, " ")
	if len(wordsArray) != 3 {
		return nil, ErrMessageNotTemplated
	}

	if c := slices.Contains(currencies, wordsArray[0]); !c {
		return nil, errors.New("invalid currency")
	}
	if c := slices.Contains(operators, wordsArray[1]); !c {
		return nil, errors.New("invalid operator")
	}

	amount, err := strconv.ParseFloat(wordsArray[2], 64)
	if err != nil {
		return nil, errors.New("wrong amount")
	}
	if amount < 0.0 {
		return nil, errors.New("amount can't be less than 0")
	}

	return &Rule{
		Currency: wordsArray[0],
		Op:       wordsArray[1],
		Amount:   amount,
	}, nil
}
