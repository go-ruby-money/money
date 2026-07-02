// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"math/big"
	"testing"
)

func TestNegAbsSignZero(t *testing.T) {
	if New(100, usd()).Neg().Fractional() != -100 {
		t.Error("neg")
	}
	if New(-100, usd()).Abs().Fractional() != 100 {
		t.Error("abs neg")
	}
	if New(100, usd()).Abs().Fractional() != 100 {
		t.Error("abs pos")
	}
	if !New(0, usd()).Zero() || New(1, usd()).Zero() {
		t.Error("zero")
	}
	if !New(1, usd()).Positive() || New(0, usd()).Positive() {
		t.Error("positive")
	}
	if !New(-1, usd()).Negative() || New(0, usd()).Negative() {
		t.Error("negative")
	}
	if New(0, usd()).Nonzero() != nil {
		t.Error("nonzero of 0 should be nil")
	}
	if m := New(5, usd()); m.Nonzero() != m {
		t.Error("nonzero of 5 should be self")
	}
}

func TestAddSub(t *testing.T) {
	a, err := New(100, usd()).Add(New(250, usd()))
	if err != nil || a.Fractional() != 350 {
		t.Errorf("add=%v %v", a, err)
	}
	s, err := New(100, usd()).Sub(New(30, usd()))
	if err != nil || s.Fractional() != 70 {
		t.Errorf("sub=%v %v", s, err)
	}
}

func TestAddSubCrossCurrency(t *testing.T) {
	b := NewVariableExchange()
	b.AddRate(usd(), eur(), big.NewRat(1, 2)) // 1 USD -> 0.5 EUR
	m := NewWithBank(100, eur(), b)
	other := NewWithBank(200, usd(), b) // 200 cents USD -> 100 cents EUR
	sum, err := m.Add(other)
	if err != nil || sum.Fractional() != 200 || sum.Currency().ID != "eur" {
		t.Errorf("cross add = %v (%v)", sum, err)
	}
	// No rate -> error surfaces from Add and Sub.
	bad := NewWithBank(1, MustCurrency("GBP"), NewVariableExchange())
	if _, err := NewWithBank(1, usd(), NewVariableExchange()).Add(bad); err == nil {
		t.Error("expected add error on missing rate")
	}
	if _, err := NewWithBank(1, usd(), NewVariableExchange()).Sub(bad); err == nil {
		t.Error("expected sub error on missing rate")
	}
}

func TestMulDiv(t *testing.T) {
	if New(100, usd()).Mul(big.NewRat(3, 1)).Fractional() != 300 {
		t.Error("mul int")
	}
	if New(100, usd()).Mul(big.NewRat(1, 2)).Fractional() != 50 {
		t.Error("mul frac")
	}
	if New(100, usd()).MulInt(3).Fractional() != 300 {
		t.Error("mulint")
	}
	d, err := New(100, usd()).Div(big.NewRat(3, 1))
	if err != nil || d.Fractional() != 33 {
		t.Errorf("div=%v %v", d, err)
	}
	di, err := New(100, usd()).DivInt(3)
	if err != nil || di.Fractional() != 33 {
		t.Errorf("divint=%v", di)
	}
	if _, err := New(100, usd()).Div(big.NewRat(0, 1)); err == nil {
		t.Error("expected zero-division error")
	}
}

func TestDivMoney(t *testing.T) {
	r, err := New(100, usd()).DivMoney(New(10, usd()))
	if err != nil || r.Cmp(big.NewRat(10, 1)) != 0 {
		t.Errorf("divmoney=%v %v", r, err)
	}
	if _, err := New(100, usd()).DivMoney(New(0, usd())); err == nil {
		t.Error("expected zero-money division error")
	}
	// exchange failure path
	bad := NewWithBank(10, MustCurrency("GBP"), NewVariableExchange())
	if _, err := NewWithBank(100, usd(), NewVariableExchange()).DivMoney(bad); err == nil {
		t.Error("expected exchange error")
	}
}

