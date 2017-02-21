// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"os"
	"path/filepath"
)

// ensure the path is absolute
// if not, prepend the directory to make absolute path
func EnsureAbsolute(directory string, filePath string) string {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(directory, filePath)
	}
	return filepath.Clean(filePath)
}

// check if file exists
func EnsureFileExists(name string) bool {
	_, err := os.Stat(name)
	return nil == err
}
