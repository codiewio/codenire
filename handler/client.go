package handler

import (
	"net/http"
	"sync"
)

var sandboxBackendOnce struct {
	sync.Once
	c *http.Client
}

func SandboxBackendClient() *http.Client {
	sandboxBackendOnce.Do(func() {
		sandboxBackendOnce.c = http.DefaultClient
	})

	return sandboxBackendOnce.c
}
