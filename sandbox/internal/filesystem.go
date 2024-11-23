// Copyright:
//
// 2024 The Codenire Authors. All rights reserved.
// Authors:
//   - Maksim Fedorov mfedorov@codiew.io
//
// Licensed under the MIT License.
package internal

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
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

func CopyFilesToTmpDir(tmpDir string, files map[string]string) error {
	for f, src := range files {
		if !strings.Contains(f, "/") {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, f, src, parser.PackageClauseOnly)
			if err == nil && f.Name.Name != "main" {
				return errors.New(fmt.Sprintf("package Name must be main", err.Error()))
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
