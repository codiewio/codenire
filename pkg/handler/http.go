package handler

import (
	"net/http"
	"strconv"
)

type HTTPHeader map[string]string

type HTTPResponse struct {
	StatusCode int
	Body       string
	Header     HTTPHeader
	Prevent    bool
}

func (resp HTTPResponse) WriteTo(w http.ResponseWriter) {
	headers := w.Header()
	for key, value := range resp.Header {
		headers.Set(key, value)
	}

	if len(resp.Body) > 0 {
		headers.Set("Content-Length", strconv.Itoa(len(resp.Body)))
	}

	w.WriteHeader(resp.StatusCode)

	if len(resp.Body) > 0 {
		w.Write([]byte(resp.Body))
	}
}

// MergeWith returns a copy of resp1, where non-default values from resp2 overwrite
// values from resp1.
func (resp1 HTTPResponse) MergeWith(resp2 HTTPResponse) HTTPResponse {
	// Clone the response 1 and use it as a basis
	newResp := resp1

	// Take the status code and body from response 2 to
	// overwrite values from response 1.
	if resp2.StatusCode != 0 {
		newResp.StatusCode = resp2.StatusCode
	}

	if len(resp2.Body) > 0 {
		newResp.Body = resp2.Body
	}

	// For the headers, me must make a new map to avoid writing
	// into the header map from response 1.
	newResp.Header = make(HTTPHeader, len(resp1.Header)+len(resp2.Header))

	for key, value := range resp1.Header {
		newResp.Header[key] = value
	}

	for key, value := range resp2.Header {
		newResp.Header[key] = value
	}

	return newResp
}
