// Copyright:
//
// 2024 The Codenire Authors. All rights reserved.
// Authors:
//   - Maksim Fedorov mfedorov@codiew.io
//
// Licensed under the MIT License.
package internal

import (
	"os"
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
