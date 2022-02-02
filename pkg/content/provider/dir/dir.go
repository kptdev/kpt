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

package dir

import (
	"io/fs"
	"os"

	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/content/extensions"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type dirProvider struct {
	path string
}

var _ content.Content = &dirProvider{}
var _ extensions.FileSystemProvider = &dirProvider{}
var _ extensions.FSProvider = &dirProvider{}

func Open(ref location.Dir) (*dirProvider, error) {
	return &dirProvider{
		path: ref.Directory,
	}, nil
}

func (p *dirProvider) Close() error {
	return nil
}

func (p *dirProvider) ProvideFileSystem() (filesys.FileSystem, string, error) {
	return filesys.MakeFsOnDisk(), p.path, nil
}

func (p *dirProvider) ProvideFS() (fs.FS, error) {
	return os.DirFS(p.path), nil
}
