package handler

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func tarToBase64(sourceDir string) (string, error) {
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	defer func() {
		_ = tarWriter.Close()
	}()

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking the directory %s: %w", path, err)
		}

		if path == sourceDir {
			return nil
		}

		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return fmt.Errorf("error creating header for %s: %w", path, err)
		}

		header.Name, err = filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("error getting relative path for %s: %w", path, err)
		}

		if err2 := tarWriter.WriteHeader(header); err2 != nil {
			return fmt.Errorf("error writing header: %w", err2)
		}

		if !info.IsDir() {
			file, err2 := os.Open(path)
			if err2 != nil {
				return fmt.Errorf("error opening file %s: %w", path, err2)
			}
			defer func() {
				_ = file.Close()
			}()

			if _, errCopy := io.Copy(tarWriter, file); errCopy != nil {
				return fmt.Errorf("error writing file contents of %s to archive: %w", path, errCopy)
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())
	return base64Data, nil
}
