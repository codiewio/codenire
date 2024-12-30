package main

import (
	"fmt"
	"log"

	"github.com/codenire/codenire/pkg/hooks"
	codeniredplugin "github.com/codenire/codenire/pkg/hooks/plugin"
	"github.com/hashicorp/go-plugin"
)

type MyHookHandler struct {
}

func (g *MyHookHandler) Setup() error {
	log.Println("MyHookHandler.Setup is invoked")
	return nil
}

func (g *MyHookHandler) InvokeHook(req hooks.HookRequest) (res hooks.HookResponse, err error) {
	log.Println("MyHookHandler.InvokeHook is invoked")

	res.HTTPResponse.Header = make(map[string]string)

	if req.Type == hooks.HookPreSandboxRequest {

	}

	return res, nil
}

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CODENIRE_PLUGIN",
	MagicCookieValue: "yes",
}

func main() {
	myHandler := &MyHookHandler{}

	var pluginMap = map[string]plugin.Plugin{
		"hookHandler": &codeniredplugin.HookHandlerPlugin{Impl: myHandler},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})

	fmt.Println("DOONE")
}
