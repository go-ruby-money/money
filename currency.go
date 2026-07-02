// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package money is a pure-Go (CGO-free), MRI-faithful reimplementation of the
// Ruby `money` gem (RubyMoney). It models an amount of a specific currency as an
// integer number of minor units (cents) plus a [*Currency], and reproduces the
// gem's arithmetic, allocate/split penny distribution, ISO 4217 currency table,
// formatting, and variable-exchange bank — byte-for-byte where the gem is
// deterministic — without any Ruby runtime.
//
// The rate source for currency exchange is a host seam: a [*VariableExchange]
// bank holds rates injected by the host (Money.add_rate / bank.add_rate), and
// the exchange math is implemented here. A [*SingleCurrency] bank refuses any
// exchange.
//
// # Value model
//
// A Money value is (fractional int64 minor units, currency, bank). A host such
// as go-embedded-ruby maps its own Money objects to and from this shape:
//
//	Ruby                       Go
//	----                       --
//	Money.new(100, "USD")      money.New(100, money.MustCurrency("USD"))
//	m.fractional / m.cents     m.Fractional() (int64)
//	m.amount / m.to_d          m.Amount() (*big.Rat)
//	m.currency                 m.Currency() (*Currency)
//	Money::Currency.new("USD") money.Currency(...) / money.MustCurrency(...)
package money

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

//go:embed config/currency_iso.json
var currencyISOJSON []byte

//go:embed config/currency_non_iso.json
var currencyNonISOJSON []byte

//go:embed config/currency_backwards_compatible.json
var currencyBackwardsCompatibleJSON []byte

// Currency represents a specific currency unit — the Go analogue of
// Money::Currency. It is loaded from the embedded ISO 4217 table (with the
// gem's non-ISO and backwards-compatible additions merged in) and is immutable
// once registered. Look one up with [Currency] / [MustCurrency] / [FindCurrency]
// / [FindCurrencyByISONumeric], or add your own with [RegisterCurrency].
type Currency struct {
	// ID is the lowercase identifier (e.g. "usd"); it is the canonical key.
	ID string
	// Priority orders the currency list (lower sorts first).
	Priority int
	// ISOCode is the upper-case 3-letter ISO 4217 code (e.g. "USD").
	ISOCode string
	// ISONumeric is the 3-digit ISO 4217 numeric code, zero-padded (e.g. "840").
	ISONumeric string
	// Name is the currency's human name (e.g. "United States Dollar").
	Name string
	// Symbol is the currency symbol (e.g. "$"); may be empty.
	Symbol string
	// DisambiguateSymbol is an alternative symbol used when Symbol is ambiguous
	// (e.g. "US$" / "C$").
	DisambiguateSymbol string
	// HTMLEntity is the HTML entity for the symbol (e.g. "$", "&#x00A5;").
	HTMLEntity string
	// AlternateSymbols are other symbols in use for the currency.
	AlternateSymbols []string
	// Subunit is the name of the fractional unit (e.g. "Cent"); may be empty.
	Subunit string
	// SubunitToUnit is the proportion between unit and subunit (e.g. 100).
	SubunitToUnit int64
	// DecimalMark separates the whole part from the subunit (e.g. ".").
	DecimalMark string
	// ThousandsSeparator groups the whole part (e.g. ",").
	ThousandsSeparator string
	// SymbolFirst reports whether the symbol precedes the amount.
	SymbolFirst bool
	// SmallestDenomination is the smallest cash amount, in subunits (e.g. 1, 5).
	SmallestDenomination int64
	// Format is an optional "%u%n"-style template overriding the default layout.
	Format string
}

