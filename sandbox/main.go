// -----------------------------------------------------------
// Copyright:
//
// 2024 The Codenire Authors. All rights reserved.
// Authors:
// 	- Maksim Fedorov mfedorov@codiew.io
//
// Licensed under the MIT License.
//
// This project based on Source of Original Copyright (below)

// -----------------------------------------------------------
// **** The Original Copyright:
//
// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The sandbox program is an HTTP server that receives untrusted
// linux/amd64 binaries in a POST request and then executes them in
// a gvisor sandbox using Docker, returning the output as a response
// to the POST.
//
// It's part of the Go playground (https://play.golang.org/).

// **** End of the Original Copyright
// -----------------------------------------------------------
package main

import (
	"context"
	"errors"
	"flag"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"sandbox/manager"

	"github.com/go-chi/chi/v5"
	"go.opencensus.io/plugin/ochttp"
)

var httpServer *http.Server
var codenireManager manager.ContainerManager

var dockerPath = "docker"

const graceTimeout = 5 * time.Second

func main() {
	if *dev {
		dockerPath = "/usr/local/bin/docker"
	}

	out, err := exec.Command(dockerPath, "version").CombinedOutput()
	if err != nil {
		log.Fatalf("failed to connect to docker: %v, %s", err, out)
	}
	log.Fatalf("failed to connect to docker: %v, %s, %s", err, out, dockerPath)

	flag.Parse()

	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}
	log.Printf("Go playground sandbox starting.")

	codenireManager = manager.NewCodenireManager(*dev, *replicContainerCnt)
	codenireManager.KillAll()

	readyContainer = make(chan *Container)
	runSem = make(chan struct{}, *numWorkers)

	done := make(chan struct{})
	go handleSignals(done)

	err = codenireManager.Boot()
	if err != nil {
		codenireManager.KillAll()
		panic("Can't boot server")
	}

	if *dev {
		log.Printf("Running in dev mode; container published to host at: http://localhost:8080/")
	} else {
		log.Printf("Listening on %s", *listenAddr)
	}

	h := chi.NewRouter()
	h.Use(middleware.Recoverer)

	h.Get("/", rootHandler)
	h.Post("/run", runHandler)

	h.Post("/images/start", startImageHandler)
	h.Post("/images/list", listImageHandler)
	h.Post("/images/register", registerImageHandler)

	httpServer = &http.Server{
		Addr:    *listenAddr,
		Handler: &ochttp.Handler{Handler: h},
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	log.Println("Application is running...")
	<-done
	log.Println("Shutdown complete.")
}

func handleSignals(done chan struct{}) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	s := <-c
	log.Printf("Received signal: %s", s)

	gracefulShutdown(done)
}

func gracefulShutdown(done chan struct{}) {
	log.Println("Starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), graceTimeout)
	defer cancel()

	go func() {
		defer close(done)

		codenireManager.KillAll()
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Shutdown timed out!")
		}
	}
}
