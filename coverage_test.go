// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import "testing"

// TestFormatDefaultSeparators registers a currency whose decimal_mark and
// thousands_separator are empty, so Format falls back to the built-in "." mark
// and the empty (no-grouping) separator — branches the shipped ISO data never
// reaches.
func TestFormatDefaultSeparators(t *testing.T) {
	defer resetCurrencies()
	RegisterCurrency(Currency{
		ISOCode:            "ZZA",
		Name:               "No Separators",
		Symbol:             "Z",
		SubunitToUnit:      100,
		SymbolFirst:        true,
		DecimalMark:        "",
		ThousandsSeparator: "",
	})
	got := New(123456, MustCurrency("ZZA")).Format(Options{})
	if got != "Z1234.56" {
		t.Errorf("default separators = %q, want Z1234.56", got)
	}
}

// TestFormatSubunitOne registers a subunit_to_unit=1 currency (exponent 0) to
// hit the decimal_places<=0 path in extractWholeAndDecimal.
func TestFormatSubunitOne(t *testing.T) {
	defer resetCurrencies()
	RegisterCurrency(Currency{
		ISOCode:            "ZZB",
		Name:               "Unit Only",
		Symbol:             "U",
		SubunitToUnit:      1,
		SymbolFirst:        true,
		ThousandsSeparator: ",",
	})
	got := New(1234, MustCurrency("ZZB")).Format(Options{})
	if got != "U1,234" {
		t.Errorf("subunit-one = %q, want U1,234", got)
	}
}

// TestFormatCurrencyOwnTemplate registers a currency that carries an explicit
// format string, exercising the currency.Format branch of formatTemplate.
func TestFormatCurrencyOwnTemplate(t *testing.T) {
	defer resetCurrencies()
	RegisterCurrency(Currency{
		ISOCode:       "ZZC",
		Name:          "Templated",
		Symbol:        "T",
		SubunitToUnit: 100,
		SymbolFirst:   true,
		Format:        "%n%u",
	})
	got := New(100, MustCurrency("ZZC")).Format(Options{})
	if got != "1.00T" {
		t.Errorf("own template = %q, want 1.00T", got)
	}
}

// TestGroupingShortValues covers the n<=3 early return of both grouping helpers.
func TestGroupingShortValues(t *testing.T) {
	if got := groupThousands("12", ","); got != "12" {
		t.Errorf("short thousands = %q", got)
	}
	if got := groupSouthAsian("12", ","); got != "12" {
		t.Errorf("short south-asian = %q", got)
	}
	if got := groupThousands("123456", ","); got != "123,456" {
		t.Errorf("exact-multiple thousands = %q", got)
	}
	// South-asian with a single leading digit (head <= 2 loop skipped).
	if got := groupSouthAsian("1234", ","); got != "1,234" {
		t.Errorf("south-asian 1234 = %q", got)
	}
}

// TestFlexIntNullAndBad covers the null and error branches of flexInt.
func TestFlexIntNullAndBad(t *testing.T) {
	var f flexInt
	if err := f.UnmarshalJSON([]byte("null")); err != nil || f != 0 {
		t.Errorf("null = %v %v", f, err)
	}
	if err := f.UnmarshalJSON([]byte(`""`)); err != nil || f != 0 {
		t.Errorf("empty = %v %v", f, err)
	}
	if err := f.UnmarshalJSON([]byte("42")); err != nil || f != 42 {
		t.Errorf("num = %v %v", f, err)
	}
	if err := f.UnmarshalJSON([]byte(`"oops"`)); err == nil {
		t.Error("expected error on non-numeric string")
	}
}

// TestResetCurrenciesPanicsOnCorruptData covers the panic branch of
// resetCurrencies by temporarily corrupting the embedded ISO blob.
func TestResetCurrenciesPanicsOnCorruptData(t *testing.T) {
	orig := currencyISOJSON
	defer func() {
		currencyISOJSON = orig
		resetCurrencies() // restore a good table for later tests
		if r := recover(); r == nil {
			t.Error("expected panic on corrupt embedded data")
		}
	}()
	currencyISOJSON = []byte("not json")
	resetCurrencies()
}

// TestLoadCurrencyTableError covers the JSON-error branch of loadCurrencyTable.
func TestLoadCurrencyTableError(t *testing.T) {
	if _, err := loadCurrencyTable([]byte("not json")); err == nil {
		t.Error("expected error on malformed currency JSON")
	}
	// Valid input succeeds.
	if tbl, err := loadCurrencyTable([]byte(`{"foo":{"iso_code":"FOO","subunit_to_unit":100}}`)); err != nil || tbl["foo"] == nil {
		t.Errorf("valid table = %v %v", tbl, err)
	}
}

// TestExtractZeroDecimalCurrency exercises the rem==0 path for a hundredth
// currency whose fractional value is a whole number of units.
func TestExtractZeroDecimalCurrency(t *testing.T) {
	whole, decimal := New(500, usd()).extractWholeAndDecimal()
	if whole != "5" || decimal != "0" {
		t.Errorf("extract 500 usd = %q,%q", whole, decimal)
	}
}
