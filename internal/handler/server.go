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
	"github.com/go-chi/httprate"
	"go.opencensus.io/plugin/ochttp"
)

var log = newStdLogger()

func NewServer(config *Config) (*http.Server, error) {
	handler := Handler{
		Config: config,
	}

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Group(func(r chi.Router) {
		r.Use(httprate.LimitByRealIP(1, 3*time.Second))
		r.Use(middleware.ThrottleBacklog(
			config.ThrottleLimit,
			config.ThrottleLimit+config.ThrottleLimit,
			time.Second*60,
		))

		r.Get("/run", handler.RunFilesHandler) // To avoid file-server handling
		r.Post("/run", handler.RunFilesHandler)

		r.Get("/run-script", handler.RunScriptHandler) // To avoid file-server handling
		r.Post("/run-script", handler.RunScriptHandler)

		r.Group(func(r chi.Router) {
			r.Get("/actions", handler.ActionListHandler)
		})
	})

	filesDir := http.Dir("/static")
	FileServer(router, "/", filesDir)

	return &http.Server{
		Addr:              ":" + config.Port,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           &ochttp.Handler{Handler: router},
	}, nil
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
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
