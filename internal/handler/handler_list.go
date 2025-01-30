package handler

import (
	"net/http"

	"github.com/codiewio/codenire/internal/images"
)

func (h *Handler) ImagesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	res := images.ConfigList

	writeJSONResponse(w, res, http.StatusOK)
}

func (h *Handler) RefreshImageConfigList(w http.ResponseWriter, r *http.Request) {

}
