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

// Функция архивирования в tar и кодирования в base64
func tarToBase64(sourceDir string) (string, error) {
	// Создаем буфер для записи TAR-архива
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	defer func() {
		_ = tarWriter.Close()
	}()

	// Рекурсивно архивируем файлы и папки
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("ошибка обхода директории %s: %w", path, err)
		}

		// Пропускаем корневую директорию
		if path == sourceDir {
			return nil
		}

		// Создаем заголовок для файла или папки
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return fmt.Errorf("ошибка создания заголовка для %s: %w", path, err)
		}

		// Устанавливаем относительный путь
		header.Name, err = filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("ошибка получения относительного пути для %s: %w", path, err)
		}

		// Пишем заголовок в архив
		if err2 := tarWriter.WriteHeader(header); err2 != nil {
			return fmt.Errorf("ошибка записи заголовка: %w", err2)
		}

		// Если это файл, записываем его содержимое
		if !info.IsDir() {
			file, err2 := os.Open(path)
			if err2 != nil {
				return fmt.Errorf("ошибка открытия файла %s: %w", path, err2)
			}
			defer func() {
				_ = file.Close()
			}()

			if _, errCopy := io.Copy(tarWriter, file); errCopy != nil {
				return fmt.Errorf("ошибка записи содержимого файла %s в архив: %w", path, errCopy)
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
