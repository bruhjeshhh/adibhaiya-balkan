package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateNumericOTP(n int) (string, error) {
	if n <= 0 {
		n = 6
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	format := fmt.Sprintf("%%0%dd", n)
	return fmt.Sprintf(format, num.Int64()), nil
}
