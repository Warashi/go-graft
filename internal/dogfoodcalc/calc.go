package dogfoodcalc

import "math/big"

func AddInt64(a int64, b int64) int64 {
	left := big.NewInt(a)
	right := big.NewInt(b)
	return new(big.Int).Add(left, right).Int64()
}
