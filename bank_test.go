// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"math/big"
	"testing"
)

func TestVariableExchange(t *testing.T) {
	b := NewVariableExchange()
	if r := b.AddRate(usd(), eur(), big.NewRat(9, 10)); r.Cmp(big.NewRat(9, 10)) != 0 {
		t.Errorf("add_rate returned %v", r)
	}
	if r, ok := b.GetRate(usd(), eur()); !ok || r.Cmp(big.NewRat(9, 10)) != 0 {
		t.Errorf("get_rate = %v %v", r, ok)
	}
	if _, ok := b.GetRate(eur(), usd()); ok {
		t.Error("reverse rate should be unknown")
	}
	// 1000 USD cents -> 900 EUR cents.
	got, err := b.ExchangeWith(NewWithBank(1000, usd(), b), eur())
	if err != nil || got.Fractional() != 900 || got.Currency().ID != "eur" {
		t.Errorf("exchange = %v (%v)", got, err)
	}
	// Same currency is a no-op.
	m := NewWithBank(5, usd(), b)
	if same, _ := b.ExchangeWith(m, usd()); same != m {
		t.Error("same-currency exchange should return self")
	}
	// Unknown rate -> error.
	if _, err := b.ExchangeWith(NewWithBank(1, MustCurrency("GBP"), b), eur()); err == nil {
		t.Error("expected unknown rate error")
	} else if _, ok := err.(*UnknownRateError); !ok {
		t.Errorf("wrong error type %T", err)
	}
}

func TestVariableExchangeSubunitScaling(t *testing.T) {
	// USD (100 subunit) -> BHD (1000 subunit) at rate 1: 100 USD cents = $1.00
	// = 1000 BHD subunits.
	b := NewVariableExchange()
	b.AddRate(usd(), MustCurrency("BHD"), big.NewRat(1, 1))
	got, _ := b.ExchangeWith(NewWithBank(100, usd(), b), MustCurrency("BHD"))
	if got.Fractional() != 1000 {
		t.Errorf("subunit scaling = %d, want 1000", got.Fractional())
	}
}

func TestSingleCurrencyBank(t *testing.T) {
	b := NewSingleCurrency()
	m := NewWithBank(100, usd(), b)
	// same currency ok.
	if same, err := b.ExchangeWith(m, usd()); err != nil || same != m {
		t.Errorf("same currency = %v %v", same, err)
	}
	if _, err := b.ExchangeWith(m, eur()); err == nil {
		t.Error("expected different-currency error")
	} else if _, ok := err.(*DifferentCurrencyError); !ok {
		t.Errorf("wrong error type %T", err)
	}
}

func TestErrorMessages(t *testing.T) {
	if (&UnknownRateError{From: "USD", To: "EUR"}).Error() == "" {
		t.Error("empty unknown rate message")
	}
	if (&DifferentCurrencyError{From: "USD", To: "EUR"}).Error() == "" {
		t.Error("empty different currency message")
	}
}

func TestDefaultBank(t *testing.T) {
	orig := DefaultBank()
	defer SetDefaultBank(orig)

	ve := NewVariableExchange()
	SetDefaultBank(ve)
	if DefaultBank() != ve {
		t.Error("SetDefaultBank/DefaultBank mismatch")
	}
	// AddRate goes through the default bank.
	AddRate(usd(), eur(), big.NewRat(1, 2))
	if r, ok := ve.GetRate(usd(), eur()); !ok || r.Cmp(big.NewRat(1, 2)) != 0 {
		t.Errorf("global add_rate = %v", r)
	}
	// New() uses the default bank.
	if New(1, usd()).Bank() != ve {
		t.Error("New did not pick up default bank")
	}

	DisallowCurrencyConversion()
	if _, ok := DefaultBank().(*SingleCurrency); !ok {
		t.Error("DisallowCurrencyConversion did not install SingleCurrency")
	}
}

func TestAddRatePanicsOnNonVariableBank(t *testing.T) {
	orig := DefaultBank()
	defer SetDefaultBank(orig)
	SetDefaultBank(NewSingleCurrency())
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	AddRate(usd(), eur(), big.NewRat(1, 1))
}

func TestExchangeTo(t *testing.T) {
	b := NewVariableExchange()
	b.AddRate(usd(), eur(), big.NewRat(1, 2))
	m := NewWithBank(200, usd(), b)
	got, err := m.ExchangeTo(eur())
	if err != nil || got.Fractional() != 100 {
		t.Errorf("exchange_to = %v %v", got, err)
	}
	// no-op path.
	if same, _ := m.ExchangeTo(usd()); same != m {
		t.Error("exchange_to same currency should return self")
	}
}
