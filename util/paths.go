// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"os"
	"path/filepath"
)

// EnsureAbsolute - ensure the path is absolute
// if not, prepend the directory to make absolute path
func EnsureAbsolute(directory string, filePath string) string {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(directory, filePath)
	}
	return filepath.Clean(filePath)
}

// EnsureFileExists - check if file exists
func EnsureFileExists(name string) bool {
	_, err := os.Stat(name)
	return nil == err
}
