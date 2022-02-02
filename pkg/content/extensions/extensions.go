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

package extensions

import (
	"io/fs"

	"github.com/GoogleContainerTools/kpt/pkg/location"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type ChangeCommitter interface {
	CommitChanges() (location.Location, error)
}

type FileSystemProvider interface {
	ProvideFileSystem() (filesys.FileSystem, string, error)
}

type FSProvider interface {
	ProvideFS() (fs.FS, error)
}

type ReaderProvider interface {
	ProvideReader() (kio.Reader, error)
}
