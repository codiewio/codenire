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
	"bytes"
	"cloud.google.com/go/compute/metadata"
	"encoding/json"
	"errors"
	"fmt"
	api "github.com/codenire/codenire/api/gen"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func runHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req api.SubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "box")
	if err != nil {
		return
	}
	defer os.RemoveAll(tmpDir)

	err = copyFilesToTmpDir(tmpDir, req.Files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	b, err := tarToBase64(tmpDir)
	if err != nil {
		http.Error(w, "fail on create tar files: "+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(
		api.SandboxRequest{
			Args:   req.Args,
			SandId: req.TemplateId,
			Binary: b,
			IsExec: false,
		},
	)
	if err != nil {
		fmt.Println("Ошибка сериализации:", err)
		return
	}

	sreq, err := http.NewRequestWithContext(ctx, "POST", *BackendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}

	sreq.Header.Add("Idempotency-Key", "1")

	sreq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(jsonData)), nil }
	resp, err := sandboxBackendClient().Do(sreq)
	if err != nil {
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected response from backend: %v", resp.Status)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	var execRes api.SandboxResponse

	if err = json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		log.Printf("JSON decode error from backend: %v", err)
		http.Error(w, "error parsing JSON from backend", http.StatusInternalServerError)
		return
	}

	//if *execRes.Error != "" {
	//	writeJSONResponse(w, &api.SubmissionResponse{Errors: execRes.Error}, http.StatusOK)
	//}

	rec := new(Recorder)
	rec.Stdout().Write(execRes.Stdout)
	rec.Stderr().Write(execRes.Stderr)
	events, err := rec.Events()
	if err != nil {
		log.Printf("error decoding events: %v", err)
		http.Error(w, "error parsing JSON from backend", http.StatusInternalServerError)
		return
	}

	apiRes := &api.SubmissionResponse{
		Events: events,
		Meta:   nil,
		Time:   nil,
	}

	writeJSONResponse(w, apiRes, http.StatusOK)
}

func copyFilesToTmpDir(tmpDir string, files map[string]string) error {
	for f, src := range files {
		if !strings.Contains(f, "/") {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, f, src, parser.PackageClauseOnly)
			if err == nil && f.Name.Name != "main" {
				return errors.New(fmt.Sprintf("package name must be main", err.Error()))
			}
		}

		in := filepath.Join(tmpDir, f)
		if strings.Contains(f, "/") {
			if err := os.MkdirAll(filepath.Dir(in), 0755); err != nil {
				return err
			}
		}
		if err := os.WriteFile(in, []byte(src), 0644); err != nil {
			return errors.New(fmt.Sprintf("error creating temp file %q: %v", in, err))
		}
	}

	return nil
}

var sandboxBackendOnce struct {
	sync.Once
	c *http.Client
}

func sandboxBackendClient() *http.Client {
	sandboxBackendOnce.Do(initSandboxBackendClient)
	return sandboxBackendOnce.c
}

// initSandboxBackendClient runs from a sync.Once and initializes
// sandboxBackendOnce.c with the *http.Client we'll use to contact the
// sandbox execution backend.
func initSandboxBackendClient() {
	id, _ := metadata.ProjectID()
	switch id {
	default:
		sandboxBackendOnce.c = http.DefaultClient
	}
}

func writeJSONResponse(w http.ResponseWriter, resp interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		log.Errorf("error encoding response: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	if _, err := io.Copy(w, &buf); err != nil {
		log.Errorf("io.Copy(w, &buf): %v", err)
		return
	}
}
