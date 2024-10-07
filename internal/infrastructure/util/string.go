package util

import (
	"crypto/rand"
	"math/big"
)

func RandomStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[0]
		if randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset)))); err == nil {
			b[i] = charset[randomIndex.Int64()]
		}
	}
	return string(b)
}

func RandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return RandomStringWithCharset(length, charset)
}
