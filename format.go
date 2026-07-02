// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"math/big"
	"strings"
)

// Options are the formatting rules for [Money.Format], the analogue of the hash
// Money#format accepts. Tri-state fields (String / Symbol / …) use pointers so a
// nil pointer means "not supplied" (fall back to defaults) and a non-nil pointer
// carries an explicit value — matching how the gem distinguishes an absent key
// from a false/empty value.
type Options struct {
	// DisplayFree, when set, formats a zero amount as its string (or "free" when
	// set to the empty string via DisplayFreeText).
	DisplayFree     bool
	DisplayFreeText *string // custom free text; nil => "free"

	// WithCurrency appends the ISO code (e.g. " USD") after the amount.
	WithCurrency bool

	// NoCents omits the decimal part entirely.
	NoCents bool
	// NoCentsIfWhole omits the decimal part only when it is all zeros.
	NoCentsIfWhole bool
	// DropTrailingZeros trims trailing zeros from the decimal part.
	DropTrailingZeros bool

	// Symbol controls the currency symbol. nil => default (the currency symbol);
	// use SymbolOff(), SymbolOn() or SymbolString(s) to build it.
	Symbol *SymbolRule

	// Disambiguate uses the currency's disambiguate_symbol when available.
	Disambiguate bool

	// DecimalMark overrides the decimal separator; nil => currency/default ".".
	DecimalMark *string
	// ThousandsSeparator overrides the grouping separator; nil => currency/
	// default. A pointer to "" (or a false-y value) disables grouping.
	ThousandsSeparator *string

	// SignPositive prefixes a "+" to positive amounts.
	SignPositive bool
	// SignBeforeSymbol places the sign before the symbol (e.g. "-$1.00").
	SignBeforeSymbol bool

	// Format overrides the "%u%n" template; nil => currency/symbol-position
	// default.
	Format *string

	// SouthAsianNumberFormatting groups with the Indian lakh/crore pattern.
	SouthAsianNumberFormatting bool

	// IgnoreDefaults skips merging Money's process-wide default rules (used by
	// Money#to_s). This package has no process-wide default rules, so it only
	// affects nothing today, but is kept for API parity.
	IgnoreDefaults bool
}

// SymbolRule is the tri-state for the :symbol option: off, on (currency symbol),
// or a fixed string.
type SymbolRule struct {
	on  bool
	str *string
}

// SymbolOff disables the symbol (:symbol => false).
func SymbolOff() *SymbolRule { return &SymbolRule{on: false} }

// SymbolOn selects the currency's own symbol (:symbol => true).
func SymbolOn() *SymbolRule { return &SymbolRule{on: true} }

// SymbolString forces a specific symbol string (:symbol => "ƒ").
func SymbolString(s string) *SymbolRule { return &SymbolRule{on: true, str: &s} }

// Format renders the amount as a price string per the given options, matching
// Money#format / Money::Formatter. Passing the zero Options{} reproduces the
// gem's default formatting for the currency.
func (m *Money) Format(opts Options) string {
	// free text short-circuit.
	if m.Zero() && opts.DisplayFree {
		if opts.DisplayFreeText != nil {
			return *opts.DisplayFreeText
		}
		return "free"
	}

	number := m.formatNumber(opts)
	formatted := m.appendSign(number, opts)
	return m.appendCurrencySymbol(formatted, opts)
}

// decimalMark resolves the decimal separator.
func (m *Money) decimalMark(opts Options) string {
	if opts.DecimalMark != nil {
		return *opts.DecimalMark
	}
	if m.currency.DecimalMark != "" {
		return m.currency.DecimalMark
	}
	return "."
}

// thousandsSeparator resolves the grouping separator ("" disables grouping).
func (m *Money) thousandsSeparator(opts Options) string {
	if opts.ThousandsSeparator != nil {
		return *opts.ThousandsSeparator
	}
	if m.currency.ThousandsSeparator != "" {
		return m.currency.ThousandsSeparator
	}
	return ""
}

// formatNumber builds the delimited whole + decimal parts joined by the decimal
// mark, matching Formatter#format_number.
func (m *Money) formatNumber(opts Options) string {
	whole, decimal := m.extractWholeAndDecimal()
	decimal = m.formatDecimalPart(decimal, opts)
	whole = m.formatWholePart(whole, opts)
	if decimal == "" {
		return whole
	}
	return whole + m.decimalMark(opts) + decimal
}

