package client

import (
	"net/http"
	"sync"
	"time"
)

var sandboxBackendOnce struct {
	sync.Once
	c *http.Client
}

func SandboxBackendClient() *http.Client {
	sandboxBackendOnce.Do(func() {
		sandboxBackendOnce.c = &http.Client{
			Timeout: time.Second * 60,
		}
	})

	return sandboxBackendOnce.c
}
