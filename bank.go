// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
)

// Bank is the currency-exchange seam. The rate source is supplied by the host —
// [VariableExchange] holds rates injected via AddRate, and [SingleCurrency]
// refuses every exchange. A Bank converts a Money into another currency, the
// analogue of Money::Bank::Base#exchange_with.
type Bank interface {
	// ExchangeWith converts from into to, or returns an error when no rate is
	// known (or exchange is disallowed).
	ExchangeWith(from *Money, to *Currency) (*Money, error)
}

// UnknownRateError is returned when a bank has no rate for a currency pair,
// mirroring Money::Bank::UnknownRate.
type UnknownRateError struct{ From, To string }

func (e *UnknownRateError) Error() string {
	return fmt.Sprintf("money: no conversion rate known for '%s' -> '%s'", e.From, e.To)
}

// DifferentCurrencyError is returned by [SingleCurrency] when any exchange is
// attempted, mirroring Money::Bank::DifferentCurrencyError.
type DifferentCurrencyError struct{ From, To string }

func (e *DifferentCurrencyError) Error() string {
	return fmt.Sprintf("money: no exchanging of currencies allowed: %s to %s", e.From, e.To)
}

// VariableExchange is a bank that exchanges using rates injected by the host,
// the analogue of Money::Bank::VariableExchange with the in-memory rate store.
// Rates are keyed by ISO-code pair and each direction is independent.
type VariableExchange struct {
	mu    sync.RWMutex
	rates map[string]*big.Rat
}

// NewVariableExchange returns an empty variable-exchange bank.
func NewVariableExchange() *VariableExchange {
	return &VariableExchange{rates: map[string]*big.Rat{}}
}

func rateKey(from, to *Currency) string {
	return strings.ToUpper(from.ISOCode) + "_TO_" + strings.ToUpper(to.ISOCode)
}

// AddRate registers the rate for converting from -> to and returns it, matching
// VariableExchange#add_rate / #set_rate. Only this direction is set.
func (b *VariableExchange) AddRate(from, to *Currency, rate *big.Rat) *big.Rat {
	b.mu.Lock()
	b.rates[rateKey(from, to)] = new(big.Rat).Set(rate)
	b.mu.Unlock()
	return rate
}

// GetRate returns the rate for from -> to and whether it is known, matching
// VariableExchange#get_rate.
func (b *VariableExchange) GetRate(from, to *Currency) (*big.Rat, bool) {
	b.mu.RLock()
	r, ok := b.rates[rateKey(from, to)]
	b.mu.RUnlock()
	return r, ok
}

// ExchangeWith converts from into to using the stored rate, matching
// VariableExchange#exchange_with. The fractional amount is first rescaled for
// the two currencies' subunit_to_unit, multiplied by the rate, then rounded back
// to an integer number of minor units with the default rounding mode.
func (b *VariableExchange) ExchangeWith(from *Money, to *Currency) (*Money, error) {
	if from.currency.Equal(to) {
		return from, nil
	}
	rate, ok := b.GetRate(from.currency, to)
	if !ok {
		return nil, &UnknownRateError{From: from.currency.ISOCode, To: to.ISOCode}
	}
	// calculate_fractional: fractional / (from_subunit / to_subunit)
	// = fractional * to_subunit / from_subunit
	frac := new(big.Rat).SetFrac(
		new(big.Int).Mul(big.NewInt(from.fractional), big.NewInt(to.SubunitToUnit)),
		big.NewInt(from.currency.SubunitToUnit),
	)
	exchanged := new(big.Rat).Mul(frac, rate)
	cents := roundRatToInt(exchanged, DefaultRoundingMode)
	return NewWithBank(cents, to, b), nil
}

// SingleCurrency is a bank that refuses every exchange, the analogue of
// Money::Bank::SingleCurrency — useful for apps that operate in one currency.
type SingleCurrency struct{}

// NewSingleCurrency returns a single-currency bank.
func NewSingleCurrency() *SingleCurrency { return &SingleCurrency{} }

// ExchangeWith always returns a [*DifferentCurrencyError] unless from and to are
// the same currency, matching SingleCurrency#exchange_with.
func (b *SingleCurrency) ExchangeWith(from *Money, to *Currency) (*Money, error) {
	if from.currency.Equal(to) {
		return from, nil
	}
	return nil, &DifferentCurrencyError{From: from.currency.String(), To: to.String()}
}

// defaultBank is the process-wide bank used by [New], guarded for concurrent
// swap via [SetDefaultBank] / [DefaultBank].
var (
	defaultBankMu sync.RWMutex
	defaultBank   Bank = NewVariableExchange()
)

// DefaultBank returns the process-wide default bank, the analogue of
// Money.default_bank. It defaults to an empty [VariableExchange].
func DefaultBank() Bank {
	defaultBankMu.RLock()
	defer defaultBankMu.RUnlock()
	return defaultBank
}

// SetDefaultBank sets the process-wide default bank, the analogue of
// Money.default_bank=.
func SetDefaultBank(b Bank) {
	defaultBankMu.Lock()
	defaultBank = b
	defaultBankMu.Unlock()
}

// DisallowCurrencyConversion sets the default bank to a [SingleCurrency] bank,
// the analogue of Money.disallow_currency_conversion!.
func DisallowCurrencyConversion() { SetDefaultBank(NewSingleCurrency()) }

// AddRate registers a rate on the default bank (which must be a
// [*VariableExchange]) and returns it, the analogue of Money.add_rate. It panics
// if the default bank does not support rates.
func AddRate(from, to *Currency, rate *big.Rat) *big.Rat {
	ve, ok := DefaultBank().(*VariableExchange)
	if !ok {
		panic("money: default bank does not support add_rate")
	}
	return ve.AddRate(from, to, rate)
}

// ExchangeTo converts m into another currency using m's bank, the analogue of
// Money#exchange_to. It is a no-op when already in the target currency.
func (m *Money) ExchangeTo(to *Currency) (*Money, error) {
	if m.currency.Equal(to) {
		return m, nil
	}
	return m.bank.ExchangeWith(m, to)
}
