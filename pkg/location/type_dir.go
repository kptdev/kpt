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

package location

import (
	"io/fs"
	"path/filepath"
)

type Dir struct {
	// Directory is absolute or relative to current working directory.
	// This path is always based on the OS file system.
	Directory string

	// original is the value before parsing, it is returned
	// by String() to improve round-trip accuracy
	original string
}

var _ Reference = Dir{}

func parseDir(location string) Reference {
	dir := filepath.Clean(location)

	if fs.ValidPath(dir) {
		return Dir{
			Directory: dir,
			original:  location,
		}
	}

	return nil
}

// String implements location.Reference
func (ref Dir) String() string {
	if ref.original != "" {
		return ref.original
	}
	return ref.Directory
}

// Type implements location.Reference
func (ref Dir) Type() string {
	return "dir"
}

// Validate implements location.Reference
func (ref Dir) Validate() error {
	return nil
}
