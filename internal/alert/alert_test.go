package alert_test

import (
	"context"
	"testing"
	"time"

	"ratewatch/internal/alert"
	"ratewatch/internal/market"
	"ratewatch/internal/storage/memstore"
)

func TestTriggered(t *testing.T) {
	tests := []struct {
		name        string
		op          alert.Op
		prev, price float64
		target      float64
		want        bool
	}{
		{name: "above", op: alert.Above, prev: 90, price: 101, target: 100, want: true},
		{name: "not above", op: alert.Above, prev: 90, price: 100, target: 100, want: false},
		{name: "below", op: alert.Below, prev: 110, price: 99, target: 100, want: true},
		{name: "not below", op: alert.Below, prev: 110, price: 100, target: 100, want: false},
		{name: "equal exact", op: alert.Equal, prev: 100, price: 100, target: 100, want: true},
		{name: "equal jumped up", op: alert.Equal, prev: 99, price: 101, target: 100, want: true},
		{name: "equal jumped down", op: alert.Equal, prev: 101, price: 99, target: 100, want: true},
		{name: "equal missed", op: alert.Equal, prev: 98, price: 99, target: 100, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Triggered(tt.prev, tt.price, tt.target); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeSender struct {
	messages chan string
}

func (s fakeSender) SendMessage(_ context.Context, _ int64, text string) error {
	s.messages <- text
	return nil
}

func TestWatcher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := memstore.New()
	if _, err := store.Add(ctx, alert.Rule{ChatID: 1, Currency: "eth", Op: alert.Below, Amount: 3000}); err != nil {
		t.Fatal(err)
	}

	sender := fakeSender{messages: make(chan string, 10)}
	w := alert.NewWatcher(store, sender)
	quotes := make(chan market.Quote)
	done := make(chan error)
	go func() { done <- w.Run(ctx, quotes) }()

	if _, err := w.AddRule(ctx, alert.Rule{ChatID: 2, Currency: "btc", Op: alert.Above, Amount: 50000}); err != nil {
		t.Fatal(err)
	}

	quotes <- market.Quote{Currency: "btc", Price: 49000}
	quotes <- market.Quote{Currency: "btc", Price: 51000}
	expectMessage(t, sender.messages, "Сработало правило #2: btc > 50000, текущая цена 51000")

	quotes <- market.Quote{Currency: "eth", Price: 2900}
	expectMessage(t, sender.messages, "Сработало правило #1: eth < 3000, текущая цена 2900")

	close(quotes)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	rules, err := store.All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Errorf("rules left in store: %+v", rules)
	}
}

func expectMessage(t *testing.T, messages <-chan string, want string) {
	t.Helper()
	select {
	case got := <-messages:
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("no message, want %q", want)
	}
}