// extractWholeAndDecimal splits |fractional| / subunit_to_unit into its whole and
// (unpadded) decimal digit strings, matching BigDecimal#to_s('F').split('.').
func (m *Money) extractWholeAndDecimal() (whole, decimal string) {
	frac := m.fractional
	if frac < 0 {
		frac = -frac
	}
	sub := m.currency.SubunitToUnit
	whole = big.NewInt(frac / sub).String()
	rem := frac % sub
	if rem == 0 {
		return whole, "0"
	}
	// Decimal digits: rem / sub, to exactly decimal_places digits, trailing
	// zeros trimmed to mirror BigDecimal's F formatting (which we then re-pad in
	// format_decimal_part).
	places := m.currency.DecimalPlaces()
	if places <= 0 {
		return whole, "0"
	}
	// Build rem padded to `places` digits.
	digits := big.NewInt(rem).String()
	for len(digits) < places {
		digits = "0" + digits
	}
	digits = strings.TrimRight(digits, "0")
	if digits == "" {
		digits = "0"
	}
	return whole, digits
}

// formatDecimalPart applies no_cents / no_cents_if_whole / padding / trailing-zero
// rules, matching Formatter#format_decimal_part. It returns "" when the decimal
// part should be omitted.
func (m *Money) formatDecimalPart(value string, opts Options) string {
	if m.currency.DecimalPlaces() == 0 {
		return ""
	}
	if opts.NoCents {
		return ""
	}
	if opts.NoCentsIfWhole && decimalIsZero(value) {
		return ""
	}
	// Pad to decimal_places.
	places := m.currency.DecimalPlaces()
	for len(value) < places {
		value += "0"
	}
	if opts.DropTrailingZeros {
		value = strings.TrimRight(value, "0")
	}
	return value
}

func decimalIsZero(value string) bool {
	for _, r := range value {
		if r != '0' {
			return false
		}
	}
	return true
}

// formatWholePart inserts the thousands separator, matching
// Formatter#format_whole_part with the default (or south-asian) delimiter
// pattern.
func (m *Money) formatWholePart(value string, opts Options) string {
	sep := m.thousandsSeparator(opts)
	if sep == "" {
		return value
	}
	if opts.SouthAsianNumberFormatting {
		return groupSouthAsian(value, sep)
	}
	return groupThousands(value, sep)
}

// groupThousands groups the digit string in threes from the right.
func groupThousands(value, sep string) string {
	n := len(value)
	if n <= 3 {
		return value
	}
	var b strings.Builder
	first := n % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(value[:first])
	for i := first; i < n; i += 3 {
		b.WriteString(sep)
		b.WriteString(value[i : i+3])
	}
	return b.String()
}

// groupSouthAsian groups with the Indian lakh/crore pattern: the last three
// digits, then twos.
func groupSouthAsian(value, sep string) string {
	n := len(value)
	if n <= 3 {
		return value
	}
	head := value[:n-3]
	tail := value[n-3:]
	// group head in twos from the right.
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, sep) + sep + tail
}

// appendSign applies the sign and symbol via the format template, matching
// Formatter#append_sign.
func (m *Money) appendSign(formatted string, opts Options) string {
	sign := ""
	if m.Negative() {
		sign = "-"
	}
	if opts.SignPositive && m.Positive() {
		sign = "+"
	}
	signBefore := ""
	if opts.SignBeforeSymbol {
		signBefore = sign
		sign = ""
	}

	symbolValue := m.symbolValue(opts)
	if symbolValue != "" {
		tmpl := m.formatTemplate(opts)
		tmpl = strings.ReplaceAll(tmpl, "%u", signBefore+symbolValue)
		tmpl = strings.ReplaceAll(tmpl, "%n", sign+formatted)
		return tmpl
	}
	return signBefore + sign + formatted
}

// appendCurrencySymbol appends " ISO" when with_currency is set, matching
// Formatter#append_currency_symbol.
func (m *Money) appendCurrencySymbol(formatted string, opts Options) string {
	if opts.WithCurrency {
		return formatted + " " + m.currency.String()
	}
	return formatted
}

// formatTemplate resolves the "%u%n"-style template, matching
// FormattingRules#determine_format / #default_format.
func (m *Money) formatTemplate(opts Options) string {
	if opts.Format != nil {
		return *opts.Format
	}
	if m.currency.Format != "" {
		return m.currency.Format
	}
	if m.currency.SymbolFirst {
		return "%u%n"
	}
	return "%n %u"
}

// symbolValue resolves the symbol string, matching Formatter#symbol_value_from.
func (m *Money) symbolValue(opts Options) string {
	if opts.Symbol != nil {
		if opts.Symbol.str != nil {
			return *opts.Symbol.str
		}
		if opts.Symbol.on {
			if opts.Disambiguate && m.currency.DisambiguateSymbol != "" {
				return m.currency.DisambiguateSymbol
			}
			return m.Symbol()
		}
		return ""
	}
	if opts.Disambiguate && m.currency.DisambiguateSymbol != "" {
		return m.currency.DisambiguateSymbol
	}
	return m.Symbol()
}
