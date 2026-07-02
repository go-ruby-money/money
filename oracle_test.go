// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"math/big"
	"os/exec"
	"strings"
	"testing"
)

// moneyRuby locates a `ruby` that can `require "money"` and whose RUBY_VERSION is
// at least 4.0 (the prompt's version gate), once. The oracle tests skip
// themselves when it is absent — the qemu cross-arch lanes, the Windows lane, and
// any machine without the gem — so the deterministic golden suite alone drives
// the 100% coverage gate there.
func moneyRuby(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping money-gem oracle")
	}
	out, err := exec.Command(bin, "-e",
		`exit(RUBY_VERSION >= "4.0" ? 0 : 3) rescue exit(2)`).CombinedOutput()
	if err != nil {
		t.Skipf("ruby unusable or < 4.0 (%v): %s", err, out)
	}
	if err := exec.Command(bin, "-e", `require "money"`).Run(); err != nil {
		t.Skip("money gem not installed; skipping oracle")
	}
	return bin
}

// runMoney runs a Ruby script with the money gem required and the currency
// locale backend selected, and returns its stdout. $stdout.binmode keeps Windows
// text-mode from polluting the bytes (the go-ruby-erb lesson).
func runMoney(t *testing.T, bin, script string) string {
	t.Helper()
	full := "$stdout.binmode\nrequire 'money'\nMoney.locale_backend = :currency\n" + script
	out, err := exec.Command(bin, "-e", full).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return string(out)
}

// TestOracleFormat differentially checks Format against the gem across a matrix
// of currencies and options.
func TestOracleFormat(t *testing.T) {
	bin := moneyRuby(t)
	dm := ","
	ts := "."
	f := "%u %n"
	cases := []struct {
		frac int64
		cur  string
		opts Options
		rb   string // Ruby format(...) argument list
	}{
		{100, "USD", Options{}, ""},
		{123456, "USD", Options{}, ""},
		{-500, "USD", Options{}, ""},
		{1000, "JPY", Options{}, ""},
		{1234, "BHD", Options{}, ""},
		{100000, "EUR", Options{}, ""},
		{7, "MRU", Options{}, ""},
		{123456, "USD", Options{Symbol: SymbolOff()}, "symbol: false"},
		{599, "USD", Options{NoCents: true}, "no_cents: true"},
		{10000, "USD", Options{NoCentsIfWhole: true}, "no_cents_if_whole: true"},
		{10034, "USD", Options{NoCentsIfWhole: true}, "no_cents_if_whole: true"},
		{-100, "GBP", Options{SignBeforeSymbol: true}, "sign_before_symbol: true"},
		{100, "GBP", Options{SignPositive: true}, "sign_positive: true"},
		{85, "USD", Options{WithCurrency: true}, "with_currency: true"},
		{100, "EUR", Options{DecimalMark: &dm}, `decimal_mark: ","`},
		{100000, "USD", Options{ThousandsSeparator: &ts}, `thousands_separator: "."`},
		{10000000, "INR", Options{SouthAsianNumberFormatting: true}, "south_asian_number_formatting: true"},
		{0, "USD", Options{DisplayFree: true}, "display_free: true"},
		{10000, "USD", Options{Format: &f}, `format: "%u %n"`},
		{10000, "CAD", Options{Disambiguate: true}, "disambiguate: true"},
		{110, "USD", Options{DropTrailingZeros: true}, "drop_trailing_zeros: true"},
	}
	for _, c := range cases {
		got := New(c.frac, MustCurrency(c.cur)).Format(c.opts)
		script := "print Money.new(" + itoa(c.frac) + ", \"" + c.cur + "\").format(" + c.rb + ")"
		want := runMoney(t, bin, script)
		if got != want {
			t.Errorf("format(%d %s {%s}) = %q, want gem %q", c.frac, c.cur, c.rb, got, want)
		}
	}
}

