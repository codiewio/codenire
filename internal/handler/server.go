// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var log = newStdLogger()

type Server struct {
	mux     *http.ServeMux
	log     logger
	handler Handler
}

func NewServer(config *Config, options ...func(s *Server) error) (*Server, error) {
	s := &Server{
		mux: http.NewServeMux(),
		handler: Handler{
			Config: config,
		},
		log: log,
	}

	for _, o := range options {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	if s.log == nil {
		return nil, fmt.Errorf("must provide an option func that specifies a logger")
	}

	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		_, _ = io.WriteString(w, "Hi from playground\n")
	})

	s.mux.HandleFunc("/run", s.handler.RunFilesHandler)
	s.mux.HandleFunc("/run-script", s.handler.RunScriptHandler)

	s.mux.HandleFunc("/actions", s.handler.ActionListHandler)
	// s.mux.HandleFunc("/action/add", s.handler.ActionAddHandler)

	// s.mux.HandleFunc("/template/list", s.handler.TemplateListHandler)
	// s.mux.HandleFunc("/template/add", s.handler.TemplateAddHandler)

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
	s.mux.ServeHTTP(w, r)
}

func (s *Server) SetupSignalHandler(options ...func()) <-chan struct{} {
	shutdownComplete := make(chan struct{})

	// We read up to two signals, so use a capacity of 2 here to not miss any signal
	c := make(chan os.Signal, 2)

	// os.Interrupt is mapped to SIGINT on Unix and to the termination instructions on Windows.
	// On Unix we also listen to SIGTERM.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		// First interrupt signal
		<-c
		log.Printf("Received interrupt signal. Shutting down codenire...")

		// Wait for second interrupt signal, while also shutting down the existing server
		go func() {
			<-c
			log.Printf("Received second interrupt signal. Exiting immediately!")
			os.Exit(1)
		}()

		_, cancel := context.WithTimeout(context.Background(), s.handler.Config.ShutdownTimeout)
		defer cancel()

		for _, o := range options {
			o()
		}

		close(shutdownComplete)
	}()

	return shutdownComplete
}
