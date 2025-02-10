package handler

import (
	"time"

	"github.com/codiewio/codenire/pkg/hooks"
)

type Config struct {
	BackendURL string
	Port       string

	FileHooksDir                     string
	PluginHookPath                   string
	PreRequestCallback               func(hook hooks.CodeHookEvent) (hooks.HookResponse, error)
	GracefulRequestCompletionTimeout time.Duration
	ShutdownTimeout                  time.Duration
	ThrottleLimit                    int
}
