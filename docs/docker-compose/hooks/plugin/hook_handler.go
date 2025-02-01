package main

import (
	"log"

	"github.com/codiewio/codenire/pkg/hooks"
	codeniredplugin "github.com/codiewio/codenire/pkg/hooks/plugin"
	"github.com/hashicorp/go-plugin"
)

type CodenireHandler struct {
}

func (g *CodenireHandler) Setup() (*hooks.ExternalTemplates, error) {
	log.Println("CodenireHandler.Setup is invoked")
	return nil, nil
}

func (g *CodenireHandler) InvokeHook(req hooks.HookRequest) (res hooks.HookResponse, err error) {
	log.Println("CodenireHandler.InvokeHook is invoked")

	res.Header = make(map[string]string)

	if req.Type == hooks.HookPreSandboxRequest {
		// Some handle logic (auth or transform request or separate code handle)
	}

	return res, nil
}

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CODENIRE_PLUGIN",
	MagicCookieValue: "yes",
}

func main() {
	myHandler := &CodenireHandler{}

	var pluginMap = map[string]plugin.Plugin{
		"hookHandler": &codeniredplugin.HookHandlerPlugin{Impl: myHandler},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
