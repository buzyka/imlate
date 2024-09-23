package util

import (
	// #nosec
	"crypto/md5"
	"encoding/hex"
)

func Md5(buffer []byte) string {
	// #nosec
	hash := md5.Sum(buffer)
	return hex.EncodeToString(hash[:])
}
