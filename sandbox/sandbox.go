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
	maxBinarySize    = 100 << 20
	startTimeout     = 100 * time.Second
	runTimeout       = 5 * time.Second
	maxOutputSize    = 100 << 20
	memoryLimitBytes = 100 << 20
)

var (
	errTooMuchOutput = errors.New("output too large")
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	io.WriteString(w, "Hi from sandbox\n")
}

func listImageHandler(w http.ResponseWriter, r *http.Request) {
	rows := codenireManager.ImageList("codenire/")
	for _, row := range rows {
		fmt.Println("[row]: ", row)
	}

	//sendRunResponse(w, res)
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
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "tmp_sandbox")
	if err != nil {
		http.Error(w, "create tmp dir failed", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	err = internal.Base64ToTar(req.Binary, tmpDir)
	if err != nil {
		http.Error(w, "encode  files failed", http.StatusInternalServerError)
		return
	}

	cont, err := codenireManager.GetContainer(r.Context(), req.SandId)
	if err != nil {
		http.Error(w, fmt.Sprintf("get container %s failed", req.SandId), http.StatusInternalServerError)
		return
	}

	if cont == nil {
		http.Error(w, fmt.Sprintf("container %s not found", req.SandId), http.StatusInternalServerError)
		return
	}

	if cont.Image == nil {
		http.Error(w, fmt.Sprintf("image for %s not found", cont.CId), http.StatusInternalServerError)
	}

	defer func() {
		err = codenireManager.KillContainer(cont.CId)
		if err != nil {
			// TODO:: handle it and log
		}
	}()

	out, err := exec.Command("docker", "cp", tmpDir+"/.", cont.CId+":/tmp").CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to docker: %v, %s", err, out), http.StatusInternalServerError)
		return
	}

	// todo:: call compileCmd with compileTimeout
	if cont.Image.CompileCmd != nil {

	}

	cmd := exec.Command("docker", "exec", cont.CId, "php", "/tmp/"+cont.Image.RunCmd)

	// Буферы для stdout и stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Выполняем команду
	err = cmd.Run()

	// Получаем код выхода
	var exitCode int
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return
		}
	}

	//// Получение вывода команды
	//output, err := cmd.CombinedOutput()
	//
	//// Вывод результата команды
	//fmt.Printf("Output from container: %s\n", output)

	res := &api.SandboxResponse{}
	res.ExitCode = exitCode
	res.Stderr = stderr.Bytes()
	res.Stdout = stdout.Bytes()

	sendRunResponse(w, res)
}

func errExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}

func sendError(w http.ResponseWriter, errMsg string) {
	sendRunResponse(w, &api.SandboxResponse{Error: &errMsg})
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
