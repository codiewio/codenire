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

// TarToBase64 Функция архивирования в tar и кодирования в base64.
func TarToBase64(sourceDir string) (string, error) {
	// Создаем буфер для записи TAR-архива
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	defer tarWriter.Close()

	// Рекурсивно архивируем файлы и папки
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("ошибка обхода директории %s: %v", path, err)
		}

		// Пропускаем корневую директорию
		if path == sourceDir {
			return nil
		}

		// Создаем заголовок для файла или папки
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return fmt.Errorf("ошибка создания заголовка для %s: %v", path, err)
		}

		// Устанавливаем относительный путь
		header.Name, err = filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("ошибка получения относительного пути для %s: %v", path, err)
		}

		// Пишем заголовок в архив
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("ошибка записи заголовка: %v", err)
		}

		// Если это файл, записываем его содержимое
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
	if err != nil {
		return "", err
	}

	// Кодируем TAR-архив в Base64
	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())
	return base64Data, nil
}

// DirToTar Функция архивирования в tar и кодирования в base64
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

//func AddFilesToTar(dir string, tw *tar.Writer) error {
//	dirInfo, err := os.Stat(dir)
//	if err != nil {
//		return err
//	}
//
//	if dirInfo.IsDir() {
//		files, err := os.ReadDir(dir)
//		if err != nil {
//			return err
//		}
//		for _, file := range files {
//			filePath := fmt.Sprintf("%s/%s", dir, file.Name())
//			err := AddFilesToTar(filePath, tw)
//			if err != nil {
//				return err
//			}
//		}
//	} else {
//		file, err := os.Open(dir)
//		if err != nil {
//			return err
//		}
//		defer file.Close()
//
//		name := strings.TrimPrefix(dir, imagesDir+"/php7/")
//		header := &tar.Header{
//			Name: name, // удаляем префикс для корректного пути в архиве
//			Mode: 0600,
//			Size: dirInfo.Size(),
//		}
//		if err := tw.WriteHeader(header); err != nil {
//			return err
//		}
//		_, err = io.Copy(tw, file)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
