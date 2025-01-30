package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	api "github.com/codiewio/codenire/api/gen"
)

var (
	MaxFilesSnippetSize int64 = 60 * 1024
)

func (h *Handler) RunFilesHandler(w http.ResponseWriter, r *http.Request) {
	c := h.getContext(w, r)

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	reader := http.MaxBytesReader(nil, r.Body, MaxFilesSnippetSize)
	defer reader.Close()

	var req api.SubmissionRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		maxBytesErr := new(http.MaxBytesError)
		if errors.As(err, &maxBytesErr) {
			http.Error(w, fmt.Sprintf("code snippet too large (max %d bytes): ", MaxFilesSnippetSize)+err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "[playground] invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if h.Config.PreRequestCallback != nil {
		resp2, err := h.Config.PreRequestCallback(newHookEvent(c, req))
		if err != nil {
			err = fmt.Errorf("[playground] pre-SubmissionRequest callback failed: %w", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if resp2.IsTerminated {
			resp2.WriteTo(w)
			return
		}
	}

	apiRes, err := runCode(r.Context(), req, h.Config.BackendURL+"/run")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, apiRes, http.StatusOK)
}

func runCode(ctx context.Context, req api.SubmissionRequest, backendUrl string) (*api.SubmissionResponse, error) {
	tmpDir, err := os.MkdirTemp("", "box")
	if err != nil {
		return nil, fmt.Errorf("[playground] create tmp dir error: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	err = copyFilesToTmpDir(tmpDir, req.Files)
	if err != nil {
		return nil, fmt.Errorf("[playground] copying files into tmp dir failed: %w", err)
	}

	b, err := tarToBase64(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("fail on create tar files: %w", err)
	}

	jsonData, err := json.Marshal(
		api.SandboxRequest{
			Args:   req.Args,
			SandId: req.TemplateId,
			Binary: b,
		},
	)
	if err != nil {
		return nil, err
	}

	sreq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		backendUrl,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("[playground] request marshal error: " + err.Error())
	}

	sreq.Header.Add("Idempotency-Key", "1")

	sreq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(jsonData)), nil }
	resp, err := SandboxBackendClient().Do(sreq)
	if err != nil {
		return nil, fmt.Errorf("[playground] sandbox client request error: %w", err)
	}
	defer resp.Body.Close()

	// TODO:: [HOOK] post-response hook

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[playground] unexpected http status from backend: %d", resp.StatusCode)
	}

	var execRes api.SandboxResponse

	if err = json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		return nil, fmt.Errorf("[playground] jSON decode error from backend: %w", err)
	}

	rec := new(Recorder)
	rec.Stdout().Write(execRes.Stdout)
	rec.Stderr().Write(execRes.Stderr)
	events, err := rec.Events()
	if err != nil {
		return nil, fmt.Errorf("[playground] error decoding events: %w", err)
	}

	apiRes := &api.SubmissionResponse{
		Events: events,
		Meta:   nil,
		Time:   nil,
	}

	return apiRes, nil
}
