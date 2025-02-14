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
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	api "github.com/codiewio/codenire/api/gen"
	"github.com/codiewio/codenire/internal/handler"
	"github.com/codiewio/codenire/internal/images"
	"github.com/codiewio/codenire/pkg/hooks"
	"github.com/codiewio/codenire/pkg/hooks/file"
	"github.com/codiewio/codenire/pkg/hooks/plugin"
)

var (
	backendURL        = flag.String("backend-url", "http://sandbox_dev", "URL for sandbox backend that runs Go binaries.")
	Port              = flag.String("port", "8081", "URL for sandbox backend that runs Go binaries.")
	PluginHookPath    = flag.String("hooks-plugins", "", "URL for sandbox backend that runs Go binaries.")
	FileHooksDir      = flag.String("hooks-dir", "", "Directory to search for available hooks scripts")
	ExternalTemplates = flag.String("external-templates", "", "Comma separated list of templates which will handled externally (plugin for example)")
	ThrottleLimit     = flag.Int("throttle-limit", 15, "currently processed requests at a time across all users")
	JWTSecretKey      = flag.String("jwt-secret-key", "", "secret key to enable authentication")
	dev               = flag.Bool("dev", false, "run in dev mode")
)

func main() {
	flag.Parse()
	log.Printf("Use backend URL on :%s ...", *backendURL)

	ShowVersion()

	if err := waitForSandbox(10, 3*time.Second); err != nil {
		log.Println(err)
		return
	}

	cfg := handler.Config{
		BackendURL:                       *backendURL,
		Port:                             *Port,
		PluginHookPath:                   *PluginHookPath,
		FileHooksDir:                     *FileHooksDir,
		GracefulRequestCompletionTimeout: 10 * time.Second,
		ShutdownTimeout:                  10 * time.Second,
		ThrottleLimit:                    *ThrottleLimit,
		JWTSecretKey:                     *JWTSecretKey,
		Dev:                              *dev,
	}

	hookHandler := getHookHandler(&cfg)
	if hookHandler != nil {
		err := hookHandler.Setup()

		parseExternalTemplates()

		if err != nil {
			log.Fatalf("unable to setup hooks for handler: %s", err)
		}

		cfg.PreRequestCallback = func(ev hooks.CodeHookEvent) (hooks.HookResponse, error) {
			return hooks.PreSandboxRequestCallback(ev, hookHandler)
		}
	}

	s, err := handler.NewServer(&cfg)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	shutdownComplete := setupSignalHandler(cfg.ShutdownTimeout, func() {
		plugin.CleanupPlugins()
	})

	{
		res, terr := images.PullImageConfigList(cfg.BackendURL)
		if terr != nil {
			panic("sandbox not ready yet")
		}
		images.ImageTemplateList = res
	}

	log.Printf("playground is running, port %s", cfg.Port)

	err = s.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		// ErrServerClosed means that http.Server.Shutdown was called due to an interruption signal.
		// We wait until the interruption procedure is complete or times out and then exit main.
		<-shutdownComplete
	} else {
		// Any other error is relayed to the user.
		log.Fatalf("unable to serve: %s", err)
	}
}

func parseExternalTemplates() {
	templates := strings.Split(*ExternalTemplates, ",")
	for _, t := range templates {
		// Fake config
		images.ExtendedTemplates = append(images.ExtendedTemplates, api.ImageConfig{
			Template: t,
			Provider: "external",
			Actions: map[string]api.ImageActionConfig{
				"default": {
					CompileCmd:   "",
					DefaultFiles: nil,
					Id:           "1",
					// Very important
					IsDefault:     true,
					Name:          "",
					RunCmd:        "",
					ScriptOptions: api.ImageConfigScriptOptions{},
				},
			},
		})
	}
}

func getHookHandler(config *handler.Config) hooks.HookHandler {
	if config.PluginHookPath != "" {
		return &plugin.PluginHook{
			Path: config.PluginHookPath,
		}
	}

	if config.FileHooksDir != "" {
		return &file.FileHook{
			Directory: config.FileHooksDir,
		}
	}

	return nil
}

func waitForSandbox(maxRetries int, interval time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		//nolint
		resp, err := http.Get(*backendURL + "/health")

		if err == nil && resp.StatusCode == http.StatusOK {
			fmt.Println("sandbox is healthy!")
			return nil
		}
		fmt.Printf("waiting for sandbox... (%d/%d)\n", i+1, maxRetries)
		time.Sleep(interval)
	}

	return fmt.Errorf("sandbox is not available after %d retries", maxRetries)
}

func setupSignalHandler(shutdownTimeout time.Duration, options ...func()) <-chan struct{} {
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

		_, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		for _, o := range options {
			o()
		}

		close(shutdownComplete)
	}()

	return shutdownComplete
}
