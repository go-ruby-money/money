// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"fmt"
	"math/big"
)

// RoundingMode selects how a non-integer number of minor units is rounded back
// to an integer, mirroring the BigDecimal::ROUND_* modes the gem uses. The gem's
// default is HalfUp (round half away from zero).
type RoundingMode int

const (
	// HalfUp rounds to the nearest neighbour; ties round away from zero
	// (BigDecimal::ROUND_HALF_UP — the gem default).
	HalfUp RoundingMode = iota
	// HalfDown rounds to the nearest neighbour; ties round toward zero.
	HalfDown
	// HalfEven rounds to the nearest neighbour; ties round to the even neighbour
	// (banker's rounding).
	HalfEven
	// Up rounds away from zero (ceil in magnitude).
	Up
	// Down truncates toward zero.
	Down
	// Ceiling rounds toward positive infinity.
	Ceiling
	// Floor rounds toward negative infinity.
	Floor
)

// DefaultRoundingMode is the process-wide rounding mode used when converting a
// non-integer minor-unit value to an integer (Money.rounding_mode). It defaults
// to [HalfUp], as in the gem.
var DefaultRoundingMode = HalfUp

// Money is an immutable amount of a specific currency, held as an integer number
// of minor units (cents) plus a [*Currency] and a [Bank]. It is the Go analogue
// of the gem's Money value object.
type Money struct {
	fractional int64
	currency   *Currency
	bank       Bank
}

// New returns a Money of fractional minor units (cents) in the given currency,
// using the [DefaultBank] — the analogue of Money.new(fractional, currency).
func New(fractional int64, currency *Currency) *Money {
	return NewWithBank(fractional, currency, DefaultBank())
}

// NewWithBank is like [New] but binds an explicit bank for currency exchange.
func NewWithBank(fractional int64, currency *Currency, bank Bank) *Money {
	if currency == nil {
		panic("money: nil currency")
	}
	if bank == nil {
		bank = DefaultBank()
	}
	return &Money{fractional: fractional, currency: currency, bank: bank}
}

// FromAmount returns a Money for a decimal amount in the given currency, scaling
// by the currency's subunit_to_unit and rounding to an integer with the default
// rounding mode — the analogue of Money.from_amount(amount, currency). The
// amount is given as a [*big.Rat] so callers keep exact decimals.
func FromAmount(amount *big.Rat, currency *Currency) *Money {
	if currency == nil {
		panic("money: nil currency")
	}
	scaled := new(big.Rat).Mul(amount, big.NewRat(currency.SubunitToUnit, 1))
	return New(roundRatToInt(scaled, DefaultRoundingMode), currency)
}

// Fractional returns the amount in minor units (cents), the analogue of
// Money#fractional / Money#cents.
func (m *Money) Fractional() int64 { return m.fractional }

// Cents is an alias for [Money.Fractional], matching Money#cents.
func (m *Money) Cents() int64 { return m.fractional }

// Currency returns the money's currency, the analogue of Money#currency.
func (m *Money) Currency() *Currency { return m.currency }

// Bank returns the money's exchange bank, the analogue of Money#bank.
func (m *Money) Bank() Bank { return m.bank }

// Amount returns the decimal value as an exact rational — fractional divided by
// the currency's subunit_to_unit — the analogue of Money#amount / Money#to_d.
func (m *Money) Amount() *big.Rat {
	return new(big.Rat).SetFrac(big.NewInt(m.fractional), big.NewInt(m.currency.SubunitToUnit))
}

// Symbol returns the currency symbol, or "¤" when the currency has none — the
// analogue of Money#symbol.
func (m *Money) Symbol() string { return m.currency.SymbolOrDefault() }

// dupWith returns a copy of m with the given fractional and currency, keeping the
// bank — the analogue of Money#dup_with. A nil currency keeps m's currency.
func (m *Money) dupWith(fractional int64, currency *Currency) *Money {
	if currency == nil {
		currency = m.currency
	}
	return &Money{fractional: fractional, currency: currency, bank: m.bank}
}

