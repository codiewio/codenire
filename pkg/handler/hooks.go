package handler

import (
	"context"

	api "github.com/codenire/codenire/api/gen"
)

type HookEvent struct {
	Context context.Context
	Request api.SubmissionRequest
}

func newHookEvent(ctx context.Context, req api.SubmissionRequest) HookEvent {
	return HookEvent{
		Context: ctx,
		Request: req,
	}
}
