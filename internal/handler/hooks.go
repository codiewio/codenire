package handler

import (
	"context"
	"github.com/codiewio/codenire/pkg/hooks"

	api "github.com/codiewio/codenire/api/gen"
)

type CodeHookEvent struct {
	Context           context.Context
	SubmissionRequest api.SubmissionRequest
	HTTPRequest       hooks.HTTPRequest
}

func newCodeHookEvent(c *httpContext, sr api.SubmissionRequest) CodeHookEvent {
	return CodeHookEvent{
		Context:           c,
		SubmissionRequest: sr,
		HTTPRequest: hooks.HTTPRequest{
			Method:     c.req.Method,
			URI:        c.req.RequestURI,
			RemoteAddr: c.req.RemoteAddr,
			Header:     c.req.Header,
		},
	}
}
