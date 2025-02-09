// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opencensus.io/plugin/ochttp"
)

var log = newStdLogger()

func NewServer(config *Config) (*http.Server, error) {
	handler := Handler{
		Config: config,
	}

	h := chi.NewRouter()
	h.Use(middleware.Recoverer)
	h.Use(middleware.Throttle(15))

	h.Get("/", rootHandler)

	h.Group(func(r chi.Router) {
		r.Post("/run", handler.RunFilesHandler)
		r.Post("/run-script", handler.RunScriptHandler)
	})

	h.Group(func(r chi.Router) {
		r.Get("/actions", handler.ActionListHandler)
	})

	return &http.Server{
		Addr:              ":" + config.Port,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           &ochttp.Handler{Handler: h},
	}, nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, _ = io.WriteString(w, "Hi from playground\n")
}