// rawCurrency mirrors the JSON shape in config/*.json. Numeric fields that the
// gem's data sometimes ships as an empty string (a nil denomination) are decoded
// through [flexInt].
type rawCurrency struct {
	Priority             *int     `json:"priority"`
	ISOCode              string   `json:"iso_code"`
	ISONumeric           string   `json:"iso_numeric"`
	Name                 string   `json:"name"`
	Symbol               string   `json:"symbol"`
	DisambiguateSymbol   string   `json:"disambiguate_symbol"`
	HTMLEntity           string   `json:"html_entity"`
	AlternateSymbols     []string `json:"alternate_symbols"`
	Subunit              string   `json:"subunit"`
	SubunitToUnit        flexInt  `json:"subunit_to_unit"`
	DecimalMark          string   `json:"decimal_mark"`
	ThousandsSeparator   string   `json:"thousands_separator"`
	SymbolFirst          bool     `json:"symbol_first"`
	SmallestDenomination flexInt  `json:"smallest_denomination"`
	Format               string   `json:"format"`
}

// flexInt decodes a JSON number, or an empty string / null as zero — the shape
// the gem's currency data uses for an undefined denomination.
type flexInt int64

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == `""` || s == "" {
		*f = 0
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexInt(n)
	return nil
}

// registry holds the loaded currency table, guarded for concurrent registration.
var registry struct {
	mu    sync.RWMutex
	table map[string]*Currency // by lowercase id
}

func init() { resetCurrencies() }

// resetCurrencies rebuilds the table from the embedded JSON, matching the gem's
// Loader.load_currencies merge order (iso, then non-iso, then backwards-compat).
func resetCurrencies() {
	table, err := loadCurrencyTable(currencyISOJSON, currencyNonISOJSON, currencyBackwardsCompatibleJSON)
	if err != nil {
		panic("money: embedded currency data is corrupt: " + err.Error())
	}
	registry.mu.Lock()
	registry.table = table
	registry.mu.Unlock()
}

// loadCurrencyTable parses and merges the given JSON blobs into a currency table
// keyed by lowercase id. It returns an error on malformed JSON.
func loadCurrencyTable(blobs ...[]byte) (map[string]*Currency, error) {
	table := map[string]*Currency{}
	for _, blob := range blobs {
		var raw map[string]rawCurrency
		if err := json.Unmarshal(blob, &raw); err != nil {
			return nil, err
		}
		for id, rc := range raw {
			table[strings.ToLower(id)] = rc.toCurrency(id)
		}
	}
	return table, nil
}

func (rc rawCurrency) toCurrency(id string) *Currency {
	c := &Currency{
		ID:                   strings.ToLower(id),
		ISOCode:              rc.ISOCode,
		ISONumeric:           rc.ISONumeric,
		Name:                 rc.Name,
		Symbol:               rc.Symbol,
		DisambiguateSymbol:   rc.DisambiguateSymbol,
		HTMLEntity:           rc.HTMLEntity,
		AlternateSymbols:     rc.AlternateSymbols,
		Subunit:              rc.Subunit,
		SubunitToUnit:        int64(rc.SubunitToUnit),
		DecimalMark:          rc.DecimalMark,
		ThousandsSeparator:   rc.ThousandsSeparator,
		SymbolFirst:          rc.SymbolFirst,
		SmallestDenomination: int64(rc.SmallestDenomination),
		Format:               rc.Format,
	}
	if rc.Priority != nil {
		c.Priority = *rc.Priority
	}
	return c
}

// UnknownCurrencyError is returned when a currency id / code cannot be resolved,
// mirroring Money::Currency::UnknownCurrency.
type UnknownCurrencyError struct{ ID string }

func (e *UnknownCurrencyError) Error() string {
	return fmt.Sprintf("money: unknown currency '%s'", e.ID)
}

// NewCurrency looks up a currency by its id or ISO code (case-insensitive), the
// analogue of Money::Currency.new. It returns an [*UnknownCurrencyError] for an
// unknown id.
func NewCurrency(id string) (*Currency, error) { return currencyByID(id) }

// currencyByID is the shared lookup used by the public constructors.
func currencyByID(id string) (*Currency, error) {
	key := strings.ToLower(strings.TrimSpace(id))
	registry.mu.RLock()
	c, ok := registry.table[key]
	registry.mu.RUnlock()
	if !ok {
		return nil, &UnknownCurrencyError{ID: id}
	}
	return c, nil
}

// MustCurrency is like [NewCurrency] but panics on an unknown id; it is convenient
// for currencies known to exist at call sites (e.g. MustCurrency("USD")).
func MustCurrency(id string) *Currency {
	c, err := currencyByID(id)
	if err != nil {
		panic(err)
	}
	return c
}

