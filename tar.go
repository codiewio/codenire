package main

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
