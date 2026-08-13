package bot

import (
	"errors"
	"testing"
)

func TestNewRule(t *testing.T) {

	// 1 - correct input
	s := "btc = 1"
	_, err := newRule(s)
	if err != nil {
		t.Errorf("test 1. Shouldn't be any errors")
		return
	}

	// 2 - several "space bars" between words
	s = "btc   =  1"
	v, err := newRule(s)
	if err != nil {
		t.Error("test 2. Several spaces bars doesn't precessed correctly")
		return
	} else {
		if v.Currency != "btc" || v.Op != "=" || v.Amount != 1.000000 {
			t.Errorf("test 2. Several space bars doesn't precessed correctly. Currency - want: btc, got: %s, operator - want: =, got: %s, amount- want: 1.000000, got: %f", v.Currency, v.Op, v.Amount)
			return
		}
	}

	// 3 - incorrect amount
	s = "btc = -1"
	v, err = newRule(s)
	if !errors.Is(err, lowAmount) {
		t.Errorf("test 3. Incorrect error - want \"%s\", got: \"%s\"", lowAmount, err)
		return
	}

	// 4 - generally wrong message
	s = "btc btc = 1"
	v, err = newRule(s)
	if !errors.Is(err, ErrMessageNotTemplated) {
		t.Errorf("test 4. Incorrect error - want \"%s\", got: \"%s\"", ErrMessageNotTemplated, err)
		return
	}

	// 5 - generally wrong message with wrong amount
	s = "btc btc = -1"
	v, err = newRule(s)
	if !errors.Is(err, ErrMessageNotTemplated) {
		t.Errorf("test 5. Incorrect error - want \"%s\", got: \"%s\"", ErrMessageNotTemplated, err)
		return
	}

	// 6 - invalid currency
	s = "invalidCurrency < 1"
	v, err = newRule(s)
	if !errors.Is(err, invalidCurrency) {
		t.Errorf("test 6. Incorrect error - want \"%s\", got: \"%s\"", invalidCurrency, err)
		return
	}

	// 7 - invalid operator
	s = "btc invalidOperator 1"
	v, err = newRule(s)
	if !errors.Is(err, invalidOperator) {
		t.Errorf("test 7. Incorrect error - want \"%s\", got: \"%s\"", invalidOperator, err)
		return
	}

	// 8 - wrong amount
	s = "btc < wrongAmount"
	v, err = newRule(s)
	if !errors.Is(err, wrongAmount) {
		t.Errorf("test 8. Incorrect error - want \"%s\", got: \"%s\"", wrongAmount, err)
		return
	}

	// 9 - every word is wrong
	s = "wrongWord wrongWord wrongWord"
	v, err = newRule(s)
	if !errors.Is(err, invalidCurrency) || !errors.Is(err, invalidOperator) || !errors.Is(err, wrongAmount) {
		t.Errorf("test 9. Incorrect error - got: \"%s\"", err)
		return
	}

	// 10 - zero (0) shold works correctly
	s = "btc = 0"
	v, err = newRule(s)
	if err != nil {
		t.Errorf("test 10. Shouldn't have any errors, got: \"%s\"", err)
		return
	}

	// 11 - shouldn't use < and 0
	s = "btc < 0"
	v, err = newRule(s)
	if !errors.Is(err, invalidOperatorWithZeroAmount) {
		t.Errorf("test 11. Incorrect error - want \"%s\", got: \"%s\"", invalidOperatorWithZeroAmount, err)
		return
	}

	// 12 - fractional number
	s = "btc < 1.5"
	v, err = newRule(s)
	if err != nil {
		t.Errorf("test 12. Shouldn't have any errors, got: \"%s\"", err)
		return
	}
}
