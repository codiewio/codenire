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
)

func DirToTar(sourceDir string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	defer tarWriter.Close()

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("ошибка обхода директории %s: %v", path, err)
		}

		if path == sourceDir {
			return nil
		}

		// Создаем заголовок для файла или папки
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return fmt.Errorf("ошибка создания заголовка для %s: %v", path, err)
		}

		header.Name, err = filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("ошибка получения относительного пути для %s: %v", path, err)
		}

		// Пишем заголовок в архив
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("ошибка записи заголовка: %v", err)
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("ошибка открытия файла %s: %v", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("ошибка записи содержимого файла %s в архив: %v", path, err)
			}
		}

		return nil
	})

	return buf, err
}

func Base64ToTar(base64Data, destDir string, stdin string) (stdinFile *string, err error) {
	tarData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %v", err)
	}

	tarReader := tar.NewReader(bytes.NewReader(tarData))

	{
		inputName := fmt.Sprintf("input_%s.txt", RandHex(8))
		inputFilePath := filepath.Join(destDir, inputName)
		file, err := os.OpenFile(inputFilePath, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return nil, fmt.Errorf("error creating input.txt file: %w", err)
		}
		defer file.Close()

		_, err = file.WriteString(stdin)
		if err != nil {
			return nil, fmt.Errorf("error writing to %s: %w", inputName, err)
		}

		stdinFile = &inputName
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of the tar archive
		}
		if err != nil {
			return nil, fmt.Errorf("error reading header: %v", err)
		}

		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory if it doesn't exist
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return nil, fmt.Errorf("error creating directory %s: %v", targetPath, err)
			}
		case tar.TypeReg:
			// Open file for writing, create it if it doesn't exist
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return nil, fmt.Errorf("error creating file %s: %v", targetPath, err)
			}
			defer file.Close()

			// Copy the content from tar archive to the file
			if _, err := io.Copy(file, tarReader); err != nil {
				return nil, fmt.Errorf("error writing to file %s: %v", targetPath, err)
			}
		}
	}

	return stdinFile, nil
}
