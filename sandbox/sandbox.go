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
	"regexp"
	"strings"
	"time"

	contract "sandbox/api/gen"
	"sandbox/internal"

	chi "github.com/go-chi/chi/v5"
)

const (
	CompileCmd = "CompileCmd"
	RunCmd     = "RunCmd"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, _ = io.WriteString(w, "Hi from sandbox\n")
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	var req contract.SandboxRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "tmp_sandbox")
	if err != nil {
		http.Error(w, "createDB tmp dir failed", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	stdinFile, err := internal.SaveRequestFiles(req, tmpDir)
	if err != nil {
		sendRunError(w, "encode  files failed", nil)
		return
	}

	cont, err := codenireManager.GetContainer(r.Context(), req.SandId)
	if err != nil {
		sendRunError(w, fmt.Sprintf("get container %s failed with %s", req.SandId, err.Error()), nil)
		return
	}

	defer func() {
		err = codenireManager.KillContainer(*cont)
		if err != nil {
			sendRunError(w, fmt.Sprintf("kill contaier err: %s", err.Error()), nil)
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
		sendRunError(w, fmt.Sprintf("action %s not found with template %s", req.Action, req.SandId), nil)
	}

	//nolint
	out, err := exec.Command(
		"docker",
		"cp",
		tmpDir+"/.",
		cont.CId+":"+cont.Image.Workdir,
	).CombinedOutput()

	if err != nil {
		log.Printf("Invalid request 7")
		sendRunError(w, fmt.Sprintf("failed to connect to docker: %v, %s", err, out), nil)
		return
	}

	totalTimeout := time.Duration(*cont.Image.ContainerOptions.CompileTTL+*cont.Image.ContainerOptions.RunTTL) * time.Second
	timeoutCtx := registerCmdTimeout(r.Context(), totalTimeout)

	res := &contract.SandboxResponse{}
	res.RunEnvironment.ActionName = action.Name
	var stdout, stderr bytes.Buffer

	compileCmd := getCommand(action.CompileCmd, CompileCmd, req.ExtendedOptions, action)
	if compileCmd != "" {
		compileCtx := registerCmdTimeout(r.Context(), totalTimeout)
		{
			start := time.Now()
			parsedCmd := replacePlaceholders(compileCmd, req.Args, nil)
			runErr := execContainerShell(
				compileCtx,
				&stderr,
				&stdout,
				*cont,
				parsedCmd,
				cont.Image,
			)

			res.RunEnvironment.CompileCmd = compileCmd
			res.RunEnvironment.CompileTime = float32(time.Since(start).Seconds())

			if runErr != nil {
				if errors.Is(compileCtx.Err(), context.DeadlineExceeded) {
					sendRunError(w, "timeout compilation", res)
					return
				}

				flushStdWithErr(res, stderr, stdout)
				sendResponse(w, res)
				return
			}
		}
	}

	// TODO:: disconnect?

	runTTL := time.Duration(*cont.Image.ContainerOptions.RunTTL) * time.Second
	runTimeoutCtx := registerCmdTimeout(timeoutCtx, runTTL)
	runCmd := getCommand(action.RunCmd, RunCmd, req.ExtendedOptions, action)
	{
		start := time.Now()
		parsedRunCmd := replacePlaceholders(runCmd, req.Args, stdinFile)
		runErr := execContainerShell(
			runTimeoutCtx,
			&stderr,
			&stdout,
			*cont,
			parsedRunCmd,
			cont.Image,
		)

		res.RunEnvironment.RunCmd = runCmd
		res.RunEnvironment.RunTime = float32(time.Since(start).Seconds())

		if runErr != nil {
			if errors.Is(runTimeoutCtx.Err(), context.DeadlineExceeded) {
				sendRunError(w, "timeout execute", res)
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

func getCommand(cmd string, key string, externalData *map[string]string, action contract.ImageActionConfig) string {
	if externalData == nil {
		return cmd
	}

	if action.EnableExternalCommands == ExternalCommandsModeNode ||
		(action.EnableExternalCommands == ExternalCommandsModeCompile && key == RunCmd) ||
		(action.EnableExternalCommands == ExternalCommandsModeRun && key == CompileCmd) {
		return cmd
	}

	if value, ok := (*externalData)[key]; ok {
		if value != "" {
			return value
		}
	}

	return cmd
}

func sendResponse(w http.ResponseWriter, res *contract.SandboxResponse) {
	body, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	_, _ = w.Write(body)
}

func sendRunError(w http.ResponseWriter, err string, ctxRes *contract.SandboxResponse) {
	res := &contract.SandboxResponse{}
	res.Stderr = []byte(err)
	if ctxRes != nil {
		res.RunEnvironment = ctxRes.RunEnvironment
	}

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

func execContainerShell(ctx context.Context, stderr *bytes.Buffer, stdout *bytes.Buffer, container StartedContainer, runCmd string, cfg BuiltImage) error {
	sh := fmt.Sprintf("cd %s && %s", cfg.Workdir, runCmd)

	//nolint
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
	body, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	_, _ = w.Write(body)
}
func replacePlaceholders(input string, args string, stdinFileName *string) string {
	placeholders := map[string]string{
		"ARGS": args,
	}
	if stdinFileName != nil {
		placeholders["STDIN"] = *stdinFileName
	}

	re := regexp.MustCompile(`\{\s*([A-Z0-9_]+)\s*}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSpace(match[1 : len(match)-1])
		if val, exists := placeholders[key]; exists {
			return val
		}
		return match
	})
}

func listTemplatesHandler(w http.ResponseWriter, _ *http.Request) {
	body, err := json.MarshalIndent((*CodenireOrchestrator).GetTemplates(&CodenireOrchestrator{}), "", "  ")
	if err != nil {
		http.Error(w, "error encoding JSON", http.StatusInternalServerError)
		log.Printf("json marshal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}
func getTemplateByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	template, err := (*CodenireOrchestrator).GetTemplateByID(&CodenireOrchestrator{}, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func AddTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var template contract.ImageConfig
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := (*CodenireOrchestrator).AddTemplate(&CodenireOrchestrator{}, template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func runTemplateHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	container, err := (*CodenireOrchestrator).runTemplate(&CodenireOrchestrator{}, id)
	if err != nil {
		http.Error(w, "Failed to run template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(container)
}

func deleteTemplateHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := (*CodenireOrchestrator).DeleteTemplate(&CodenireOrchestrator{}, id)
	if err != nil {
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var template contract.ImageConfig
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if err := (*CodenireOrchestrator).updateTemplate(&CodenireOrchestrator{}, id, template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
