package autils

import (
	"math/rand"
	"time"
)

const mixedCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(charset))]
	}
	return string(b)
}

// RandomString :
func RandomString(length int) string {
	return stringWithCharset(length, mixedCharset)
}
