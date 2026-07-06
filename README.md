<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-money/brand/main/social/go-ruby-money-money.png" alt="go-ruby-money/money" width="720"></p>

# money — go-ruby-money

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-money.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the Ruby [`money`](https://github.com/RubyMoney/money)
gem (RubyMoney)** — the integer-cents `Money` value object, its arithmetic and
penny-safe `allocate` / `split`, the embedded ISO 4217 currency table, byte-faithful
`format`, and the variable-exchange bank. It reproduces the gem's behaviour
**without any Ruby runtime**, validated differentially against the `money` gem on
Ruby ≥ 4.0.

It is the money backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module — a sibling of
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and
[go-ruby-erb](https://github.com/go-ruby-erb/erb).

> **What it is — and isn't.** A `Money` is `(fractional int64 minor units,
> Currency, Bank)`. Arithmetic, the allocate/split remainder distribution, the
> currency data, and formatting are fully deterministic and need **no
> interpreter**, so they live here as pure Go. The **exchange-rate source is a
> host seam**: a `VariableExchange` bank holds rates the host injects
> (`AddRate`), and this library implements the exchange math; a `SingleCurrency`
> bank refuses every conversion.

## Features

Faithful port of the gem, validated against the `money` gem on every platform
that has it:

- **Integer-cents value model** — `New(fractional, currency)`, `.Fractional()` /
  `.Cents()`, `.Amount()` (exact decimal `*big.Rat`), `.Currency()`, `.Bank()`,
  and `FromAmount` (decimal → cents).
- **Arithmetic** — `Add` / `Sub` (auto-exchanging a differing currency), `Mul` /
  `Div` by a scalar, `DivMoney` (ratio), `DivMod`, `Modulo` / `%`, `Remainder`,
  `Neg`, `Abs`, `Cmp`, `Eql` — with the gem's rounding on non-integer results.
- **Penny-safe distribution** — `Allocate([weights])` and `Split(n)` reproduce
  the gem's exact remainder round-robin (**no lost pennies**), including the
  integer floored-division semantics for negative amounts.
- **ISO 4217 currency table** — the gem's `currency_iso.json` (+ non-ISO and
  backwards-compatible additions) is `go:embed`ded: `MustCurrency("USD")`,
  `FindCurrency`, `FindCurrencyByISONumeric`, `RegisterCurrency`,
  `AllCurrencies`; each carries symbol, subunit, `subunit_to_unit`, decimal mark,
  thousands separator, ISO numeric, exponent, and more.
- **Formatting** — `Format(Options{…})`: symbol / no-symbol / fixed symbol,
  decimal mark, thousands separator, symbol position, sign / sign-before-symbol /
  sign-positive, `no_cents` / `no_cents_if_whole`, `drop_trailing_zeros`,
  `display_free`, `with_currency`, `disambiguate`, custom `%u`/`%n` templates, and
  South-Asian (lakh/crore) grouping — **byte-faithful** to the gem.
- **Exchange (host seam)** — `VariableExchange` with injected rates and the gem's
  subunit-aware conversion math; `SingleCurrency` refuses exchanges;
  `DefaultBank` / `SetDefaultBank` / `AddRate` mirror the class-level API.

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x).

## Install

```sh
go get github.com/go-ruby-money/money
```

## Usage

```go
package main

import (
	"fmt"
	"math/big"

	"github.com/go-ruby-money/money"
)

func main() {
	usd := money.MustCurrency("USD")

	price := money.New(123456, usd) // 123456 cents
	fmt.Println(price.Format(money.Options{})) // $1,234.56

	// Penny-safe split — no lost cents.
	for _, part := range money.New(100, usd).Allocate([]int64{1, 1, 1}) {
		fmt.Print(part.Fractional(), " ") // 34 33 33
	}
	fmt.Println()

	// Arithmetic.
	total, _ := price.Add(money.New(44, usd))
	fmt.Println(total.Format(money.Options{})) // $1,235.00

	// Exchange — rates are a host seam.
	bank := money.NewVariableExchange()
	bank.AddRate(usd, money.MustCurrency("EUR"), big.NewRat(9, 10))
	eur, _ := bank.ExchangeWith(money.NewWithBank(1000, usd, bank), money.MustCurrency("EUR"))
	fmt.Println(eur.Fractional()) // 900
}
```

## Value model

A host such as go-embedded-ruby maps its own `Money` objects to and from this
shape:

| Ruby                         | Go                                             |
| ---------------------------- | ---------------------------------------------- |
| `Money.new(100, "USD")`      | `money.New(100, money.MustCurrency("USD"))`    |
| `m.fractional` / `m.cents`   | `m.Fractional()` (`int64`)                     |
| `m.amount` / `m.to_d`        | `m.Amount()` (`*big.Rat`)                      |
| `m.currency`                 | `m.Currency()` (`*money.Currency`)             |
| `Money::Currency.new("USD")` | `money.MustCurrency("USD")` / `money.NewCurrency` |
| `Money.default_bank`         | `money.DefaultBank()` / `money.SetDefaultBank` |
| `bank.add_rate(...)`         | `bank.AddRate(...)`                            |

## Tests & coverage

The suite pairs deterministic, ruby-free golden vectors (which alone hold
coverage at 100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential oracle** against the real `money` gem: format across
currencies+options, arithmetic, allocate/split remainder distribution, currency
lookups, and the exchange math are all compared to the gem's output. The oracle
version-gates `RUBY_VERSION >= "4.0"`, `$stdout.binmode`s its scripts, and skips
itself where `ruby` / the gem is absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-money/money authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
