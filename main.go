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
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
)

var log = newStdLogger()

var (
	BackendURL = flag.String("backend-url", "http://127.0.0.1:8080/run", "URL for sandbox backend that runs Go binaries.")
)

func main() {
	flag.Parse()
	s, err := newServer(func(s *server) error {
		pid := projectID()

		if caddr := os.Getenv("MEMCACHED_ADDR"); caddr != "" {
			s.cache = newGobCache(caddr)
			log.Printf("App (project ID: %q) is caching results", pid)
		} else {
			s.cache = (*gobCache)(nil) // Use a no-op cache implementation.
			log.Printf("App (project ID: %q) is NOT caching results", pid)
		}
		s.log = log
		if gotip := os.Getenv("GOTIP"); gotip == "true" {
			s.gotip = true
		}
		execpath, _ := os.Executable()
		if execpath != "" {
			if fi, _ := os.Stat(execpath); fi != nil {
				s.modtime = fi.ModTime()
			}
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	//if *runtests {
	//	s.test()
	//	return
	//}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Get the backend dialer warmed up. This starts
	// RegionInstanceGroupDialer queries and health checks.
	go sandboxBackendClient()

	log.Printf("Listening on :%v ...", port)
	log.Fatalf("Error listening on :%v: %v", port, http.ListenAndServe(":"+port, s))
}

func projectID() string {
	id, err := metadata.ProjectID()
	if err != nil && os.Getenv("GAE_INSTANCE") != "" {
		log.Fatalf("Could not determine the project ID: %v", err)
	}
	return id
}
