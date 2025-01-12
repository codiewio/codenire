package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	api "github.com/codiewio/codenire/api/gen"
)

var (
	// MaxSnippetSize TODO:: implement in goplay plugin 60 * 1024
	MaxSnippetSize int64 = 1 * 1024 * 1024
)

type Handler struct {
	Config *Config
}

func (h *Handler) RunHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reader := http.MaxBytesReader(nil, r.Body, MaxSnippetSize)
	defer reader.Close()

	var req api.SubmissionRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		maxBytesErr := new(http.MaxBytesError)
		if errors.As(err, &maxBytesErr) {
			http.Error(w, fmt.Sprintf("code snippet too large (max %d bytes): ", MaxSnippetSize)+err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Println("Playground: invalid request", err)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if h.Config.PreRequestCallback != nil {
		resp2, err := h.Config.PreRequestCallback(newHookEvent(r.Context(), req))
		if err != nil {
			fmt.Println("Playground: Pre-Request callback failed", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if resp2.IsTerminated {
			resp2.WriteTo(w)
			return
		}
	}

	tmpDir, err := os.MkdirTemp("", "box")
	if err != nil {
		fmt.Println("Playground: create tmp dir error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	err = copyFilesToTmpDir(tmpDir, req.Files)
	if err != nil {
		fmt.Println("Playground: copying files into tmp dir failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := tarToBase64(tmpDir)
	if err != nil {
		fmt.Println("Playground: zip tmp dit to base66 failed", err)
		http.Error(w, "fail on create tar files: "+err.Error(), http.StatusInternalServerError)
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
		fmt.Println("Playground: request marshal error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sreq, err := http.NewRequestWithContext(ctx, "POST", h.Config.BackendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Playground: Sandbox client request error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sreq.Header.Add("Idempotency-Key", "1")

	sreq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(jsonData)), nil }
	resp, err := SandboxBackendClient().Do(sreq)
	if err != nil {
		log.Printf("Playground: Sandbox client request error: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)

		return
	}
	defer resp.Body.Close()

	// TODO:: [HOOK] post-response hook

	if resp.StatusCode != http.StatusOK {
		log.Printf("Playground: unexpected response from backend: %v", resp.Status)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	var execRes api.SandboxResponse

	if err = json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		log.Printf("Playground: JSON decode error from backend: %v", err)
		http.Error(w, "error parsing JSON from backend", http.StatusBadGateway)
		return
	}

	rec := new(Recorder)
	rec.Stdout().Write(execRes.Stdout)
	rec.Stderr().Write(execRes.Stderr)
	events, err := rec.Events()
	if err != nil {
		log.Printf("Playground: error decoding events: %v", err)
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
				return errors.New(fmt.Sprint("package name must be main", err.Error()))
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

func SandboxBackendClient() *http.Client {
	sandboxBackendOnce.Do(func() {
		sandboxBackendOnce.c = http.DefaultClient
	})

	return sandboxBackendOnce.c
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