func TestDivModAndModulo(t *testing.T) {
	q, r, err := New(100, usd()).DivModInt(9)
	if err != nil || q.Fractional() != 11 || r.Fractional() != 1 {
		t.Errorf("divmod=%v,%v,%v", q, r, err)
	}
	if _, _, err := New(100, usd()).DivModInt(0); err == nil {
		t.Error("expected zero division")
	}
	mq, mr, err := New(100, usd()).DivModMoney(New(9, usd()))
	if err != nil || mq != 11 || mr.Fractional() != 1 {
		t.Errorf("divmodmoney=%v,%v,%v", mq, mr, err)
	}
	if _, _, err := New(100, usd()).DivModMoney(New(0, usd())); err == nil {
		t.Error("expected zero money division")
	}
	// exchange failure path for divmod money
	bad := NewWithBank(9, MustCurrency("GBP"), NewVariableExchange())
	if _, _, err := NewWithBank(100, usd(), NewVariableExchange()).DivModMoney(bad); err == nil {
		t.Error("expected exchange error")
	}
	mod, err := New(100, usd()).ModuloInt(9)
	if err != nil || mod.Fractional() != 1 {
		t.Errorf("modulo=%v", mod)
	}
	if _, err := New(100, usd()).ModuloInt(0); err == nil {
		t.Error("expected error")
	}
	mmod, err := New(100, usd()).ModuloMoney(New(9, usd()))
	if err != nil || mmod.Fractional() != 1 {
		t.Errorf("modulomoney=%v", mmod)
	}
}

func TestFlooredDivMod(t *testing.T) {
	// Negative dividend takes the divisor's sign for the remainder.
	q, r := floorDivMod(-100, 9)
	if q != -12 || r != 8 {
		t.Errorf("floordivmod(-100,9)=%d,%d want -12,8", q, r)
	}
	q, r = floorDivMod(100, -9)
	if q != -12 || r != -8 {
		t.Errorf("floordivmod(100,-9)=%d,%d want -12,-8", q, r)
	}
	q, r = floorDivMod(100, 10)
	if q != 10 || r != 0 {
		t.Errorf("exact = %d,%d", q, r)
	}
}

func TestRemainder(t *testing.T) {
	// Same sign -> modulo.
	r, err := New(100, usd()).RemainderInt(9)
	if err != nil || r.Fractional() != 1 {
		t.Errorf("remainder same sign=%v", r)
	}
	// Opposite signs -> modulo(val) - val.
	r, err = New(100, usd()).RemainderInt(-9)
	// modulo(-9): floorDivMod(100,-9) => rem -8; -8 - (-9) = 1.
	if err != nil || r.Fractional() != 1 {
		t.Errorf("remainder opp sign=%v", r)
	}
	r, _ = New(-100, usd()).RemainderInt(-9)
	if r.Fractional() != -1 {
		t.Errorf("remainder both neg=%v", r)
	}
	if _, err := New(100, usd()).RemainderInt(0); err == nil {
		t.Error("expected error")
	}
}

func TestEqlAndCmp(t *testing.T) {
	if !New(100, usd()).Eql(New(100, usd())) {
		t.Error("eql same")
	}
	if New(100, usd()).Eql(New(101, usd())) {
		t.Error("eql diff amount")
	}
	if New(0, usd()).Eql(New(0, eur())) {
		t.Error("eql should be strict: 0 USD != 0 EUR")
	}
	if New(1, usd()).Eql(nil) {
		t.Error("eql nil")
	}
	c, err := New(100, usd()).Cmp(New(200, usd()))
	if err != nil || c != -1 {
		t.Errorf("cmp=%d", c)
	}
	c, _ = New(200, usd()).Cmp(New(100, usd()))
	if c != 1 {
		t.Errorf("cmp gt=%d", c)
	}
	c, _ = New(100, usd()).Cmp(New(100, usd()))
	if c != 0 {
		t.Errorf("cmp eq=%d", c)
	}
	// zero fast-path across currencies.
	c, _ = New(0, usd()).Cmp(New(5, eur()))
	if c != -1 {
		t.Errorf("cmp zero path=%d", c)
	}
	// cross-currency comparison via exchange.
	b := NewVariableExchange()
	b.AddRate(eur(), usd(), big.NewRat(2, 1)) // 1 EUR -> 2 USD
	m := NewWithBank(100, usd(), b)
	other := NewWithBank(100, eur(), b) // -> 200 USD cents
	c, err = m.Cmp(other)
	if err != nil || c != -1 {
		t.Errorf("cross cmp=%d %v", c, err)
	}
	// comparison exchange failure.
	bad := NewWithBank(5, MustCurrency("GBP"), NewVariableExchange())
	if _, err := NewWithBank(5, usd(), NewVariableExchange()).Cmp(bad); err == nil {
		t.Error("expected cmp exchange error")
	}
}
