package internal

import (
	"crypto/rand"
	"fmt"
)

func RandHex(n int) string {
	b := make([]byte, n/2)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}
