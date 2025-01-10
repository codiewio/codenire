// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var log = newStdLogger()

type Server struct {
	Router  *chi.Mux
	Handler Handler
	log     logger
}

func NewServer(config *Config, options ...func(s *Server) error) (*Server, error) {
	mux := chi.NewRouter()

	h := Handler{
		Config: config,
	}

	s := &Server{
		Router:  mux,
		Handler: h,
		log:     log,
	}

	for _, o := range options {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	if s.log == nil {
		return nil, fmt.Errorf("must provide an option func that specifies a logger")
	}

	mux.Use(middleware.Recoverer)

	mux.Group(func(r chi.Router) {
		r.
			With(func(handler http.Handler) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {

				}

				return http.HandlerFunc(fn)
			}).
			Post("/run", h.RunHandler)
	})

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Forwarded-Proto") == "http" {
		r.URL.Scheme = "https"
		r.URL.Host = r.Host
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; preload")
	}
	s.Router.ServeHTTP(w, r)
}
