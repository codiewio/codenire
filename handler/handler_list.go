package handler

import (
	"io"
	"net/http"
)

func (h *Handler) ImagesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	context := r.Context()

	sreq, err := http.NewRequestWithContext(
		context,
		"GET",
		h.Config.BackendURL+"/images/list",
		nil,
	)

	if err != nil {
		http.Error(w, "Playground: Sandbox client request error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := SandboxBackendClient().Do(sreq)
	if err != nil {
		log.Printf("Playground: Sandbox client request error: %v", err)
		http.Error(w, "Playground: Sandbox client request error: "+err.Error(), http.StatusBadGateway)

		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)

	if err != nil {
		http.Error(w, "Playground: Failed to copy response body", http.StatusInternalServerError)
	}
}
