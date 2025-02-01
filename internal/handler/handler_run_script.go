package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	api "github.com/codiewio/codenire/api/gen"
	"github.com/codiewio/codenire/internal/images"
	"github.com/codiewio/codenire/pkg/hooks"
)

var (
	MaxScriptSnippetSize int64 = 1 * 1024 * 1024
)

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

		http.Error(w, "[playground] invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	cfg := images.GetImageConfig(preReq.TemplateId)
	if cfg == nil {
		http.Error(w, fmt.Sprintf("[playground] template `%s` not found", preReq.TemplateId), http.StatusBadRequest)
		return
	}

	req := api.SubmissionRequest{
		TemplateId: preReq.TemplateId,
		Args:       preReq.Args,
		Files:      make(map[string]string),
	}

	sourceFile := cfg.ScriptOptions.SourceFile
	req.Files[sourceFile] = preReq.Code
	req.Files = addDefaultFiles(req.Files, cfg.DefaultFiles)

	if h.Config.PreRequestCallback != nil {
		resp2, err := h.Config.PreRequestCallback(hooks.NewCodeHookEvent(c, req))
		if err != nil {
			err = fmt.Errorf("[playground] pre-SubmissionRequest callback failed: %w", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
