package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"
)

type Handler struct {
	Config *Config
	Conn   *websocket.Conn
}

func NewHandler(config *Config) *Handler {
	return &Handler{
		Config: config,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func copyFilesToTmpDir(tmpDir string, files map[string]string) error {
	for f, src := range files {
		in := filepath.Join(tmpDir, f)
		if strings.Contains(f, "/") {
			if err := os.MkdirAll(filepath.Dir(in), 0755); err != nil {
				return err
			}
		}
		//nolint
		if err := os.WriteFile(in, []byte(src), 0644); err != nil {
			return fmt.Errorf("error creating temp file %q: %w", in, err)
		}
	}

	return nil
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

func addDefaultFiles(files, defaultFiles map[string]string) map[string]string {
	for key, value := range defaultFiles {
		if _, exists := files[key]; !exists {
			files[key] = value
		}
	}

	return files
}
