package util

import (
	"fmt"
	"math/big"
)

func StringToBig(value string, base int) (*big.Int, error) {
	val, ok := new(big.Int).SetString(value, base)
	if !ok {
		return nil, fmt.Errorf("unable to parse value")
	}

	return val, nil
}
