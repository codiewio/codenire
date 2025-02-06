// Package internal
// Copyright:
//
// 2024 The Codenire Authors. All rights reserved.
// Authors:
//   - Maksim Fedorov mfedorov@codiew.io
//
// Licensed under the MIT License.
package internal

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	contract "sandbox/api/gen"
	"strings"
)

const maxFilesLimit = 1 * 1024 * 1024

// DirToTar creates a tar archive from the specified directory
func DirToTar(sourceDir string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	defer func() {
		_ = tarWriter.Close()
	}()

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking through directory %s: %w", path, err)
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

			if _, err3 := io.Copy(tarWriter, file); err3 != nil {
				return fmt.Errorf("error writing file content %s to archive: %w", path, err3)
			}
		}

		return nil
	})

	return buf, err
}

// nolint:gocognit
func SaveRequestFiles(req contract.SandboxRequest, destDir string) (stdinFile *string, err error) {
	tarData, err := base64.StdEncoding.DecodeString(req.Binary)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}

	tarReader := tar.NewReader(bytes.NewReader(tarData))

	// STDIN
	{
		inputName := fmt.Sprintf("input_%s.txt", RandHex(8))
		inputFilePath := filepath.Join(destDir, inputName)
		file, err2 := os.OpenFile(inputFilePath, os.O_CREATE|os.O_RDWR, 0644)
		if err2 != nil {
			return nil, fmt.Errorf("error creating input.txt file: %w", err2)
		}
		defer func() {
			_ = file.Close()
		}()

		_, err = file.WriteString(req.Stdin)
		if err != nil {
			return nil, fmt.Errorf("error writing to %s: %w", inputName, err)
		}

		stdinFile = &inputName
	}

	for {
		header, err2 := tarReader.Next()
		if err2 == io.EOF {
			break // End of the tar archive
		}
		if err2 != nil {
			return nil, fmt.Errorf("error reading header: %w", err2)
		}

		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") {
			return nil, fmt.Errorf("detected path traversal attempt: %s", header.Name)
		}

		targetPath := filepath.Join(destDir, cleanName)

		//nolint:gosec
		mode := os.FileMode(header.Mode)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory if it doesn't exist
			if err = os.MkdirAll(targetPath, mode); err != nil {
				return nil, fmt.Errorf("error creating directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			// Open file for writing, create it if it doesn't exist
			file, err3 := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, mode)
			if err3 != nil {
				return nil, fmt.Errorf("error creating file %s: %w", targetPath, err3)
			}

			defer func() {
				_ = file.Close()
			}()

			// Copy the content from tar archive to the file
			limitedReader := io.LimitReader(tarReader, maxFilesLimit)

			if _, err4 := io.Copy(file, limitedReader); err4 != nil {
				return nil, fmt.Errorf("error writing to file %s: %w", targetPath, err4)
			}
		}
	}

	return stdinFile, nil
}
