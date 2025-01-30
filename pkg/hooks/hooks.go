package hooks

import (
	"github.com/codiewio/codenire/internal/handler"
	"log/slog"
)

type HookRequest struct {
	Type  HookType
	Event handler.HookEvent
}

type HookResponse struct {
	HTTPResponse handler.HTTPResponse
}

type HookHandler interface {
	Setup() error
	InvokeHook(req HookRequest) (res HookResponse, err error)
}

type HookType string

const (
	HookPreSandboxRequest HookType = "pre-sandbox-request"
)

var AvailableHooks []HookType = []HookType{HookPreSandboxRequest}

func PreSandboxRequestCallback(event handler.HookEvent, hookHandler HookHandler) (handler.HTTPResponse, error) {
	ok, hookRes, err := invokeHookSync(HookPreSandboxRequest, event, hookHandler)
	if !ok || err != nil {
		return handler.HTTPResponse{}, err
	}

	httpRes := hookRes.HTTPResponse

	return httpRes, nil
}

func invokeHookSync(typ HookType, event handler.HookEvent, hookHandler HookHandler) (ok bool, res HookResponse, err error) {
	slog.Debug("HookInvocationStart", "type", typ)

	res, err = hookHandler.InvokeHook(HookRequest{
		Type:  typ,
		Event: event,
	})
	if err != nil {
		slog.Error("HookInvocationError", "type", typ, "error", err.Error())
		return false, HookResponse{}, err
	}

	slog.Debug("HookInvocationFinish", "type", typ)

	return true, res, nil
}
