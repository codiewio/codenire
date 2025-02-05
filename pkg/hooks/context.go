package hooks

import (
	"context"
	"net/http"
	"time"
)

// HttpContext is wrapper around context.Context that also carries the
// corresponding HTTP request and response writer, as well as an
// optional body reader
// nolint
type HttpContext struct {
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

// newContext constructs a new HttpContext for the given request. This should only be done once
// per request and the context should be stored in the request, so it can be fetched with getContext.
func newContext(w http.ResponseWriter, r *http.Request, graceTimeout time.Duration) *HttpContext {
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
	delayedCtx := newDelayedContext(cancellableCtx, graceTimeout)

	controller := http.NewResponseController(w)
	ctx := &HttpContext{
		Context: delayedCtx,
		res:     w,
		resC:    controller,
		req:     r,
		cancel:  cancelHandling,
	}

	go func() {
		<-cancellableCtx.Done()
	}()

	return ctx
}

// getContext tries to retrieve a HttpContext from the request or constructs a new one.
func GetContext(w http.ResponseWriter, r *http.Request, graceTimeout time.Duration) *HttpContext {
	c, ok := r.Context().(*HttpContext)
	if !ok {
		c = newContext(w, r, graceTimeout)
	}

	return c
}

func (c HttpContext) Value(key any) any {
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
