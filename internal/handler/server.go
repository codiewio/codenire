// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"context"
	"fmt"
	"github.com/go-chi/httprate"
	"io"
	"net/http"
	"time"

	"github.com/codiewio/codenire/internal/client"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
	"go.opencensus.io/plugin/ochttp"
)

var log = newStdLogger()

var JWTAuth *jwtauth.JWTAuth

func NewServer(config *Config) (*http.Server, error) {
	handler := NewHandler(config)

	JWTAuth = jwtauth.New("HS256", []byte(config.JWTSecretKey), nil)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.Cors.AllowOrigins,
		AllowedMethods:   config.Cors.AllowMethods,
		AllowedHeaders:   config.Cors.AllowHeaders,
		ExposedHeaders:   config.Cors.ExposeHeaders,
		AllowCredentials: config.Cors.AllowCredentials,
		MaxAge:           config.Cors.MaxAge,
	}))

	router.Get("/", rootHandler)

	router.Group(func(r chi.Router) {
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

			in.Group(func(tr chi.Router) {
				r.Use(httprate.LimitByRealIP(1, 3*time.Second))
				tr.Use(middleware.ThrottleBacklog(
					config.ThrottleLimit,
					config.ThrottleLimit+config.ThrottleLimit,
					time.Second*60,
				))

				tr.Get("/run", handler.RunFilesHandler) // To avoid file-server handling
				tr.Post("/run", handler.RunFilesHandler)
			})

			//in.Get("/session/start", handler.SessionConnectHandler)
			in.Post("/session/connect", handler.SessionConnectHandler)

		})

		r.Group(func(r chi.Router) {
			r.Get("/actions", handler.ActionListHandler)
		})
	})

	router.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			handler.Config.BackendURL+"/metrics",
			nil,
		)

		if err != nil {
			http.Error(w, "sandbox client metrics request error", http.StatusInternalServerError)
			return
		}

		resp, err := client.SandboxBackendClient().Do(req)
		if err != nil {
			http.Error(w, "Failed to fetch metrics from other service", http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			http.Error(w, "Failed to write metrics to response", http.StatusInternalServerError)
			return
		}
	})

	return &http.Server{
		Addr:              ":" + config.Port,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           &ochttp.Handler{Handler: router},
	}, nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, _ = io.WriteString(w, "Hi from playground\n")
}
