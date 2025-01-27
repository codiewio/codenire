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

func Base64ToTar(base64Data, destDir string) error {
	tarData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("ошибка декодирования Base64: %v", err)
	}

	tarReader := tar.NewReader(bytes.NewReader(tarData))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка чтения заголовка: %v", err)
		}

		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("ошибка создания директории %s: %v", targetPath, err)
			}
		case tar.TypeReg:
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("ошибка создания файла %s: %v", targetPath, err)
			}
			defer file.Close()

			if _, err := io.Copy(file, tarReader); err != nil {
				return fmt.Errorf("ошибка записи в файл %s: %v", targetPath, err)
			}
		}
	}

	return nil
}
