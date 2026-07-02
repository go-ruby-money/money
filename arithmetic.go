// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"fmt"
	"math/big"
)

// Neg returns a Money with the opposite sign, matching Money#-@.
func (m *Money) Neg() *Money { return m.dupWith(-m.fractional, nil) }

// Abs returns the absolute value, matching Money#abs.
func (m *Money) Abs() *Money {
	if m.fractional < 0 {
		return m.dupWith(-m.fractional, nil)
	}
	return m.dupWith(m.fractional, nil)
}

// Zero reports whether the amount is zero, matching Money#zero?.
func (m *Money) Zero() bool { return m.fractional == 0 }

// Positive reports whether the amount is greater than zero, matching
// Money#positive?.
func (m *Money) Positive() bool { return m.fractional > 0 }

// Negative reports whether the amount is less than zero, matching
// Money#negative?.
func (m *Money) Negative() bool { return m.fractional < 0 }

// Nonzero returns m when non-zero, else nil, matching Money#nonzero?.
func (m *Money) Nonzero() *Money {
	if m.fractional != 0 {
		return m
	}
	return nil
}

// Add returns the sum of m and other. When other has a different currency it is
// first exchanged to m's currency, matching Money#+. It returns an error only
// when that exchange fails (no rate / disallowed).
func (m *Money) Add(other *Money) (*Money, error) {
	o, err := other.ExchangeTo(m.currency)
	if err != nil {
		return nil, err
	}
	return m.dupWith(m.fractional+o.fractional, nil), nil
}

// Sub returns the difference m - other. When other has a different currency it
// is first exchanged to m's currency, matching Money#-.
func (m *Money) Sub(other *Money) (*Money, error) {
	o, err := other.ExchangeTo(m.currency)
	if err != nil {
		return nil, err
	}
	return m.dupWith(m.fractional-o.fractional, nil), nil
}

// Mul returns m multiplied by a scalar, matching Money#*. A non-integer product
// is rounded back to an integer number of minor units with the default rounding
// mode (the gem's return_value on the resulting BigDecimal).
func (m *Money) Mul(value *big.Rat) *Money {
	prod := new(big.Rat).Mul(big.NewRat(m.fractional, 1), value)
	return m.dupWith(roundRatToInt(prod, DefaultRoundingMode), nil)
}

// MulInt is a convenience wrapper for [Money.Mul] by an integer scalar; the
// product is always exact.
func (m *Money) MulInt(value int64) *Money {
	return m.dupWith(m.fractional*value, nil)
}

// Div returns m divided by a scalar, matching Money#/ (Money / Numeric => Money).
// The quotient is rounded back to an integer number of minor units with the
// default rounding mode. It returns an error on division by zero.
func (m *Money) Div(value *big.Rat) (*Money, error) {
	if value.Sign() == 0 {
		return nil, fmt.Errorf("money: divided by zero")
	}
	quo := new(big.Rat).Quo(big.NewRat(m.fractional, 1), value)
	return m.dupWith(roundRatToInt(quo, DefaultRoundingMode), nil), nil
}

// DivInt is a convenience wrapper for [Money.Div] by an integer scalar.
func (m *Money) DivInt(value int64) (*Money, error) {
	return m.Div(big.NewRat(value, 1))
}

// DivMoney returns the ratio m / other as an exact rational, matching the
// Money / Money form of Money#/ (which returns a Float in the gem). other is
// first exchanged to m's currency. It returns an error on a zero divisor.
func (m *Money) DivMoney(other *Money) (*big.Rat, error) {
	o, err := other.ExchangeTo(m.currency)
	if err != nil {
		return nil, err
	}
	if o.fractional == 0 {
		return nil, fmt.Errorf("money: divided by Money(0)")
	}
	return new(big.Rat).SetFrac(big.NewInt(m.fractional), big.NewInt(o.fractional)), nil
}

// DivModInt returns the integer quotient and Money remainder of dividing m by an
// integer scalar, matching Money#divmod(Integer). The quotient is a Money.
func (m *Money) DivModInt(value int64) (*Money, *Money, error) {
	if value == 0 {
		return nil, nil, fmt.Errorf("money: divided by zero")
	}
	q, r := floorDivMod(m.fractional, value)
	return m.dupWith(q, nil), m.dupWith(r, nil), nil
}

// DivModMoney returns the integer quotient and Money remainder of dividing m by
// another Money, matching Money#divmod(Money). other is first exchanged to m's
// currency.
func (m *Money) DivModMoney(other *Money) (int64, *Money, error) {
	o, err := other.ExchangeTo(m.currency)
	if err != nil {
		return 0, nil, err
	}
	if o.fractional == 0 {
		return 0, nil, fmt.Errorf("money: divided by Money(0)")
	}
	q, r := floorDivMod(m.fractional, o.fractional)
	return q, m.dupWith(r, nil), nil
}

// ModuloInt returns m modulo an integer scalar, matching Money#modulo(Integer) /
// Money#%.
func (m *Money) ModuloInt(value int64) (*Money, error) {
	_, r, err := m.DivModInt(value)
	return r, err
}

// ModuloMoney returns m modulo another Money, matching Money#modulo(Money).
func (m *Money) ModuloMoney(other *Money) (*Money, error) {
	_, r, err := m.DivModMoney(other)
	return r, err
}

// RemainderInt returns the remainder of m and an integer scalar, matching
// Money#remainder. Unlike modulo it takes the sign of the dividend: when the
// operands have opposite signs it is modulo(val) - val.
func (m *Money) RemainderInt(value int64) (*Money, error) {
	mod, err := m.ModuloInt(value)
	if err != nil {
		return nil, err
	}
	if (m.fractional < 0 && value < 0) || (m.fractional > 0 && value > 0) {
		return mod, nil
	}
	return mod.dupWith(mod.fractional-value, nil), nil
}

// floorDivMod computes the quotient and remainder of a divided by b following
// Ruby's Integer#divmod semantics: the remainder takes the sign of the divisor.
func floorDivMod(a, b int64) (q, r int64) {
	q = a / b
	r = a % b
	if r != 0 && (r < 0) != (b < 0) {
		q--
		r += b
	}
	return q, r
}

// Eql reports whether m and other have the same fractional value and currency,
// matching Money#eql? with strict comparison (0 USD is not 0 EUR).
func (m *Money) Eql(other *Money) bool {
	if other == nil {
		return false
	}
	return m.fractional == other.fractional && m.currency.Equal(other.currency)
}

// Cmp compares m to other, returning -1, 0 or 1, matching Money#<=>. When either
// side is zero the raw fractional values are compared directly; otherwise other
// is exchanged to m's currency first. It returns an error when a needed exchange
// fails.
func (m *Money) Cmp(other *Money) (int, error) {
	if m.Zero() || other.Zero() {
		return cmpInt64(m.fractional, other.fractional), nil
	}
	o, err := other.ExchangeTo(m.currency)
	if err != nil {
		return 0, err
	}
	return cmpInt64(m.fractional, o.fractional), nil
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
