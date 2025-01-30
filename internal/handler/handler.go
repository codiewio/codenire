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
)

type Handler struct {
	Config *Config
}

func copyFilesToTmpDir(tmpDir string, files map[string]string) error {
	for f, src := range files {
		// TODO:: golang logic — remove it
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
