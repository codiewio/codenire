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
	"github.com/codiewio/codenire/internal/client"
	"github.com/codiewio/codenire/internal/images"
	"github.com/codiewio/codenire/pkg/hooks"
)

var (
	MaxFilesSnippetSize  int64 = 60 * 1024
	MaxScriptSnippetSize int64 = 1 * 1024 * 1024
)

func (h *Handler) RunFilesHandler(w http.ResponseWriter, r *http.Request) {
	c := hooks.GetContext(w, r, h.Config.GracefulRequestCompletionTimeout)

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

		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg := images.GetImageConfig(req.TemplateId)
	if cfg == nil {
		http.Error(w, fmt.Sprintf("template `%s` not found", req.TemplateId), http.StatusBadRequest)
		return
	}

	action, err := getAction(req.ActionId, cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Files = addDefaultFiles(req.Files, action.DefaultFiles)

	if h.Config.PreRequestCallback != nil {
		resp2, err2 := h.Config.PreRequestCallback(hooks.NewCodeHookEvent(c, req))
		if err2 != nil {
			http.Error(w, fmt.Errorf("pre-SubmissionRequest callback failed: %w", err2).Error(), http.StatusInternalServerError)
			return
		}

		if resp2.IsTerminated {
			resp2.WriteTo(w)
			return
		}

		if resp2.ChangedSubmissionRequest != nil {
			req = *resp2.ChangedSubmissionRequest
		}
	}

	apiRes, err := runCode(r.Context(), req, h.Config.BackendURL+"/run")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, apiRes, http.StatusOK)
}

func (h *Handler) RunScriptHandler(w http.ResponseWriter, r *http.Request) {
	c := hooks.GetContext(w, r, h.Config.GracefulRequestCompletionTimeout)

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	reader := http.MaxBytesReader(nil, r.Body, MaxScriptSnippetSize)
	defer reader.Close()

	var preReq api.SubmissionScriptRequest
	if err := json.NewDecoder(reader).Decode(&preReq); err != nil {
		maxBytesErr := new(http.MaxBytesError)
		if errors.As(err, &maxBytesErr) {
			err = fmt.Errorf("code snippet too large (max %d bytes): %w", MaxScriptSnippetSize, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg := images.GetImageConfig(preReq.TemplateId)
	if cfg == nil {
		http.Error(w, fmt.Sprintf("template `%s` not found", preReq.TemplateId), http.StatusBadRequest)
		return
	}

	action, err := getAction(preReq.ActionId, cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req := api.SubmissionRequest{
		TemplateId:      preReq.TemplateId,
		Args:            preReq.Args,
		Files:           make(map[string]string),
		ActionId:        preReq.ActionId,
		ExternalOptions: preReq.ExternalOptions,
	}

	sourceFile := action.ScriptOptions.SourceFile
	// TODO:: check if ""
	req.Files[sourceFile] = preReq.Code
	req.Files = addDefaultFiles(req.Files, action.DefaultFiles)

	if h.Config.PreRequestCallback != nil {
		resp2, err2 := h.Config.PreRequestCallback(hooks.NewCodeHookEvent(c, req))
		if err2 != nil {
			http.Error(w, fmt.Errorf("pre-SubmissionRequest callback failed: %w", err2).Error(), http.StatusInternalServerError)
			return
		}

		if resp2.IsTerminated {
			resp2.WriteTo(w)
			return
		}

		if resp2.ChangedSubmissionRequest != nil {
			req = *resp2.ChangedSubmissionRequest
		}
	}

	apiRes, err := runCode(r.Context(), req, h.Config.BackendURL+"/run")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, apiRes, http.StatusOK)
}

func runCode(ctx context.Context, req api.SubmissionRequest, backendURL string) (*api.SubmissionResponse, error) {
	tmpDir, err := os.MkdirTemp("", "box")
	if err != nil {
		return nil, fmt.Errorf("create tmp dir error: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	err = copyFilesToTmpDir(tmpDir, req.Files)
	if err != nil {
		return nil, fmt.Errorf("copying files into tmp dir failed: %w", err)
	}

	b, err := tarToBase64(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("fail on create tar files: %w", err)
	}

	action := "default"
	if req.ActionId != nil {
		action = *req.ActionId
	}
	jsonData, err := json.Marshal(
		api.SandboxRequest{
			Args:            req.Args,
			SandId:          req.TemplateId,
			Binary:          b,
			Stdin:           req.Stdin,
			Action:          action,
			ExtendedOptions: req.ExternalOptions,
		},
	)
	if err != nil {
		return nil, err
	}

	sreq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		backendURL,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("request marshal error: %w", err)
	}

	sreq.Header.Add("Idempotency-Key", "1")

	sreq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(jsonData)), nil }
	resp, err := client.SandboxBackendClient().Do(sreq)
	if err != nil {
		return nil, fmt.Errorf("sandbox client request error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// TODO:: [HOOK] post-response hook

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status from backend: %d", resp.StatusCode)
	}

	var execRes api.SandboxResponse
	if err = json.NewDecoder(resp.Body).Decode(&execRes); err != nil {
		return nil, fmt.Errorf("jSON decode error from backend: %w", err)
	}

	rec := new(Recorder)
	_, _ = rec.Stdout().Write(execRes.Stdout)
	_, _ = rec.Stderr().Write(execRes.Stderr)
	events, err := rec.Events()
	if err != nil {
		return nil, fmt.Errorf("error decoding events: %w", err)
	}

	apiRes := &api.SubmissionResponse{
		Events: events,
		Meta:   nil,
		Time:   nil,
	}

	return apiRes, nil
}

func getAction(reqAction *string, cfg *api.ImageConfig) (*api.ImageActionConfig, error) {
	actionID := "default"
	if reqAction != nil {
		actionID = *reqAction
	}

	action, ok := cfg.Actions[actionID]
	if !ok {
		return nil, fmt.Errorf("action `%s` not found", *reqAction)
	}

	return &action, nil
}
