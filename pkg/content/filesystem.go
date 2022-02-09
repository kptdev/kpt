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

package content

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/content/extensions"
	"github.com/GoogleContainerTools/kpt/pkg/content/paths"
)

func FileSystem(content Content) (paths.FileSystemPath, error) {
	switch content := content.(type) {
	case extensions.FileSystemProvider:
		fsys, path, err := content.FileSystem()
		return paths.FileSystemPath{
			FileSystem: fsys,
			Path:       path,
		}, err
	default:
		// TODO(https://github.com/GoogleContainerTools/kpt/issues/2764) add additional cases with adapters
		return paths.FileSystemPath{}, fmt.Errorf("not implemented")
	}
}
