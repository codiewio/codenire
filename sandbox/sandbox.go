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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	api "sandbox/api/gen"
	"sandbox/internal"
)

const (
	maxBinarySize         = 100 << 20
	startContainerTimeout = 100 * time.Second
	defaultRunTimeout     = 5 * time.Second
	defaultCompileTimeout = 30 * time.Second
	totalTimeout          = defaultRunTimeout + defaultCompileTimeout
	maxOutputSize         = 100 << 20
	memoryLimitBytes      = 100 << 20
)

var (
	errTooMuchOutput = errors.New("output too large")
	errRunTimeout    = errors.New("timeout running program")
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	io.WriteString(w, "Hi from sandbox\n")
}

func listImageHandler(w http.ResponseWriter, _ *http.Request) {
	res := api.ImageConfigList{}

	rows := codenireManager.ImageList()
	for _, row := range rows {
		res = append(res, api.ImageConfig{
			Name:        row.ImageConfig.Name,
			Description: row.Description,

			RunCmd:     row.RunCmd,
			CompileCmd: row.CompileCmd,

			Options: api.ImageConfigOption{
				CompileTTL: row.Options.CompileTTL,
				RunTTL:     row.Options.RunTTL,
			},
			ScriptOptions: api.ImageConfigScriptOptions{
				SourceFile: row.ScriptOptions.SourceFile,
			},
		})
	}

	body, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	// Bound the number of requests being processed at once.
	// (Before we slurp the binary into memory)
	select {
	case runSem <- struct{}{}:
	case <-r.Context().Done():
		return
	}
	defer func() { <-runSem }()

	var req api.SandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request 1")

		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "tmp_sandbox")
	if err != nil {
		log.Printf("Invalid request 2")
		http.Error(w, "create tmp dir failed", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	err = internal.Base64ToTar(req.Binary, tmpDir)
	if err != nil {
		log.Printf("Invalid request 3")
		sendRunError(w, "encode  files failed", http.StatusInternalServerError)
		return
	}

	cont, err := codenireManager.GetContainer(r.Context(), req.SandId)
	if err != nil {
		log.Printf("Invalid request container by ID %s", req.SandId)
		sendRunError(w, fmt.Sprintf("get container %s failed with %s", req.SandId, err.Error()), http.StatusInternalServerError)
		return
	}

	defer func() {
		err = codenireManager.KillContainer(cont.CId)
		if err != nil {
			sendRunError(w, fmt.Sprintf("kill contaier err: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}()

	// TODO:: Make Timeout?
	out, err := exec.Command(
		"docker",
		"cp",
		tmpDir+"/.",
		cont.CId+":"+cont.Image.Workdir,
	).CombinedOutput()

	if err != nil {
		log.Printf("Invalid request 7")
		sendRunError(w, fmt.Sprintf("failed to connect to docker: %v, %s", err, out), http.StatusInternalServerError)
		return
	}

	timeoutCtx := registerTimeout(r.Context(), totalTimeout)
	res := &api.SandboxResponse{}

	var stdout, stderr bytes.Buffer
	var exitCode int

	if cont.Image.CompileCmd != "" {
		compileCtx := registerTimeout(r.Context(), totalTimeout)
		{
			callCmd := fmt.Sprintf("cd %s && %s", cont.Image.Workdir, cont.Image.CompileCmd)

			err := execCommandInsideContainer(compileCtx, &stderr, &stdout, *cont, callCmd)
			if err != nil {
				if errors.Is(compileCtx.Err(), context.DeadlineExceeded) {
					sendRunError(w, "timeout on compilation", http.StatusRequestTimeout)
					return
				}

				sendRunError(w, fmt.Sprintf("some compilation error: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	runTimeoutCtx := registerTimeout(timeoutCtx, defaultRunTimeout)
	{
		runCmd := fmt.Sprintf("cd %s && %s", cont.Image.Workdir, cont.Image.RunCmd)

		err := execCommandInsideContainer(runTimeoutCtx, &stderr, &stdout, *cont, runCmd)
		if err != nil {
			if errors.Is(runTimeoutCtx.Err(), context.DeadlineExceeded) {
				sendRunError(w, "timeout on running", http.StatusRequestTimeout)
				return
			}

			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				exitCode = exitError.ExitCode()
			} else {
				sendRunError(w, fmt.Sprintf("some compilation error: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	res.ExitCode = exitCode
	res.Stderr = stderr.Bytes()
	res.Stdout = stdout.Bytes()

	sendRunResponse(w, res)
}

func execCommandInsideContainer(ctx context.Context, stderr *bytes.Buffer, stdout *bytes.Buffer, container StartedContainer, execCmd string) error {
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"exec",
		container.CId,
		"sh", "-c",
		execCmd,
	)

	cmd.Stderr = stderr
	cmd.Stdout = stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
func registerTimeout(ctx context.Context, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(ctx, timeout)

	go func() {
		<-ctx.Done()
		cancel()
		return
	}()

	return ctx
}

func sendRunResponse(w http.ResponseWriter, r *api.SandboxResponse) {
	jres, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(jres)))
	w.Write(jres)
}

func sendRunError(w http.ResponseWriter, err string, code int) {
	res := &api.SandboxResponse{}
	res.Stderr = []byte(err)

	sendRunResponse(w, res)
}
