package dogfoodcalc

import (
	"math"
	"math/big"
	"slices"
)

func AddInt64(a int64, b int64) int64 {
	left := big.NewInt(a)
	right := big.NewInt(b)
	return new(big.Int).Add(left, right).Int64()
}

func MakeBigIntFromInt64(v int64) *big.Int {
	return big.NewInt(v)
}

func MakeNegatedBigIntFromInt64(v int64) *big.Int {
	return big.NewInt(-v)
}

func AddInt64ViaIdent(a int64, b int64) int64 {
	left := MakeBigIntFromInt64(a)
	right := MakeBigIntFromInt64(b)
	return new(big.Int).Add(left, right).Int64()
}

func AbsFloat64(v float64) float64 {
	return math.Abs(v)
}

func MinInt(xs []int) int {
	return slices.Min[[]int, int](xs)
}
