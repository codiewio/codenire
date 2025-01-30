package handler

import (
	"context"

	api "github.com/codiewio/codenire/api/gen"
)

type HookEvent struct {
	Context           context.Context
	SubmissionRequest api.SubmissionRequest
	HTTPRequest       HTTPRequest
}

func newHookEvent(c *httpContext, sr api.SubmissionRequest) HookEvent {
	return HookEvent{
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
