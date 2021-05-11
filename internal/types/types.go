// Copyright 2021 Google LLC
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

// Package types defines the basic types used by the kpt codebase.
package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UniquePath represents absolute unique OS-defined path to the package directory on the filesystem.
type UniquePath string

// String returns the absolute path in string format.
func (u UniquePath) String() string {
	return string(u)
}

// Empty returns true if the UniquePath is empty
func (u UniquePath) Empty() bool {
	return len(u) == 0
}

// RelativePath returns the relative path to current working directory.
func (u UniquePath) RelativePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	rPath, err := filepath.Rel(cwd, string(u))
	if err != nil {
		return string(u), err
	}
	if strings.HasPrefix(rPath, "..") {
		return string(u), nil
	}
	return rPath, nil
}

// DisplayPath represents os-agnostic Slash-separated path to the package directory on the filesystem relative
// to parent directory of root package on which the command is invoked.
// root package is defined as the package on which the command is invoked by user
// This is not guaranteed to be unique (e.g. in presence of symlinks) and should only
// be used for display purposes and is subject to change.
type DisplayPath string

// NewDisplayPath returns the os-agnostic Slash-separated path for display purposes
// the input path must be a relative path, returns error if input path is absolute
func NewDisplayPath(path string) (DisplayPath, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("input path must be a relative path, found absolute path %q", path)
	}
	path = filepath.ToSlash(path)
	return DisplayPath(path), nil
}

// Empty returns true if the DisplayPath is empty
func (u DisplayPath) Empty() bool {
	return len(u) == 0
}
