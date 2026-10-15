package alert

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	"ratewatch/internal/market"
)

type Store interface {
	Add(ctx context.Context, r Rule) (Rule, error)
	Delete(ctx context.Context, id int64) error
	All(ctx context.Context) ([]Rule, error)
}

type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

type firedRule struct {
	rule  Rule
	price float64
}

type Watcher struct {
	store  Store
	sender Sender
	added  chan Rule
	fired  chan firedRule
	rules  map[string]map[int64]Rule
	prices map[string]float64
}

func NewWatcher(store Store, sender Sender) *Watcher {
	return &Watcher{
		store:  store,
		sender: sender,
		added:  make(chan Rule),
		fired:  make(chan firedRule, 100),
		rules:  make(map[string]map[int64]Rule),
		prices: make(map[string]float64),
	}
}

func (w *Watcher) AddRule(ctx context.Context, r Rule) (Rule, error) {
	r, err := w.store.Add(ctx, r)
	if err != nil {
		return Rule{}, fmt.Errorf("can't save rule: %w", err)
	}

	select {
	case w.added <- r:
		return r, nil
	case <-ctx.Done():
		return Rule{}, ctx.Err()
	}
}

func (w *Watcher) Run(ctx context.Context, quotes <-chan market.Quote) error {
	rules, err := w.store.All(ctx)
	if err != nil {
		return fmt.Errorf("can't load rules: %w", err)
	}
	for _, r := range rules {
		w.add(r)
	}

	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()
	wg.Go(func() { w.notify(ctx) })

	for {
		select {
		case <-ctx.Done():
			return nil
		case r := <-w.added:
			w.add(r)
		case q, ok := <-quotes:
			if !ok {
				return nil
			}
			w.check(ctx, q)
		}
	}
}

func (w *Watcher) add(r Rule) {
	if w.rules[r.Currency] == nil {
		w.rules[r.Currency] = make(map[int64]Rule)
	}
	w.rules[r.Currency][r.ID] = r
}

func (w *Watcher) check(ctx context.Context, q market.Quote) {
	prev, ok := w.prices[q.Currency]
	if !ok {
		prev = q.Price
	}
	w.prices[q.Currency] = q.Price

	for id, r := range w.rules[q.Currency] {
		if !r.Op.Triggered(prev, q.Price, r.Amount) {
			continue
		}
		delete(w.rules[q.Currency], id)

		select {
		case w.fired <- firedRule{rule: r, price: q.Price}:
		case <-ctx.Done():
			return
		}
	}
}

func (w *Watcher) notify(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-w.fired:
			if err := w.store.Delete(ctx, f.rule.ID); err != nil {
				log.Printf("can't delete rule %d: %s", f.rule.ID, err)
				continue
			}

			text := fmt.Sprintf("Сработало правило #%d: %s, текущая цена %s", f.rule.ID, f.rule, strconv.FormatFloat(f.price, 'f', -1, 64))
			if err := w.sender.SendMessage(ctx, f.rule.ChatID, text); err != nil {
				log.Printf("can't notify chat %d: %s", f.rule.ChatID, err)
			}
		}
	}
}
