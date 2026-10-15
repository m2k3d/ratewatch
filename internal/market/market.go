package market

import "slices"

var Currencies = []string{"btc", "eth", "ton"}

func Supported(currency string) bool {
	return slices.Contains(Currencies, currency)
}

type Quote struct {
	Currency string
	Price    float64
}
