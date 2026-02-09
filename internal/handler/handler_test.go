package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyFilesToTmpDir_PathTraversal_ShouldBeBlocked(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
	}{
		{"dot-dot-slash", "../escape.txt"},
		{"multiple-dot-dot", "../../escape.txt"},
		{"deeply-nested-escape", "subdir/../../escape.txt"},
		{"dot-dot-in-middle", "foo/../../../escape.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subTmpDir, err := os.MkdirTemp("", "test_sub_sandbox")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(subTmpDir)

			files := map[string]string{
				tc.filename: "escaped content",
			}

			err = copyFilesToTmpDir(subTmpDir, files)

			if err == nil {
				t.Errorf("expected error for path traversal attempt %q, got nil", tc.filename)
			} else if !strings.Contains(err.Error(), "path traversal detected") {
				t.Errorf("expected 'path traversal detected' error, got: %v", err)
			}

			resultPath := filepath.Join(subTmpDir, tc.filename)
			cleanPath := filepath.Clean(resultPath)
			if _, statErr := os.Stat(cleanPath); statErr == nil {
				t.Errorf("file should NOT exist at escaped path: %s", cleanPath)
			}
		})
	}
}

func TestCopyFilesToTmpDir_ValidPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_sandbox")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validFiles := map[string]string{
		"main.go":           "package main",
		"src/util.go":       "package src",
		"src/lib/helper.go": "package lib",
	}

	err = copyFilesToTmpDir(tmpDir, validFiles)
	if err != nil {
		t.Fatalf("copyFilesToTmpDir failed for valid paths: %v", err)
	}

	for filename, expectedContent := range validFiles {
		filePath := filepath.Join(tmpDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read %s: %v", filename, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("content mismatch for %s: got %q, want %q", filename, string(content), expectedContent)
		}
	}
}
