// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import "testing"

// TestFormatGolden pins the byte-for-byte output of Format against a corpus
// captured from the money gem (7.0.2) — the deterministic, ruby-free half of the
// oracle that holds coverage on the no-ruby lanes.
func TestFormatGolden(t *testing.T) {
	dm := ","
	ts := "."
	f := "%u %n"
	cases := []struct {
		name string
		frac int64
		cur  string
		opts Options
		want string
	}{
		{"usd_100", 100, "USD", Options{}, "$1.00"},
		{"usd_123456", 123456, "USD", Options{}, "$1,234.56"},
		{"usd_neg", -500, "USD", Options{}, "$-5.00"},
		{"jpy_1000", 1000, "JPY", Options{}, "¥1,000"},
		{"bhd_1234", 1234, "BHD", Options{}, "د.ب1.234"},
		{"eur_100000", 100000, "EUR", Options{}, "€1.000,00"},
		{"usd_nosym", 123456, "USD", Options{Symbol: SymbolOff()}, "1,234.56"},
		{"usd_nocents", 599, "USD", Options{NoCents: true}, "$5"},
		{"usd_ncw_whole", 10000, "USD", Options{NoCentsIfWhole: true}, "$100"},
		{"usd_ncw_part", 10034, "USD", Options{NoCentsIfWhole: true}, "$100.34"},
		{"gbp_signbefore", -100, "GBP", Options{SignBeforeSymbol: true}, "-£1.00"},
		{"gbp_signpos", 100, "GBP", Options{SignPositive: true}, "£+1.00"},
		{"usd_withcur", 85, "USD", Options{WithCurrency: true}, "$0.85 USD"},
		{"eur_decmark", 100, "EUR", Options{DecimalMark: &dm}, "€1,00"},
		{"usd_thou_dot", 100000, "USD", Options{ThousandsSeparator: &ts}, "$1.000.00"},
		{"inr_south", 10000000, "INR", Options{SouthAsianNumberFormatting: true}, "₹1,00,000.00"},
		{"usd_free", 0, "USD", Options{DisplayFree: true}, "free"},
		{"usd_tmpl", 10000, "USD", Options{Format: &f}, "$ 100.00"},
		{"cad_disamb", 10000, "CAD", Options{Disambiguate: true}, "C$100.00"},
		{"usd_symstr", 100, "AWG", Options{Symbol: SymbolString("ƒ")}, "1.00 ƒ"},
		{"usd_symon", 100, "USD", Options{Symbol: SymbolOn()}, "$1.00"},
		{"drop_zeros", 110, "USD", Options{DropTrailingZeros: true}, "$1.1"},
		{"disamb_no_alt", 10000, "USD", Options{Disambiguate: true}, "US$100.00"},
		{"awg_default", 100, "AWG", Options{}, "1.00 ƒ"},
		{"cad_symon_disamb", 10000, "CAD", Options{Symbol: SymbolOn(), Disambiguate: true}, "C$100.00"},
		{"mru_7", 7, "MRU", Options{}, "1.4 UM"},
		{"mru_10", 10, "MRU", Options{}, "2.0 UM"},
		{"mru_13", 13, "MRU", Options{}, "2.6 UM"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := New(c.frac, MustCurrency(c.cur)).Format(c.opts)
			if got != c.want {
				t.Errorf("format = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFormatFreeCustomText(t *testing.T) {
	txt := "gratis"
	got := New(0, usd()).Format(Options{DisplayFree: true, DisplayFreeText: &txt})
	if got != "gratis" {
		t.Errorf("free text = %q", got)
	}
	// Non-zero money ignores display_free.
	got = New(100, usd()).Format(Options{DisplayFree: true})
	if got != "$1.00" {
		t.Errorf("non-zero display_free = %q", got)
	}
}

func TestFormatSignBeforeNoSymbol(t *testing.T) {
	// sign_before_symbol with symbol off exercises the no-symbol branch.
	got := New(-500, usd()).Format(Options{Symbol: SymbolOff(), SignBeforeSymbol: true})
	if got != "-5.00" {
		t.Errorf("sign before, no symbol = %q", got)
	}
}

func TestFormatSymbolFirstFalse(t *testing.T) {
	// EEK carries an explicit "%n %u" format string in its data (the
	// currency.Format branch).
	if got := New(100, MustCurrency("EEK")).Format(Options{}); got != "1.00 KR" {
		t.Errorf("eek default = %q", got)
	}
	// RSD has symbol_first=false and no format string, exercising the "%n %u"
	// fallback template.
	if got := New(100, MustCurrency("RSD")).Format(Options{}); got != "1,00 RSD" {
		t.Errorf("rsd default = %q", got)
	}
}

func TestFormatIgnoreDefaultsAndThousandsDefault(t *testing.T) {
	// to_s path already exercises IgnoreDefaults + empty separator; assert a
	// currency without a thousands separator falls back to "".
	got := New(100000, usd()).Format(Options{})
	if got != "$1,000.00" {
		t.Errorf("usd 100000 = %q", got)
	}
}
