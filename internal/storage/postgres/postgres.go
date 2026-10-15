package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"ratewatch/internal/alert"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const schema = `CREATE TABLE IF NOT EXISTS rules (
	id       BIGSERIAL PRIMARY KEY,
	chat_id  BIGINT NOT NULL,
	currency TEXT NOT NULL,
	op       TEXT NOT NULL,
	amount   DOUBLE PRECISION NOT NULL
)`

var _ alert.Store = (*Store)(nil)

type Store struct {
	db *sql.DB
}

func New(ctx context.Context, url string) (*Store, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("can't connect to database: %w", err)
	}

	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("can't create schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Add(ctx context.Context, r alert.Rule) (alert.Rule, error) {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO rules (chat_id, currency, op, amount) VALUES ($1, $2, $3, $4) RETURNING id`,
		r.ChatID, r.Currency, string(r.Op), r.Amount,
	).Scan(&r.ID)
	if err != nil {
		return alert.Rule{}, fmt.Errorf("can't insert rule: %w", err)
	}
	return r, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM rules WHERE id = $1`, id); err != nil {
		return fmt.Errorf("can't delete rule %d: %w", id, err)
	}
	return nil
}

func (s *Store) All(ctx context.Context) ([]alert.Rule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, chat_id, currency, op, amount FROM rules ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("can't select rules: %w", err)
	}
	defer rows.Close()

	var rules []alert.Rule
	for rows.Next() {
		var r alert.Rule
		var op string
		if err := rows.Scan(&r.ID, &r.ChatID, &r.Currency, &op, &r.Amount); err != nil {
			return nil, fmt.Errorf("can't scan rule: %w", err)
		}
		r.Op = alert.Op(op)
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("can't read rules: %w", err)
	}
	return rules, nil
}
