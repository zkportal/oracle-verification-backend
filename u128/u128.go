package u128

import (
	"errors"
	"math/big"
)

func SliceToU128(buf []byte) (*big.Int, error) {
	if len(buf) != 16 {
		return nil, errors.New("cannot convert slice to u128: invalid size")
	}

	result := big.NewInt(0)

	for idx, b := range buf {
		bigByte := big.NewInt(int64(b))
		bigByte.Lsh(bigByte, 8*uint(idx))
		result.Add(result, bigByte)
	}

	return result, nil
}
