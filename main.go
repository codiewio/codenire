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
	"errors"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/codiewio/codenire/internal/handler"
	"github.com/codiewio/codenire/internal/images"
	"github.com/codiewio/codenire/pkg/hooks"
	"github.com/codiewio/codenire/pkg/hooks/file"
	"github.com/codiewio/codenire/pkg/hooks/plugin"
)

var (
	BackendURL     = flag.String("backend-url", "http://sandbox_dev", "URL for sandbox backend that runs Go binaries.")
	Port           = flag.String("port", "8081", "URL for sandbox backend that runs Go binaries.")
	PluginHookPath = flag.String("hooks-plugins", "", "URL for sandbox backend that runs Go binaries.")
	FileHooksDir   = flag.String("hooks-dir", "", "Directory to search for available hooks scripts")
)

func main() {
	flag.Parse()
	log.Printf("Use backend URL on :%s ...", *BackendURL)

	cfg := handler.Config{
		BackendURL:                       *BackendURL,
		Port:                             *Port,
		PluginHookPath:                   *PluginHookPath,
		FileHooksDir:                     *FileHooksDir,
		GracefulRequestCompletionTimeout: 10 * time.Second,
		ShutdownTimeout:                  10 * time.Second,
	}

	hookHandler := getHookHandler(&cfg)
	if hookHandler != nil {
		if err := hookHandler.Setup(); err != nil {
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
	port := cfg.Port

	shutdownComplete := s.SetupSignalHandler(func() {
		plugin.CleanupPlugins()
	})

	err = images.PullImageConfigList(cfg.BackendURL + "/images/list")
	if err != nil {
		panic("sandbox not ready yet")
	}

	log.Printf("listening on :%v ...", port)
	err = http.ListenAndServe(":"+port, s)

	log.Printf("[playground] application is running, port %s", port)

	if errors.Is(err, http.ErrServerClosed) {
		// ErrServerClosed means that http.Server.Shutdown was called due to an interruption signal.
		// We wait until the interruption procedure is complete or times out and then exit main.
		<-shutdownComplete
	} else {
		// Any other error is relayed to the user.
		log.Fatalf("[playground] unable to serve: %s", err)
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
