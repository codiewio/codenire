package handler

import (
	"time"

	"github.com/codiewio/codenire/pkg/hooks"
)

type CorsConfig struct {
	AllowOrigins     []string
	AllowCredentials bool
	AllowMethods     []string
	AllowHeaders     []string
	MaxAge           int
	ExposeHeaders    []string
}

// DefaultCorsConfig is the configuration that will be used in none is provided.
var DefaultCorsConfig = CorsConfig{
	AllowOrigins:     []string{"*"},
	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowHeaders:     []string{"Authorization", "Origin", "Content-Type", "Content-Length"},
	ExposeHeaders:    []string{"Content-Length", "Authorization", "Content-Type"},
	AllowCredentials: true,
	MaxAge:           300,
}

type Config struct {
	BackendURL string
	Port       string

	FileHooksDir                     string
	PluginHookPath                   string
	PreRequestCallback               func(hook hooks.CodeHookEvent) (hooks.HookResponse, error)
	GracefulRequestCompletionTimeout time.Duration
	ShutdownTimeout                  time.Duration
	ThrottleLimit                    int
	JWTSecretKey                     string
	Dev                              bool
	Cors                             *CorsConfig
}
