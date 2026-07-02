// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import (
	"reflect"
	"testing"
)

func fracs(ms []*Money) []int64 {
	out := make([]int64, len(ms))
	for i, m := range ms {
		out[i] = m.Fractional()
	}
	return out
}

func TestAllocate(t *testing.T) {
	cases := []struct {
		amount int64
		parts  []int64
		want   []int64
	}{
		{5, []int64{3, 7}, []int64{2, 3}},
		{100, []int64{1, 1, 1}, []int64{34, 33, 33}},
		{-13, []int64{1, 1, 1}, []int64{-4, -4, -5}},
		{10, []int64{0, 0}, []int64{5, 5}}, // all-zero weights -> even
		{100, []int64{1}, []int64{100}},
	}
	for _, c := range cases {
		got := fracs(New(c.amount, usd()).Allocate(c.parts))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("allocate(%d, %v) = %v, want %v", c.amount, c.parts, got, c.want)
		}
	}
}

func TestSplit(t *testing.T) {
	if got := fracs(New(100, usd()).Split(3)); !reflect.DeepEqual(got, []int64{34, 33, 33}) {
		t.Errorf("split(3)=%v", got)
	}
	if got := fracs(New(100, usd()).Split(2)); !reflect.DeepEqual(got, []int64{50, 50}) {
		t.Errorf("split(2)=%v", got)
	}
	if got := fracs(New(100, usd()).Split(1)); !reflect.DeepEqual(got, []int64{100}) {
		t.Errorf("split(1)=%v", got)
	}
}

func TestAllocatePreservesCurrency(t *testing.T) {
	parts := New(100, eur()).Allocate([]int64{1, 1})
	for _, p := range parts {
		if p.Currency().ID != "eur" {
			t.Errorf("allocated part currency = %s", p.Currency())
		}
	}
}

func TestAllocateEmptyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic on empty parts")
		}
	}()
	New(100, usd()).Allocate(nil)
}

func TestSplitZeroPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic on split(0)")
		}
	}()
	New(100, usd()).Split(0)
}

func TestAllocateIntsEmptyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	allocateInts(1, nil)
}
