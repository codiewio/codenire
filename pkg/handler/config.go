package handler

type Config struct {
	BackendURL     string
	Port           string
	FileHooksDir   string
	PluginHookPath string

	PreRunSandboxCallback func(hook HookEvent) (HTTPResponse, error)
}
