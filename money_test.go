// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"math/big"
	"testing"
)

func usd() *Currency { return MustCurrency("USD") }
func jpy() *Currency { return MustCurrency("JPY") }
func eur() *Currency { return MustCurrency("EUR") }

// TestNewAndAccessors covers the constructor, fractional/cents/currency/amount
// accessors and the decimal amount for both hundredth and zero-decimal
// currencies.
func TestNewAndAccessors(t *testing.T) {
	m := New(1234, usd())
	if m.Fractional() != 1234 || m.Cents() != 1234 {
		t.Errorf("fractional=%d", m.Fractional())
	}
	if m.Currency().ID != "usd" {
		t.Errorf("currency=%s", m.Currency())
	}
	if got := m.Amount().FloatString(2); got != "12.34" {
		t.Errorf("amount=%s", got)
	}
	if got := New(1000, jpy()).Amount().FloatString(1); got != "1000.0" {
		t.Errorf("jpy amount=%s", got)
	}
	if m.Bank() == nil {
		t.Error("bank nil")
	}
	if m.Symbol() != "$" {
		t.Errorf("symbol=%s", m.Symbol())
	}
	if got := New(10, MustCurrency("XBA")).Symbol(); got != "¤" {
		t.Errorf("no-symbol currency = %q, want ¤", got)
	}
}

func TestNewWithBankDefaults(t *testing.T) {
	m := NewWithBank(5, usd(), nil)
	if m.Bank() == nil {
		t.Error("nil bank not defaulted")
	}
}

func TestNewNilCurrencyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic on nil currency")
		}
	}()
	New(1, nil)
}

func TestNewWithBankNilCurrencyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	NewWithBank(1, nil, nil)
}

func TestFromAmount(t *testing.T) {
	// 23.45 USD -> 2345 cents.
	if got := FromAmount(big.NewRat(2345, 100), usd()).Fractional(); got != 2345 {
		t.Errorf("from_amount usd = %d", got)
	}
	// 23.45 JPY -> 23 (rounded half-up).
	if got := FromAmount(big.NewRat(2345, 100), jpy()).Fractional(); got != 23 {
		t.Errorf("from_amount jpy = %d", got)
	}
}

func TestFromAmountNilCurrencyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	FromAmount(big.NewRat(1, 1), nil)
}

func TestInspectStringToFToI(t *testing.T) {
	m := New(1234, usd())
	if got := m.Inspect(); got != "#<Money fractional:1234 currency:USD>" {
		t.Errorf("inspect=%q", got)
	}
	if got := m.String(); got != "12.34" {
		t.Errorf("to_s=%q", got)
	}
	if got := New(1000, jpy()).String(); got != "1000" {
		t.Errorf("jpy to_s=%q", got)
	}
	if got := m.ToF(); got != 12.34 {
		t.Errorf("to_f=%v", got)
	}
	if got := m.ToI(); got != 12 {
		t.Errorf("to_i=%d", got)
	}
}

func TestWithCurrency(t *testing.T) {
	m := New(100, usd())
	if m.WithCurrency(nil) != m {
		t.Error("nil currency should return self")
	}
	if m.WithCurrency(usd()) != m {
		t.Error("same currency should return self")
	}
	e := m.WithCurrency(eur())
	if e.Currency().ID != "eur" || e.Fractional() != 100 {
		t.Errorf("with_currency = %v", e)
	}
}

func TestRoundNoOp(t *testing.T) {
	m := New(1234, usd())
	if m.Round(HalfUp) != m {
		t.Error("Round should be a no-op returning self")
	}
}

func TestToNearestCashValue(t *testing.T) {
	// CHF smallest denomination is 5.
	chf := MustCurrency("CHF")
	got, err := New(7, chf).ToNearestCashValue()
	if err != nil || got.Fractional() != 5 {
		t.Errorf("7 -> %v (%v), want 5", got.Fractional(), err)
	}
	got, _ = New(8, chf).ToNearestCashValue()
	if got.Fractional() != 10 {
		t.Errorf("8 -> %d, want 10", got.Fractional())
	}
	// Currency with no smallest denomination errors. XAG has 0.
	if _, err := New(7, MustCurrency("XBA")).ToNearestCashValue(); err == nil {
		t.Error("expected error for undefined smallest denomination")
	}
}

func TestRoundRatModes(t *testing.T) {
	cases := []struct {
		num, den int64
		mode     RoundingMode
		want     int64
	}{
		{5, 2, HalfUp, 3},    // 2.5 -> 3
		{5, 2, HalfDown, 2},  // 2.5 -> 2
		{5, 2, HalfEven, 2},  // 2.5 -> 2
		{7, 2, HalfEven, 4},  // 3.5 -> 4
		{8, 3, HalfEven, 3},  // 2.66 -> 3 (cmp>0)
		{7, 3, HalfEven, 2},  // 2.33 -> 2 (cmp<0)
		{-5, 2, HalfUp, -3},  // -2.5 -> -3
		{3, 2, Up, 2},        // 1.5 -> 2
		{3, 2, Down, 1},      // 1.5 -> 1
		{3, 2, Ceiling, 2},   // 1.5 -> 2
		{-3, 2, Ceiling, -1}, // -1.5 -> -1
		{3, 2, Floor, 1},     // 1.5 -> 1
		{-3, 2, Floor, -2},   // -1.5 -> -2
		{4, 2, HalfUp, 2},    // exact int
		{1, 3, HalfUp, 0},    // 0.33 -> 0
		{2, 3, HalfDown, 1},  // 0.66 -> 1
	}
	for _, c := range cases {
		if got := roundRatToInt(big.NewRat(c.num, c.den), c.mode); got != c.want {
			t.Errorf("round(%d/%d, %v) = %d, want %d", c.num, c.den, c.mode, got, c.want)
		}
	}
}
