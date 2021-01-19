// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package contains some path-relevant utilities.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Rel, like path.Rel, converts a path into a relative path to directory with the knowledge of cwd.
// Cwd, which should be an absolute path, will be used if either base or path is a relative paths.
func Rel(base string, path string, cwd string) (relativePath string, err error) {
	relativePath, err = filepath.Rel(base, path)

	if err != nil {
		// Rel returns an error if CWD is necessary to compute it. Here is a fix.
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}

		if !filepath.IsAbs(base) {
			base = filepath.Join(cwd, base)
		}

		relativePath, err = filepath.Rel(base, path)
	}

	return
}

// Exists returns true only if path exists and is accessible.
func Exists(path string) bool {
	_, err := os.Stat(path)

	return !os.IsNotExist(err)
}

// IsInsideDir checks if path is within directory.
// Both path and directory have to be absolute paths.
func IsInsideDir(path string, directory string) (isInside bool, err error) {
	if !filepath.IsAbs(path) {
		err = fmt.Errorf(
			"argument `path` (%s) is not an absolute path",
			path,
		)
		return
	}
	if !filepath.IsAbs(directory) {
		err = fmt.Errorf(
			"argument `directory` (%s) is not an absolute path",
			directory,
		)
		return
	}

	rel, err := filepath.Rel(directory, path)
	if err != nil {
		return false, err
	}

	isInside = rel == "." || !strings.HasPrefix(rel, ".")

	return
}
