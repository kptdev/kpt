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

type tempProvider struct {
	dirProvider
}

var _ content.Content = &dirProvider{}
var _ extensions.FileSystemProvider = &dirProvider{}
var _ extensions.FSProvider = &dirProvider{}

func Open(ref location.Dir) (*dirProvider, error) {
	return &dirProvider{
		path: ref.Directory,
	}, nil
}

// TempResult is the return value of the dir.Temp function
type TempResult struct {
	// Content is the open temporary directory. A call to
	// content.FileSystem() function may be used to access
	// the information it contains.
	//
	// Close must be called on the Content instance to remove
	// the temporary directory.
	content.Content

	// AbsolutePath is the actual location of the
	// temporary directory. It may be used as a normal
	// path in OS api calls and passed to other processes
	// in path arguments.
	AbsolutePath string
}

// Temp creates a temporary directory by calling
// os.MkdirTemp and returns the open content.Content
// as well as the AbsolutePath.
//
// Close() must be called on the returned Content
// in order to remove the temporary directory.
func Temp(pattern string) (TempResult, error) {
	path, err := os.MkdirTemp("", pattern)
	if err != nil {
		return TempResult{}, err
	}
	temp := &tempProvider{
		dirProvider{
			path: path,
		},
	}
	return TempResult{
		Content:      temp,
		AbsolutePath: path,
	}, nil
}

func (p *dirProvider) Close() error {
	return nil
}

func (p *tempProvider) Close() error {
	return os.RemoveAll(p.path)
}

func (p *dirProvider) FileSystem() (filesys.FileSystem, string, error) {
	return filesys.MakeFsOnDisk(), p.path, nil
}

func (p *dirProvider) FS() (fs.FS, error) {
	return os.DirFS(p.path), nil
}
