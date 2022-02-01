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
	"fmt"
	"io/fs"
	"path/filepath"
)

type Dir struct {
	Directory string
}

var _ Reference = Dir{}

func parseDir(location string) Reference {
	dir := filepath.Clean(location)

	if fs.ValidPath(dir) {
		return Dir{
			Directory: dir,
		}
	}

	return nil
}

// String implements location.Reference
func (ref Dir) String() string {
	return fmt.Sprintf("type:dir directory:%q", ref.Directory)
}

// Type implements location.Reference
func (ref Dir) Type() string {
	return "dir"
}

// Validate implements location.Reference
func (ref Dir) Validate() error {
	return nil
}