// TestOracleArithmetic differentially checks the arithmetic operations.
func TestOracleArithmetic(t *testing.T) {
	bin := moneyRuby(t)
	check := func(script string, got int64) {
		t.Helper()
		want := strings.TrimSpace(runMoney(t, bin, "print ("+script+")"))
		if want != itoa(got) {
			t.Errorf("%s = gem %q, got %d", script, want, got)
		}
	}
	add, _ := New(100, usd()).Add(New(250, usd()))
	check(`(Money.new(100,"USD") + Money.new(250,"USD")).fractional`, add.Fractional())
	sub, _ := New(100, usd()).Sub(New(30, usd()))
	check(`(Money.new(100,"USD") - Money.new(30,"USD")).fractional`, sub.Fractional())
	check(`(Money.new(100,"USD") * 3).fractional`, New(100, usd()).Mul(big.NewRat(3, 1)).Fractional())
	check(`(Money.new(100,"USD") * 0.5).fractional`, New(100, usd()).Mul(big.NewRat(1, 2)).Fractional())
	div, _ := New(100, usd()).Div(big.NewRat(3, 1))
	check(`(Money.new(100,"USD") / 3).fractional`, div.Fractional())
	mod, _ := New(100, usd()).ModuloInt(9)
	check(`(Money.new(100,"USD") % 9).fractional`, mod.Fractional())
	check(`Money.new(-100,"USD").abs.fractional`, New(-100, usd()).Abs().Fractional())
	check(`(-Money.new(100,"USD")).fractional`, New(100, usd()).Neg().Fractional())
}

// TestOracleAllocate differentially checks allocate/split penny distribution.
func TestOracleAllocate(t *testing.T) {
	bin := moneyRuby(t)
	cases := []struct {
		frac  int64
		parts []int64
		rb    string
	}{
		{5, []int64{3, 7}, "[3,7]"},
		{100, []int64{1, 1, 1}, "[1,1,1]"},
		{-13, []int64{1, 1, 1}, "[1,1,1]"},
		{10, []int64{0, 0}, "[0,0]"},
		{100, nil, "3"},   // split(3)
		{101, nil, "7"},   // split(7)
		{99, nil, "4"},    // split(4)
		{1234, nil, "13"}, // split(13)
	}
	for _, c := range cases {
		var got []*Money
		if c.parts != nil {
			got = New(c.frac, usd()).Allocate(c.parts)
		} else {
			got = New(c.frac, usd()).Split(atoi(c.rb))
		}
		script := "print Money.new(" + itoa(c.frac) + `,"USD").allocate(` + c.rb + ").map(&:fractional).join(',')"
		want := runMoney(t, bin, script)
		gotStr := joinFracs(got)
		if gotStr != want {
			t.Errorf("allocate(%d, %s) = %q, want gem %q", c.frac, c.rb, gotStr, want)
		}
	}
}

// TestOracleCurrency differentially checks a sample of the currency table.
func TestOracleCurrency(t *testing.T) {
	bin := moneyRuby(t)
	for _, id := range []string{"usd", "jpy", "bhd", "inr", "eur", "gbp", "mru", "btc"} {
		script := "c = Money::Currency.new(\"" + id + "\")\n" +
			`print [c.iso_code, c.symbol, c.subunit_to_unit, c.decimal_mark, c.thousands_separator, c.iso_numeric, c.exponent].join("|")`
		want := runMoney(t, bin, script)
		c := MustCurrency(id)
		got := strings.Join([]string{
			c.ISOCode, c.Symbol, itoa(c.SubunitToUnit), c.DecimalMark,
			c.ThousandsSeparator, c.ISONumeric, itoa(int64(c.Exponent())),
		}, "|")
		if got != want {
			t.Errorf("currency %s = %q, want gem %q", id, got, want)
		}
	}
}

// TestOracleExchange differentially checks the variable-exchange bank math.
func TestOracleExchange(t *testing.T) {
	bin := moneyRuby(t)
	b := NewVariableExchange()
	b.AddRate(usd(), eur(), big.NewRat(9, 10))
	got, _ := b.ExchangeWith(NewWithBank(1000, usd(), b), eur())
	script := `bank = Money::Bank::VariableExchange.new
bank.add_rate("USD","EUR", 0.9)
print bank.exchange_with(Money.new(1000,"USD"), "EUR").fractional`
	want := runMoney(t, bin, script)
	if itoa(got.Fractional()) != strings.TrimSpace(want) {
		t.Errorf("exchange = %d, want gem %q", got.Fractional(), want)
	}
}

// itoa / atoi / joinFracs are tiny local helpers so the oracle test avoids the
// strconv import churn in each call site.
func itoa(n int64) string { return big.NewInt(n).String() }

func atoi(s string) int {
	n := new(big.Int)
	n.SetString(strings.TrimSpace(s), 10)
	return int(n.Int64())
}

func joinFracs(ms []*Money) string {
	parts := make([]string, len(ms))
	for i, m := range ms {
		parts[i] = itoa(m.Fractional())
	}
	return strings.Join(parts, ",")
}
