package slogsampling

import (
	"crypto/rand"
	"math/big"
)

func randomPercentage(precision int64) (float64, error) {
	random, err := rand.Int(rand.Reader, big.NewInt(precision))
	if err != nil {
		return 0, err
	}

	return float64(random.Int64()) / float64(precision), nil
}
