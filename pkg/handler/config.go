package handler

type Config struct {
	BackendURL string
	Port       string

	FileHooksDir       string
	PluginHookPath     string
	PreRequestCallback func(hook HookEvent) (HTTPResponse, error)
}
