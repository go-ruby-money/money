// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"testing"
)

func TestNewCurrencyAndLookups(t *testing.T) {
	c, err := NewCurrency("usd")
	if err != nil || c.ISOCode != "USD" {
		t.Fatalf("NewCurrency=%v %v", c, err)
	}
	// case-insensitive + whitespace trim.
	if c2, _ := NewCurrency("  EuR "); c2 == nil || c2.ID != "eur" {
		t.Errorf("case/space lookup failed: %v", c2)
	}
	if _, err := NewCurrency("zzz"); err == nil {
		t.Error("expected unknown currency error")
	} else if _, ok := err.(*UnknownCurrencyError); !ok {
		t.Errorf("wrong error type %T", err)
	}
}

func TestUnknownCurrencyErrorMessage(t *testing.T) {
	e := &UnknownCurrencyError{ID: "zzz"}
	if e.Error() == "" {
		t.Error("empty error message")
	}
}

func TestMustCurrencyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	MustCurrency("zzz")
}

func TestFindAndWrap(t *testing.T) {
	if FindCurrency("jpy") == nil {
		t.Error("find jpy")
	}
	if FindCurrency("zzz") != nil {
		t.Error("find unknown should be nil")
	}
	if WrapCurrency("usd") == nil {
		t.Error("wrap usd")
	}
	if WrapCurrency("zzz") != nil {
		t.Error("wrap unknown should be nil")
	}
}

func TestFindByISONumeric(t *testing.T) {
	if c := FindCurrencyByISONumeric("978"); c == nil || c.ID != "eur" {
		t.Errorf("978 -> %v", c)
	}
	if c := FindCurrencyByISONumeric("840"); c == nil || c.ID != "usd" {
		t.Errorf("840 -> %v", c)
	}
	// zero-padding a short numeric.
	if c := FindCurrencyByISONumeric("51"); c == nil || c.ID != "amd" {
		t.Errorf("51 -> %v", c)
	}
	if FindCurrencyByISONumeric("") != nil {
		t.Error("empty numeric should be nil")
	}
	if FindCurrencyByISONumeric("001") != nil {
		t.Error("001 should be nil")
	}
}

func TestRegisterUnregister(t *testing.T) {
	defer resetCurrencies()
	RegisterCurrency(Currency{
		ISOCode:       "ABC",
		Name:          "Test Coin",
		Symbol:        "A",
		SubunitToUnit: 100,
	})
	c, err := NewCurrency("abc")
	if err != nil || c.Name != "Test Coin" {
		t.Fatalf("register failed: %v %v", c, err)
	}
	if !UnregisterCurrency("ABC") {
		t.Error("unregister should report true")
	}
	if UnregisterCurrency("ABC") {
		t.Error("second unregister should report false")
	}
	if FindCurrency("abc") != nil {
		t.Error("still present after unregister")
	}
}

func TestAllCurrenciesSorted(t *testing.T) {
	all := AllCurrencies()
	if len(all) < 190 {
		t.Errorf("only %d currencies loaded", len(all))
	}
	for i := 1; i < len(all); i++ {
		p, q := all[i-1], all[i]
		if p.Priority > q.Priority || (p.Priority == q.Priority && p.ID > q.ID) {
			t.Fatalf("not sorted at %d: %s(%d) before %s(%d)", i, p.ID, p.Priority, q.ID, q.Priority)
		}
	}
	// USD has priority 1 and should be first.
	if all[0].ID != "usd" {
		t.Errorf("first currency = %s, want usd", all[0].ID)
	}
}

func TestCurrencyMethods(t *testing.T) {
	usd := MustCurrency("USD")
	if usd.String() != "USD" || usd.Code() != "$" {
		t.Errorf("string/code = %s/%s", usd.String(), usd.Code())
	}
	if usd.Exponent() != 2 || usd.DecimalPlaces() != 2 {
		t.Errorf("exponent=%d", usd.Exponent())
	}
	if !usd.CentsBased() {
		t.Error("usd cents based")
	}
	if !usd.ISO() {
		t.Error("usd iso")
	}
	jpy := MustCurrency("JPY")
	if jpy.Exponent() != 0 || jpy.CentsBased() {
		t.Errorf("jpy exponent=%d cents=%v", jpy.Exponent(), jpy.CentsBased())
	}
	bhd := MustCurrency("BHD")
	if bhd.Exponent() != 3 {
		t.Errorf("bhd exponent=%d", bhd.Exponent())
	}
	// Code() falls back to ISO code when no symbol.
	xba := MustCurrency("XBA")
	if xba.Code() != xba.ISOCode {
		t.Errorf("no-symbol code = %s", xba.Code())
	}
	if xba.ISO() {
		// XBA has an iso_numeric, so it is ISO; sanity check a non-iso one.
		btc := MustCurrency("BTC")
		if btc.ISO() {
			t.Error("btc should not be iso")
		}
	}
	if xba.SymbolOrDefault() != "¤" {
		t.Errorf("no-symbol default = %s", xba.SymbolOrDefault())
	}
	// Exponent of a degenerate currency.
	if (&Currency{SubunitToUnit: 0}).Exponent() != 0 {
		t.Error("zero subunit exponent should be 0")
	}
}

func TestCurrencyEqual(t *testing.T) {
	if !MustCurrency("USD").Equal(MustCurrency("usd")) {
		t.Error("usd == usd")
	}
	if MustCurrency("USD").Equal(MustCurrency("EUR")) {
		t.Error("usd != eur")
	}
	var a *Currency
	if a.Equal(MustCurrency("USD")) {
		t.Error("nil != usd")
	}
	if !a.Equal(nil) {
		t.Error("nil == nil")
	}
}
