package bot

import (
	"context"
	"errors"
	"testing"

	"ratewatch/internal/alert"
)

func TestParseRule(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    alert.Rule
		wantErr []error
	}{
		{name: "valid", text: "btc = 1", want: alert.Rule{Currency: "btc", Op: alert.Equal, Amount: 1}},
		{name: "extra spaces", text: "btc   =  1", want: alert.Rule{Currency: "btc", Op: alert.Equal, Amount: 1}},
		{name: "upper case", text: "BTC > 50000", want: alert.Rule{Currency: "btc", Op: alert.Above, Amount: 50000}},
		{name: "fractional", text: "btc < 1.5", want: alert.Rule{Currency: "btc", Op: alert.Below, Amount: 1.5}},
		{name: "zero", text: "btc = 0", want: alert.Rule{Currency: "btc", Op: alert.Equal}},
		{name: "negative amount", text: "btc = -1", wantErr: []error{errNegativeAmount}},
		{name: "too many words", text: "btc btc = 1", wantErr: []error{errNotRule}},
		{name: "too many words with negative amount", text: "btc btc = -1", wantErr: []error{errNotRule}},
		{name: "invalid currency", text: "doge < 1", wantErr: []error{errInvalidCurrency}},
		{name: "invalid operator", text: "btc >= 1", wantErr: []error{errInvalidOperator}},
		{name: "invalid amount", text: "btc < abc", wantErr: []error{errInvalidAmount}},
		{name: "nan", text: "btc > nan", wantErr: []error{errInvalidAmount}},
		{name: "below zero", text: "btc < 0", wantErr: []error{errBelowZero}},
		{name: "everything wrong", text: "a b c", wantErr: []error{errInvalidCurrency, errInvalidOperator, errInvalidAmount}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRule(tt.text)
			if len(tt.wantErr) == 0 && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, want := range tt.wantErr {
				if !errors.Is(err, want) {
					t.Errorf("error = %v, want %v", err, want)
				}
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

type fakeRules struct {
	added []alert.Rule
}

func (f *fakeRules) AddRule(_ context.Context, r alert.Rule) (alert.Rule, error) {
	r.ID = int64(len(f.added) + 1)
	f.added = append(f.added, r)
	return r, nil
}

func TestReply(t *testing.T) {
	rules := &fakeRules{}
	b := New(nil, rules)

	tests := []struct {
		text string
		want string
	}{
		{text: "/start", want: startText},
		{text: "/help", want: helpText},
		{text: "hello", want: unknownText},
		{text: "doge > 1", want: errInvalidCurrency.Error()},
		{text: "btc > 50000", want: "Правило #1 сохранено: btc > 50000"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := b.reply(context.Background(), 42, tt.text); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	if len(rules.added) != 1 || rules.added[0].ChatID != 42 {
		t.Errorf("added rules = %+v, want one rule for chat 42", rules.added)
	}
}
