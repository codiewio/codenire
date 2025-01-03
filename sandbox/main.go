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
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"sandbox/manager"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opencensus.io/plugin/ochttp"
)

var httpServer *http.Server
var codenireManager manager.ContainerManager

var (
	listenAddr          = flag.String("listenAddr", ":80", "HTTP server listen address")
	dev                 = flag.Bool("dev", false, "run in dev mode")
	numWorkers          = flag.Int("workers", runtime.NumCPU(), "number of parallel gvisor containers to pre-spin up & let run concurrently")
	replicaContainerCnt = flag.Int("replicaContainerCnt", 1, "number of parallel containers")
	dockerFilesPath     = flag.String("dockerFilesPath", "", "configs paths")

	runSem       chan struct{}
	graceTimeout = 5 * time.Second
)

func main() {
	flag.Parse()

	out, err := exec.Command("docker", "version").CombinedOutput()
	if err != nil {
		log.Fatalf("failed to connect to docker: %v, %s", err, out)
	}

	log.Printf("Go playground sandbox starting.")

	codenireManager = manager.NewCodenireManager(*dev, *replicaContainerCnt, *dockerFilesPath)
	codenireManager.KillAll()

	readyContainer = make(chan *Container)
	runSem = make(chan struct{}, *numWorkers)
	log.Printf("Workers count: %d", *numWorkers)

	done := make(chan struct{})
	go handleSignals(done)

	log.Printf("Started boot")
	err = codenireManager.Boot()
	if err != nil {
		codenireManager.KillAll()
		panic("Can't boot server")
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

	log.Printf("Application is running, port %s", *listenAddr)
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
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Println("Shutdown timed out!")
		}
	}
}
