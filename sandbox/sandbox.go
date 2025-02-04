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
	"path/filepath"
	"regexp"
	"strings"
	"time"

	contract "sandbox/api/gen"
	"sandbox/internal"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, _ = io.WriteString(w, "Hi from sandbox\n")
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("Get sandbox req:", time.Now().UnixMilli())

	var err error

	var req contract.SandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "tmp_sandbox")
	if err != nil {
		http.Error(w, "create tmp dir failed", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	stdinFile, err := internal.Base64ToTar(req.Binary, tmpDir, req.Stdin)
	if err != nil {
		sendRunError(w, "encode  files failed")
		return
	}

	//fmt.Println("Handled sandbox req:", time.Now().UnixMilli())

	cont, err := codenireManager.GetContainer(r.Context(), req.SandId)
	if err != nil {
		sendRunError(w, fmt.Sprintf("get container %s failed with %s", req.SandId, err.Error()))
		return
	}

	//fmt.Printf("Got container %s: %d\n", cont.Image.Id, time.Now().UnixMilli())
	defer func() {
		err = codenireManager.KillContainer(*cont)
		if err != nil {
			sendRunError(w, fmt.Sprintf("kill contaier err: %s", err.Error()))
			return
		}
	}()

	// Bound the number of requests being processed at once.
	// (Before we slurp the binary into memory)
	select {
	case runSem <- struct{}{}:
	case <-r.Context().Done():
		return
	}
	defer func() { <-runSem }()

	action, exists := cont.Image.Actions[req.Action]
	if !exists {
		sendRunError(w, fmt.Sprintf("action %s not found with template %s", req.Action, req.SandId))
	}

	// TODO:: Make Timeout?
	out, err := exec.Command(
		"docker",
		"cp",
		tmpDir+"/.",
		cont.CId+":"+cont.Image.Workdir,
	).CombinedOutput()

	if err != nil {
		log.Printf("Invalid request 7")
		sendRunError(w, fmt.Sprintf("failed to connect to docker: %v, %s", err, out))
		return
	}

	totalTimeout := time.Duration(*cont.Image.ContainerOptions.CompileTTL+*cont.Image.ContainerOptions.RunTTL) * time.Second

	timeoutCtx := registerCmdTimeout(r.Context(), totalTimeout)
	res := &contract.SandboxResponse{}

	var stdout, stderr bytes.Buffer

	if action.CompileCmd != "" {
		compileCtx := registerCmdTimeout(r.Context(), totalTimeout)
		{
			err := execContainerShell(compileCtx, &stderr, &stdout, *cont, action.CompileCmd, req.Args, nil, cont.Image)
			if err != nil {
				if errors.Is(compileCtx.Err(), context.DeadlineExceeded) {
					sendRunError(w, "timeout compilation")
					return
				}

				flushStdWithErr(res, stderr, stdout)
				sendResponse(w, res)
				return
			}
		}
	}

	runTtl := time.Duration(*cont.Image.ContainerOptions.RunTTL) * time.Second
	runTimeoutCtx := registerCmdTimeout(timeoutCtx, runTtl)
	{
		err := execContainerShell(runTimeoutCtx, &stderr, &stdout, *cont, action.RunCmd, req.Args, stdinFile, cont.Image)
		if err != nil {
			if errors.Is(runTimeoutCtx.Err(), context.DeadlineExceeded) {
				sendRunError(w, "timeout execute")
				return
			}

			flushStdWithErr(res, stderr, stdout)
			sendResponse(w, res)
			return
		}
	}

	flushStd(res, stderr, stdout)
	sendResponse(w, res)
}

func sendResponse(w http.ResponseWriter, res *contract.SandboxResponse) {
	jres, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(jres)))
	_, _ = w.Write(jres)
}

func sendRunError(w http.ResponseWriter, err string) {
	res := &contract.SandboxResponse{}
	res.Stderr = []byte(err)

	sendRunResponse(w, res)
}

func flushStd(res *contract.SandboxResponse, stderr bytes.Buffer, stdout bytes.Buffer) {
	res.Stderr = stderr.Bytes()
	res.Stdout = stdout.Bytes()
}

func flushStdWithErr(res *contract.SandboxResponse, stderr bytes.Buffer, stdout bytes.Buffer) {
	mergedOutput := append(stdout.Bytes(), '\n')
	mergedOutput = append(mergedOutput, stderr.Bytes()...)
	res.Stderr = mergedOutput
	res.Stdout = nil
}

func execContainerShell(ctx context.Context, stderr *bytes.Buffer, stdout *bytes.Buffer, container StartedContainer, runCmd, args string, stdinFileName *string, cfg BuiltImage) error {
	//fmt.Printf("Call command %s: %d\n", runCmd, time.Now().UnixMilli())

	sh := fmt.Sprintf("cd %s && %s", cfg.Workdir, runCmd)
	placeholders := map[string]string{
		"ARGS": args,
	}
	if stdinFileName != nil {
		placeholders["STDIN"] = *stdinFileName
	}
	sh = replacePlaceholders(sh, placeholders)

	cmd := exec.CommandContext(
		ctx,
		"docker",
		"exec",
		container.CId,
		"sh", "-c",
		sh,
	)

	cmd.Stderr = stderr
	cmd.Stdout = stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	//fmt.Printf("Finish command %s: %d\n", runCmd, time.Now().UnixMilli())

	return nil
}

func registerCmdTimeout(ctx context.Context, timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(ctx, timeout)

	go func() {
		<-ctx.Done()
		cancel()
	}()

	return ctx
}

func sendRunResponse(w http.ResponseWriter, r *contract.SandboxResponse) {
	jres, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(jres)))
	_, _ = w.Write(jres)
}
func replacePlaceholders(input string, values map[string]string) string {
	re := regexp.MustCompile(`\{\s*([A-Z0-9_]+)\s*\}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSpace(match[1 : len(match)-1])
		if val, exists := values[key]; exists {
			return val
		}
		return match
	})
}

func listTemplatesHandler(w http.ResponseWriter, _ *http.Request) {
	body, err := json.MarshalIndent(codenireManager.ImageList(), "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func listFiles(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fmt.Println("FFFFFFFFF:", path)
		}
		return nil
	})
}
