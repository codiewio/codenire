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
)

type Handler struct {
	Config *Config
}

func copyFilesToTmpDir(tmpDir string, files map[string]string) error {
	cleanTmpDir, err := filepath.Abs(tmpDir)
	if err != nil {
		return err
	}

	for relPath, content := range files {
		targetPath := filepath.Join(cleanTmpDir, relPath)

		rel, err := filepath.Rel(cleanTmpDir, targetPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("invalid file path %q: path traversal detected", relPath)
		}

		if strings.Contains(relPath, "/") {
			if err = os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
		}

		if err = os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("error creating temp file %q: %w", targetPath, err)
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
