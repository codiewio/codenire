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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opencensus.io/plugin/ochttp"
	_ "go.uber.org/automaxprocs"

	"sandbox/internal"
)

var codenireManager ContainerOrchestrator

var (
	listenAddr          = flag.String("port", "80", "HTTP server listen address")
	dev                 = flag.Bool("dev", false, "run in dev mode")
	numWorkers          = flag.Int("workers", runtime.NumCPU(), "number of parallel gvisor containers to pre-spin up & let run concurrently")
	replicaContainerCnt = flag.Int("replicaContainerCnt", 1, "number of parallel containers for every uniq image")
	dockerFilesPath     = flag.String("dockerFilesPath", "", "directory path with templates")

	isolated                = flag.Bool("isolated", false, "use gVisor isolation for compile code")
	isolatedNetwork         = flag.String("isolatedNetwork", "none", "isolated network")
	isolatedGateway         = flag.String("isolatedGateway", "http://package_dev:3128", "proxy which pass traffik from internal newtwork")
	gvisorRuntime           = "runsc"
	isolatedPostgresDSN     = flag.String("isolatedPostgresDSN", "", "isolated postgres DB instance")
	isolatedPostgresNetwork = flag.String("isolatedPostgresNetwork", "", "isolated postgres network")

	s3DockerfilesEndpoint = flag.String("s3DockerfilesEndpoint", "", "s3 endpoint with templates")
	s3DockerfilesBucket   = flag.String("s3DockerfilesBucket", "", "s3 bucket with templates")
	s3DockerfilesPrefix   = flag.String("s3DockerfilesPrefix", "", "prefix aka directory with templates")

	runSem       chan struct{}
	graceTimeout = 15 * time.Second
)

func main() {
	flag.Parse()

	checkIsolation()

	templatePath, err := templatesDataPrepare()
	if templatePath != nil {
		defer os.RemoveAll(*templatePath)
	}
	if err != nil {
		panic(fmt.Errorf("failed handle templates dir: %w", err))
	}

	out, err := exec.Command("docker", "version").CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("failed to connect to docker: %s, err: %w", out, err))
	}

	log.Printf("Go playground sandbox starting...")
	// storage := &Storage{}
	// codenireManager = NewCodenireOrchestrator(storage)
	//Not possible to use storage in this case as it is not implemented
	codenireManager.KillAll()

	runSem = make(chan struct{}, *numWorkers)
	log.Printf("Workers count: %d", *numWorkers)

	done := make(chan struct{})
	go handleSignals(done)

	log.Printf("Started boot")

	h := chi.NewRouter()
	h.Use(middleware.Recoverer)

	h.Get("/", rootHandler)
	h.Get("/health", healthHandler)

	h.Post("/run", runHandler)
	h.Get("/templates", listTemplatesHandler)
	h.Get("/templates/{id}", getTemplateByIDHandler)
	h.Post("/templates", AddTemplateHandler)
	h.Post("/templates/{id}", runTemplateHandler)
	h.Delete("/templates/{id}", deleteTemplateHandler)
	h.Put("/templates/{id}", updateTemplateHandler)

	httpServer := &http.Server{
		Addr:              ":" + *listenAddr,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           &ochttp.Handler{Handler: h},

		// TODO:: check connections
		// ConnState: func(_ net.Conn, cs http.ConnState) {
		//	switch cs {
		//	case http.StateNew:
		//		MetricsOpenConnections.Inc()
		//	case http.StateClosed, http.StateHijacked:
		//		MetricsOpenConnections.Dec()
		//	}
		// },
	}

	err = codenireManager.Prepare()
	if err != nil {
		log.Printf("failed to prepare templates: %v", err)
		done <- struct{}{}
	}

	go func() {
		err = codenireManager.Boot()
		if err != nil {
			log.Printf("failed to boot codenire manager: %v", err)
			done <- struct{}{}
		}
	}()

	go func() {
		if sErr := httpServer.ListenAndServe(); sErr != nil && !errors.Is(sErr, http.ErrServerClosed) {
			panic(fmt.Errorf("server failed: %w", sErr))
		}
	}()

	log.Printf("sandbox is running, port %s", *listenAddr)
	<-done
	log.Println("shutdown complete.")
}

func templatesDataPrepare() (*string, error) {
	if s3DockerfilesBucket != nil && *s3DockerfilesBucket != "" {
		if s3DockerfilesEndpoint == nil || s3DockerfilesPrefix == nil {
			return nil, errors.New("s3 endpoint and prefix are required")
		}

		region := os.Getenv("AWS_REGION")
		if region == "" {
			return nil, errors.New("AWS_REGION environment variable not set")
		}

		tmpDir, err := os.MkdirTemp("", "box")
		if err != nil {
			log.Println("Err", err)
			return nil, err
		}

		path, err := internal.DownloadTemplates(
			tmpDir,
			*s3DockerfilesEndpoint,
			*s3DockerfilesBucket,
			*s3DockerfilesPrefix,
			region,
		)
		if err != nil {
			log.Println("Err", err)
			return &tmpDir, err
		}

		dockerFilesPath = path
	}

	return dockerFilesPath, nil
}

func checkIsolation() {
	if !*isolated {
		return
	}

	out, err := exec.
		Command(
			"docker",
			"info",
			"--format",
			"{{range $key, $value := .Runtimes}}{{$key}} {{end}}",
		).
		CombinedOutput()

	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	runtimes := strings.Split(string(out), " ")

	log.Println("Available Runtimes:")
	{
		for _, rt := range runtimes {
			log.Println(rt)
		}
	}

	{
		for _, rt := range runtimes {
			if rt == gvisorRuntime {
				return
			}
		}
	}

	log.Println("\n\n-----------------------------------------")
	log.Println("! runsc runtime not available in the system.")
	log.Println("\n\n-----------------------------------------")

	os.Exit(1)
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

	//nolint
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Println("Shutdown timed out!")
		}
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
