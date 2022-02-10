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

package content

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/content/extensions"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func Reader(content Content) (kio.Reader, error) {
	switch content := content.(type) {
	case extensions.ReaderProvider:
		return content.Reader()
	case extensions.FileSystemProvider:
		fsys, path, err := content.FileSystem()
		if err != nil {
			return nil, err
		}
		return &kio.LocalPackageReader{
			PackagePath:        path,
			IncludeSubpackages: true,

			FileSystem: filesys.FileSystemOrOnDisk{FileSystem: fsys},
		}, err
	default:
		// TODO(https://github.com/GoogleContainerTools/kpt/issues/2764) add additional cases with adapters
		return nil, fmt.Errorf("not implemented")
	}
}
