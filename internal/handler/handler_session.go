package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/codiewio/codenire/internal/client"
	"io"
	"net/http"

	api "github.com/codiewio/codenire/api/gen"

	"github.com/gorilla/websocket"
)

func (h *Handler) SessionConnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	reader := http.MaxBytesReader(nil, r.Body, MaxFilesSnippetSize)
	defer func() {
		_ = reader.Close()
	}()

	var req api.StartSessionRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(
		api.StartSessionRequest{
			Id:      req.Id,
			Version: req.Version,
			Cluster: req.Cluster,
		},
	)
	if err != nil {
		http.Error(w, "invalid marshaling: "+err.Error(), http.StatusBadRequest)
		return
	}

	sreq, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		h.Config.BackendURL+"/session/connect",
		bytes.NewBuffer(jsonData),
	)
	sreq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBuffer(jsonData)), nil }
	resp, err := client.SandboxBackendClient().Do(sreq)
	if err != nil {
		sandboxErr := fmt.Errorf("sandbox client request error: %w", err)
		http.Error(w, sandboxErr.Error(), http.StatusConflict)

		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		resErr := fmt.Errorf("unexpected http status from backend: %d", resp.StatusCode)
		http.Error(w, resErr.Error(), http.StatusConflict)

		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "error reading backend response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(body)
}

func (h *Handler) SessionCodeHandler(w http.ResponseWriter, r *http.Request) {
	reader := http.MaxBytesReader(nil, r.Body, MaxFilesSnippetSize)
	defer func() {
		_ = reader.Close()
	}()

	var req api.StartSessionRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	cluster := 1
	req.Cluster = &cluster

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	{
		err = conn.WriteMessage(websocket.TextMessage, []byte("ID connected: "+req.Id))
		if err != nil {
			fmt.Println("Ошибка отправки данных:", err)
		}
	}
}
