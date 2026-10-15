package memstore

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"sync"

	"ratewatch/internal/alert"
)

type Store struct {
	mu     sync.Mutex
	nextID int64
	rules  map[int64]alert.Rule
}

func New() *Store {
	return &Store{rules: make(map[int64]alert.Rule)}
}

func (s *Store) Add(_ context.Context, r alert.Rule) (alert.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	r.ID = s.nextID
	s.rules[r.ID] = r
	return r, nil
}

func (s *Store) Delete(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.rules, id)
	return nil
}

func (s *Store) All(_ context.Context) ([]alert.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules := slices.Collect(maps.Values(s.rules))
	slices.SortFunc(rules, func(a, b alert.Rule) int { return cmp.Compare(a.ID, b.ID) })
	return rules, nil
}
