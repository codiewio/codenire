package handler

import (
	"github.com/codiewio/codenire/pkg/hooks"
	"time"
)

type Config struct {
	BackendURL string
	Port       string

	FileHooksDir                     string
	PluginHookPath                   string
	PreRequestCallback               func(hook CodeHookEvent) (hooks.HookResponse, error)
	GracefulRequestCompletionTimeout time.Duration
	ShutdownTimeout                  time.Duration
}
