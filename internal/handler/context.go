package handler

import (
	"context"
	"net/http"
	"time"
)

// httpContext is wrapper around context.Context that also carries the
// corresponding HTTP request and response writer, as well as an
// optional body reader
type httpContext struct {
	context.Context

	// res and req are the native request and response instances
	res  http.ResponseWriter
	resC *http.ResponseController
	req  *http.Request

	// cancel allows a user to cancel the internal request context, causing
	// the request body to be closed.
	cancel context.CancelCauseFunc

	// log is the logger for this request. It gets extended with more properties as the
	// request progresses and is identified.
}

// newContext constructs a new httpContext for the given request. This should only be done once
// per request and the context should be stored in the request, so it can be fetched with getContext.
func (h *Handler) newContext(w http.ResponseWriter, r *http.Request) *httpContext {
	// requestCtx is the context from the native request instance. It gets cancelled
	// if the connection closes, the request is cancelled (HTTP/2), ServeHTTP returns
	// or the server's base context is cancelled.
	requestCtx := r.Context()
	// On top of requestCtx, we construct a context that we can cancel, for example when
	// the post-receive hook stops an upload or if another uploads requests a lock to be released.
	cancellableCtx, cancelHandling := context.WithCancelCause(requestCtx)
	// On top of cancellableCtx, we construct a new context which gets cancelled with a delay.
	// See CodeHookEvent.Context for more details, but the gist is that we want to give data stores
	// some more time to finish their buisness.
	delayedCtx := newDelayedContext(cancellableCtx, h.Config.GracefulRequestCompletionTimeout)

	ctx := &httpContext{
		Context: delayedCtx,
		res:     w,
		resC:    http.NewResponseController(w),
		req:     r,
		cancel:  cancelHandling,
	}

	go func() {
		<-cancellableCtx.Done()
	}()

	return ctx
}

// getContext tries to retrieve a httpContext from the request or constructs a new one.
func (h *Handler) getContext(w http.ResponseWriter, r *http.Request) *httpContext {
	c, ok := r.Context().(*httpContext)
	if !ok {
		c = h.newContext(w, r)
	}

	return c
}

func (c httpContext) Value(key any) any {
	// We overwrite the Value function to ensure that the values from the request
	// context are returned because c.Context does not contain any values.
	return c.req.Context().Value(key)
}

// newDelayedContext returns a context that is cancelled with a delay. If the parent context
// is done, the new context will also be cancelled but only after waiting the specified delay.
// Note: The parent context MUST be cancelled or otherwise this will leak resources. In the
// case of http.SubmissionRequest.Context, the net/http package ensures that the context is always cancelled.
func newDelayedContext(parent context.Context, delay time.Duration) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-parent.Done()
		<-time.After(delay)
		cancel()
	}()

	return ctx
}
