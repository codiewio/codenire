package hooks

import (
	"context"
	api "github.com/codiewio/codenire/api/gen"
	"log/slog"
)

type HookRequest struct {
	Type  HookType
	Event CodeHookEvent
}

type HookHandler interface {
	Setup() error
	InvokeHook(req HookRequest) (res HookResponse, err error)
}

type CodeHookEvent struct {
	Context           context.Context
	SubmissionRequest api.SubmissionRequest
	HTTPRequest       HTTPRequest
}

func NewCodeHookEvent(c *HttpContext, sr api.SubmissionRequest) CodeHookEvent {
	return CodeHookEvent{
		Context:           c,
		SubmissionRequest: sr,
		HTTPRequest: HTTPRequest{
			Method:     c.req.Method,
			URI:        c.req.RequestURI,
			RemoteAddr: c.req.RemoteAddr,
			Header:     c.req.Header,
		},
	}
}

type HookType string

const (
	HookPreSandboxRequest HookType = "pre-sandbox-request"
)

var AvailableHooks []HookType = []HookType{HookPreSandboxRequest}

func PreSandboxRequestCallback(event CodeHookEvent, hookHandler HookHandler) (HookResponse, error) {
	ok, hookRes, err := invokeHookSync(HookPreSandboxRequest, event, hookHandler)
	if !ok || err != nil {
		return HookResponse{}, err
	}

	return hookRes, nil
}

func invokeHookSync(hookType HookType, event CodeHookEvent, hookHandler HookHandler) (ok bool, res HookResponse, err error) {
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
