package alert

import "strconv"

type Op string

const (
	Above Op = ">"
	Below Op = "<"
	Equal Op = "="
)

func ParseOp(s string) (Op, bool) {
	switch op := Op(s); op {
	case Above, Below, Equal:
		return op, true
	}
	return "", false
}

func (o Op) Triggered(prev, price, target float64) bool {
	switch o {
	case Above:
		return price > target
	case Below:
		return price < target
	case Equal:
		return min(prev, price) <= target && target <= max(prev, price)
	default:
		return false
	}
}

type Rule struct {
	ID       int64
	ChatID   int64
	Currency string
	Op       Op
	Amount   float64
}

func (r Rule) String() string {
	return r.Currency + " " + string(r.Op) + " " + strconv.FormatFloat(r.Amount, 'f', -1, 64)
}
