// Copyright 2022 Google LLC
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

package paths

import (
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// FileSystemPath represents a FileSystem and Path pair. The Path is always valid on the
// associated FileSystem, but is not guaranteed to be meaningful on the os disk or other
// unrelated FileSystem instances.
type FileSystemPath struct {
	FileSystem filesys.FileSystem
	Path       string
}

// String returns the path in string format.
func (fsp FileSystemPath) String() string {
	return fsp.Path
}

type FSPath struct {
	FileSystem filesys.FileSystem
	Path       string
}

func (fsp FSPath) String() string {
	return fsp.Path
}
