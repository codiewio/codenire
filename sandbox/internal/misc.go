package internal

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/url"
)

func RandHex(n int) string {
	b := make([]byte, n/2)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

func ParseDsnHost(dsn string) string {
	parsedUrl, err := url.Parse(dsn)
	if err != nil {
		log.Fatal(err)
	}

	return parsedUrl.Hostname()
}
