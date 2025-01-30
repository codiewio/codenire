package handler

import (
	"net/http"

	"github.com/codiewio/codenire/internal/images"
)

func (h *Handler) ImagesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		refreshData(w)
		return
	}

	if r.Method == http.MethodPost {
		_ = images.PullImageConfigList(h.Config.BackendURL + "/images/list")
		refreshData(w)
		return
	}

	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	return
}

func refreshData(w http.ResponseWriter) {
	res := images.ConfigList
	writeJSONResponse(w, res, http.StatusOK)
}
