// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type server struct {
	mux   *http.ServeMux
	log   logger
	cache responseCache
	gotip bool // if set, server is using gotip

	// When the executable was last modified. Used for caching headers of compiled assets.
	modtime time.Time
}

func newServer(options ...func(s *server) error) (*server, error) {
	s := &server{mux: http.NewServeMux()}
	for _, o := range options {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	if s.log == nil {
		return nil, fmt.Errorf("must provide an option func that specifies a logger")
	}
	s.init()
	return s, nil
}

func (s *server) init() {

	s.mux.HandleFunc("/run", runHandler)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Forwarded-Proto") == "http" {
		r.URL.Scheme = "https"
		r.URL.Host = r.Host
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; preload")
	}
	s.mux.ServeHTTP(w, r)
}

func (s *server) writeJSONResponse(w http.ResponseWriter, resp interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		s.log.Errorf("error encoding response: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	if _, err := io.Copy(w, &buf); err != nil {
		s.log.Errorf("io.Copy(w, &buf): %v", err)
		return
	}
}
