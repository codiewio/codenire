// Copyright:
//
// 2024 The Codenire Authors. All rights reserved.
// Authors:
//   - Maksim Fedorov mfedorov@codiew.io
//
// Licensed under the MIT License.
package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ListDirectories(path string) []string {
	var dd []string

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dd = append(dd, entry.Name())
		}
	}

	return dd
}

func ListDirRecursively(dirPath string, level int) error {

	prefTmp := "%s└── "

	// Читаем содержимое директории
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать директорию %s: %v", dirPath, err)
	}

	// Проходим по каждому элементу
	for _, entry := range entries {
		// Формируем отступ для вложенных элементов
		prefix := ""
		if level > 0 {
			prefix = strings.Repeat(prefTmp, level)
		}

		// Определяем тип элемента
		if entry.IsDir() {
			log.Printf("%s[Папка] %s\n", prefix, entry.Name())
			// Рекурсивно обрабатываем вложенную папку
			subDirPath := filepath.Join(dirPath, entry.Name())
			err := ListDirRecursively(subDirPath, level+1)
			if err != nil {
				return err
			}
		} else {
			log.Printf("%s[Файл] %s\n", prefix, entry.Name())
		}
	}

	return nil
}
