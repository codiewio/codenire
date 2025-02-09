// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"net/http"
	"strings"
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

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.Throttle(15))

	filesDir := http.Dir("/static")
	FileServer(router, "/", filesDir)

	router.Group(func(r chi.Router) {
		r.Post("/run", handler.RunFilesHandler)
		r.Post("/run-script", handler.RunScriptHandler)
	})

	router.Group(func(r chi.Router) {
		r.Get("/actions", handler.ActionListHandler)
	})

	return &http.Server{
		Addr:              ":" + config.Port,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           &ochttp.Handler{Handler: router},
	}, nil
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
