package hooks

import (
	"github.com/codiewio/codenire/internal/handler"
	"log/slog"
)

type HookRequest struct {
	Type  HookType
	Event handler.CodeHookEvent
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

func PreSandboxRequestCallback(event handler.CodeHookEvent, hookHandler HookHandler) (HookResponse, error) {
	ok, hookRes, err := invokeHookSync(HookPreSandboxRequest, event, hookHandler)
	if !ok || err != nil {
		return HookResponse{}, err
	}

	return hookRes, nil
}

func invokeHookSync(hookType HookType, event handler.CodeHookEvent, hookHandler HookHandler) (ok bool, res HookResponse, err error) {
	slog.Debug("HookInvocationStart", "type", hookType)

	res, err = hookHandler.InvokeHook(HookRequest{
		Type:  hookType,
		Event: event,
	})
	if err != nil {
		slog.Error("HookInvocationError", "type", hookType, "error", err.Error())
		return false, HookResponse{}, err
	}

	slog.Debug("HookInvocationFinish", "type", hookType)

	return true, res, nil
}