// FindCurrency looks up a currency by id / code, returning nil (no error) when
// it is unknown — the analogue of Money::Currency.find.
func FindCurrency(id string) *Currency {
	c, err := currencyByID(id)
	if err != nil {
		return nil
	}
	return c
}

// WrapCurrency returns c unchanged when non-nil, else looks id up. It mirrors
// Money::Currency.wrap for the string case and is used internally to accept
// either a *Currency or a code where the gem does.
func WrapCurrency(id string) *Currency { return FindCurrency(id) }

// FindCurrencyByISONumeric looks a currency up by its ISO 4217 numeric code
// (e.g. 840 or "840"), zero-padding to three digits first, matching
// Money::Currency.find_by_iso_numeric. It returns nil when unknown.
func FindCurrencyByISONumeric(num string) *Currency {
	num = strings.TrimSpace(num)
	if num == "" {
		return nil
	}
	for len(num) < 3 {
		num = "0" + num
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	for _, c := range registry.table {
		if c.ISONumeric == num {
			return c
		}
	}
	return nil
}

// RegisterCurrency adds or replaces a currency keyed by its (lowercased) ISO
// code, matching Money::Currency.register. A caller that mutates the returned
// table after registration does not affect the registry.
func RegisterCurrency(c Currency) {
	c.ID = strings.ToLower(c.ISOCode)
	cp := c
	registry.mu.Lock()
	registry.table[cp.ID] = &cp
	registry.mu.Unlock()
}

// UnregisterCurrency removes a currency by id / code and reports whether it
// existed, matching Money::Currency.unregister.
func UnregisterCurrency(id string) bool {
	key := strings.ToLower(strings.TrimSpace(id))
	registry.mu.Lock()
	defer registry.mu.Unlock()
	_, ok := registry.table[key]
	if ok {
		delete(registry.table, key)
	}
	return ok
}

// AllCurrencies returns every registered currency sorted by priority then id,
// matching Money::Currency.all / .each iteration order.
func AllCurrencies() []*Currency {
	registry.mu.RLock()
	out := make([]*Currency, 0, len(registry.table))
	for _, c := range registry.table {
		out = append(out, c)
	}
	registry.mu.RUnlock()
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// String returns the upper-case ISO id (e.g. "USD"), matching Currency#to_s.
func (c *Currency) String() string { return strings.ToUpper(c.ID) }

// Code returns the symbol, or the ISO code when there is no symbol, matching
// Currency#code.
func (c *Currency) Code() string {
	if c.Symbol != "" {
		return c.Symbol
	}
	return c.ISOCode
}

// Equal reports whether two currencies denote the same currency (same id),
// matching Currency#==. A nil receiver equals only a nil other.
func (c *Currency) Equal(other *Currency) bool {
	if c == nil || other == nil {
		return c == other
	}
	return c.ID == other.ID
}

// Exponent returns the base-10 exponent relating subunit to unit, i.e.
// round(log10(subunit_to_unit)), matching Currency#exponent / #decimal_places.
func (c *Currency) Exponent() int {
	if c.SubunitToUnit <= 0 {
		return 0
	}
	return int(math.Round(math.Log10(float64(c.SubunitToUnit))))
}

// DecimalPlaces is an alias for [Currency.Exponent], matching the gem's alias.
func (c *Currency) DecimalPlaces() int { return c.Exponent() }

// CentsBased reports whether the subunit is hundredths, matching
// Currency#cents_based?.
func (c *Currency) CentsBased() bool { return c.SubunitToUnit == 100 }

// ISO reports whether the currency carries an ISO 4217 numeric code, matching
// Currency#iso?.
func (c *Currency) ISO() bool { return c.ISONumeric != "" }

// SymbolOrDefault returns the currency symbol, or the generic "¤" placeholder
// when the currency has none — the value Money#symbol uses.
func (c *Currency) SymbolOrDefault() string {
	if c.Symbol == "" {
		return "¤"
	}
	return c.Symbol
}
