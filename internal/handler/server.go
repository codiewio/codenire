// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
	"go.opencensus.io/plugin/ochttp"
)

var log = newStdLogger()

var JWTAuth *jwtauth.JWTAuth

func NewServer(config *Config) (*http.Server, error) {
	handler := Handler{
		Config: config,
	}

	JWTAuth = jwtauth.New("HS256", []byte(config.JWTSecretKey), nil)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/", rootHandler)

	router.Group(func(r chi.Router) {
		r.Use(httprate.LimitByRealIP(1, 3*time.Second))
		r.Use(middleware.ThrottleBacklog(
			config.ThrottleLimit,
			config.ThrottleLimit+config.ThrottleLimit,
			time.Second*60,
		))

		r.Group(func(in chi.Router) {
			if config.JWTSecretKey != "" {

				// For debugging/example purposes, we generate and print in `--dev` mode
				// a sample jwt token with claims `user_id:123` here:
				if config.Dev {
					_, tokenString, _ := JWTAuth.Encode(map[string]interface{}{"user_id": 123})
					fmt.Printf("DEBUG: a sample jwt is %s\n\n", tokenString)
				}

				in.Use(jwtauth.Verifier(JWTAuth))
				in.Use(jwtauth.Authenticator(JWTAuth))

				log.Printf("Enabled JWT handling")
			}

			in.Get("/run", handler.RunFilesHandler) // To avoid file-server handling
			in.Post("/run", handler.RunFilesHandler)

			in.Get("/run-script", handler.RunScriptHandler) // To avoid file-server handling
			in.Post("/run-script", handler.RunScriptHandler)
		})

		r.Group(func(r chi.Router) {
			r.Get("/actions", handler.ActionListHandler)
		})
	})

	filesDir := http.Dir("/static")
	// FileServer(router, "/", filesDir)
	FileServer(router, "/blbllblblb", filesDir)

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

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, _ = io.WriteString(w, "Hi from playground\n")
}
