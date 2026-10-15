package bot

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"ratewatch/internal/alert"
	"ratewatch/internal/market"
)

var (
	errNotRule         = errors.New("сообщение не похоже на правило")
	errInvalidCurrency = errors.New("неизвестная валюта")
	errInvalidOperator = errors.New("неизвестный оператор")
	errInvalidAmount   = errors.New("сумма должна быть числом")
	errNegativeAmount  = errors.New("сумма не может быть меньше 0")
	errBelowZero       = errors.New("цена не бывает меньше 0, правило \"< 0\" никогда не сработает")
)

func parseRule(text string) (alert.Rule, error) {
	fields := strings.Fields(strings.ToLower(text))
	if len(fields) != 3 {
		return alert.Rule{}, errNotRule
	}

	var errs []error

	currency := fields[0]
	if !market.Supported(currency) {
		errs = append(errs, errInvalidCurrency)
	}

	op, ok := alert.ParseOp(fields[1])
	if !ok {
		errs = append(errs, errInvalidOperator)
	}

	amount, err := strconv.ParseFloat(fields[2], 64)
	switch {
	case err != nil || math.IsNaN(amount) || math.IsInf(amount, 0):
		errs = append(errs, errInvalidAmount)
	case amount < 0:
		errs = append(errs, errNegativeAmount)
	case op == alert.Below && amount == 0:
		errs = append(errs, errBelowZero)
	}

	if err := errors.Join(errs...); err != nil {
		return alert.Rule{}, err
	}
	return alert.Rule{Currency: currency, Op: op, Amount: amount}, nil
}
