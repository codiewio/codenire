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
	"flag"
	"github.com/codenire/codenire/pkg/hooks/file"
	"log"
	"net/http"

	"github.com/codenire/codenire/pkg/handler"
	"github.com/codenire/codenire/pkg/hooks"
	"github.com/codenire/codenire/pkg/hooks/plugin"
)

var (
	BackendURL     = flag.String("backend-url", "http://sandbox_dev/run", "URL for sandbox backend that runs Go binaries.")
	Port           = flag.String("port", "8081", "URL for sandbox backend that runs Go binaries.")
	PluginHookPath = flag.String("hooks-plugins", "", "URL for sandbox backend that runs Go binaries.")
	FileHooksDir   = flag.String("hooks-dir", "", "Directory to search for available hooks scripts")
)

func main() {
	flag.Parse()
	log.Printf("Use backend URL on :%s ...", *BackendURL)

	cfg := handler.Config{
		BackendURL:     *BackendURL,
		Port:           *Port,
		PluginHookPath: *PluginHookPath,
		FileHooksDir:   *FileHooksDir,
	}

	hookHandler := getHookHandler(&cfg)
	if hookHandler != nil {
		if err := hookHandler.Setup(); err != nil {
			log.Fatalf("unable to setup hooks for handler: %s", err)
		}

		cfg.PreRequestCallback = func(ev handler.HookEvent) (handler.HTTPResponse, error) {
			return hooks.PreSandboxRequestCallback(ev, hookHandler)
		}
	}

	s, err := handler.NewServer(&cfg)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	port := cfg.Port

	log.Printf("Listening on :%v ...", port)
	log.Fatalf("Error listening on :%v: %v", port, http.ListenAndServe(":"+port, s))
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