// WithCurrency returns a Money with the same fractional value but a new currency,
// performing no conversion — the analogue of Money#with_currency.
func (m *Money) WithCurrency(c *Currency) *Money {
	if c == nil || m.currency.Equal(c) {
		return m
	}
	return m.dupWith(m.fractional, c)
}

// Inspect returns the gem's #<Money fractional:… currency:…> debug string,
// matching Money#inspect.
func (m *Money) Inspect() string {
	return fmt.Sprintf("#<Money fractional:%d currency:%s>", m.fractional, m.currency)
}

// String returns the amount as a plain decimal string with no symbol and no
// thousands separator, matching Money#to_s (no_cents_if_whole for zero-decimal
// currencies).
func (m *Money) String() string {
	return m.Format(Options{
		ThousandsSeparator: strPtr(""),
		NoCentsIfWhole:     m.currency.DecimalPlaces() == 0,
		Symbol:             SymbolOff(),
		IgnoreDefaults:     true,
	})
}

// ToF returns the amount as a float64, matching Money#to_f. Prefer [Money.Amount]
// for exact decimals; floats lose precision.
func (m *Money) ToF() float64 {
	f, _ := m.Amount().Float64()
	return f
}

// ToI returns the whole-unit part of the amount as an integer (truncated toward
// zero), matching Money#to_i.
func (m *Money) ToI() int64 { return m.fractional / m.currency.SubunitToUnit }

// Round returns a Money whose fractional value is rounded to an integer with the
// given mode — a no-op for the already-integer fractional this package holds,
// but provided for API parity with Money#round.
func (m *Money) Round(mode RoundingMode) *Money { return m }

// ToNearestCashValue rounds the amount to the nearest multiple of the currency's
// smallest_denomination, matching Money#to_nearest_cash_value. It returns an
// error when the currency has no smallest denomination.
func (m *Money) ToNearestCashValue() (*Money, error) {
	if m.currency.SmallestDenomination == 0 {
		return nil, fmt.Errorf("money: smallest denomination of %s is not defined", m.currency)
	}
	denom := m.currency.SmallestDenomination
	q := new(big.Rat).SetFrac(big.NewInt(m.fractional), big.NewInt(denom))
	rounded := roundRatToInt(q, DefaultRoundingMode) * denom
	return m.dupWith(rounded, nil), nil
}

// roundRatToInt rounds a rational to an int64 using the given mode.
func roundRatToInt(r *big.Rat, mode RoundingMode) int64 {
	if r.IsInt() {
		return r.Num().Int64()
	}
	num := new(big.Int).Set(r.Num())
	den := new(big.Int).Set(r.Denom())
	neg := num.Sign() < 0
	absNum := new(big.Int).Abs(num)

	quo := new(big.Int)
	rem := new(big.Int)
	quo.QuoRem(absNum, den, rem)

	// twice = 2*rem compared to den decides half.
	twice := new(big.Int).Lsh(rem, 1)
	cmp := twice.Cmp(den) // -1: below half, 0: exactly half, 1: above half

	roundUp := false
	switch mode {
	case HalfUp:
		roundUp = cmp >= 0
	case HalfDown:
		roundUp = cmp > 0
	case HalfEven:
		if cmp > 0 {
			roundUp = true
		} else if cmp == 0 {
			roundUp = quo.Bit(0) == 1 // round to even
		}
	case Up:
		roundUp = rem.Sign() != 0
	case Down:
		roundUp = false
	case Ceiling:
		roundUp = rem.Sign() != 0 && !neg
	case Floor:
		roundUp = rem.Sign() != 0 && neg
	}
	if roundUp {
		quo.Add(quo, big.NewInt(1))
	}
	if neg {
		quo.Neg(quo)
	}
	return quo.Int64()
}

// strPtr / boolPtr are small helpers for building Options with tri-state fields.
func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
