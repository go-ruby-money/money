// Copyright (c) the go-ruby-money/money authors
//
// SPDX-License-Identifier: BSD-3-Clause

package money

import "math/big"

// Allocate distributes m into parts proportional to the given weights, without
// losing pennies — the analogue of Money#allocate(parts). The left-over minor
// units are handed out so that earlier parts receive them first, exactly as the
// gem does. Passing all-zero weights splits evenly. It panics on an empty parts
// slice (matching the gem's ArgumentError).
func (m *Money) Allocate(parts []int64) []*Money {
	amounts := allocateInts(m.fractional, parts)
	out := make([]*Money, len(amounts))
	for i, a := range amounts {
		out[i] = m.dupWith(a, nil)
	}
	return out
}

// Split divides m evenly into n parts without losing pennies — the analogue of
// Money#split(n) (an alias of allocate with equal weights). It panics when n < 1.
func (m *Money) Split(n int) []*Money {
	if n < 1 {
		panic("money: need at least one part")
	}
	weights := make([]int64, n)
	for i := range weights {
		weights[i] = 1
	}
	return m.Allocate(weights)
}

// allocateInts is the integer core of Money::Allocation.generate with the
// default whole-amount truncation. It reproduces the gem's loop: parts are
// consumed from the end, each split is truncate(remaining*part/parts_sum), and
// the running remainder feeds the next iteration — so any left-over units land
// on the earliest parts.
func allocateInts(amount int64, parts []int64) []int64 {
	if len(parts) == 0 {
		panic("money: need at least one part")
	}
	// When every weight is zero, the gem replaces them with all-ones.
	allZero := true
	for _, p := range parts {
		if p != 0 {
			allZero = false
			break
		}
	}
	work := make([]int64, len(parts))
	if allZero {
		for i := range work {
			work[i] = 1
		}
	} else {
		copy(work, parts)
	}

	result := make([]int64, len(work))
	remaining := big.NewInt(amount)
	partsSum := int64(0)
	for _, p := range work {
		partsSum += p
	}
	sum := big.NewInt(partsSum)

	for i := len(work) - 1; i >= 0; i-- {
		part := big.NewInt(work[i])
		split := big.NewInt(0)
		if sum.Sign() > 0 {
			// The gem keeps integer amounts as Integers, so the split is Ruby's
			// Integer division (which floors toward negative infinity) followed by
			// a no-op truncate. big.Int.Div implements the same floored division.
			num := new(big.Int).Mul(remaining, part)
			split = new(big.Int).Div(num, sum)
		}
		result[i] = split.Int64()
		remaining.Sub(remaining, split)
		sum.Sub(sum, part)
	}
	return result
}
