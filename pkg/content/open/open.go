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

package open

import (
	"fmt"
	"io/fs"

	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/content/paths"
	"github.com/GoogleContainerTools/kpt/pkg/content/provider/dir"
	"github.com/GoogleContainerTools/kpt/pkg/content/provider/git"
	"github.com/GoogleContainerTools/kpt/pkg/content/provider/oci"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type open struct {
	content.Content
	location.Location
}

func Content(ref location.Reference, opts ...Option) (open, error) {
	opt := makeOptions(opts...)

	switch ref := ref.(type) {
	case location.Dir:
		provider, err := dir.Open(ref)
		if err != nil {
			return open{}, err
		}
		return open{
			Content: provider,
			Location: location.Location{
				Reference: ref,
			},
		}, nil
	case location.GitLock:
	case location.Git:
		provider, lock, err := git.Open(opt.ctx, ref)
		if err != nil {
			return open{}, err
		}
		return open{
			Content: provider,
			Location: location.Location{
				Reference:     ref,
				ReferenceLock: lock,
			},
		}, nil
	case location.OciLock:
	case location.Oci:
		provider, lock, err := oci.Open(ref, remote.WithAuthFromKeychain(gcrane.Keychain))
		if err != nil {
			return open{}, err
		}
		return open{
			Content: provider,
			Location: location.Location{
				Reference:     ref,
				ReferenceLock: lock,
			},
		}, nil
	}
	return open{}, fmt.Errorf("not supported")
}

type openFS struct {
	content.Content
	location.Location
	FS fs.FS
}

func FS(ref location.Reference, opts ...Option) (openFS, error) {
	open, err := Content(ref, opts...)
	if err != nil {
		return openFS{}, err
	}
	fsys, err := content.FS(open.Content)
	if err != nil {
		open.Close()
		return openFS{}, err
	}
	return openFS{open.Content, open.Location, fsys}, nil
}

type openFileSystem struct {
	content.Content
	location.Location
	paths.FileSystemPath
}

func FileSystem(ref location.Reference, opts ...Option) (openFileSystem, error) {
	open, err := Content(ref, opts...)
	if err != nil {
		return openFileSystem{}, err
	}
	path, err := content.FileSystem(open.Content)
	if err != nil {
		open.Close()
		return openFileSystem{}, err
	}
	return openFileSystem{open.Content, open.Location, path}, nil
}

type openReader struct {
	content.Content
	location.Location
	Reader kio.Reader
}

func Reader(ref location.Reference, opts ...Option) (openReader, error) {
	open, err := Content(ref, opts...)
	if err != nil {
		return openReader{}, err
	}
	reader, err := content.Reader(open.Content)
	if err != nil {
		open.Close()
		return openReader{}, err
	}
	return openReader{open.Content, open.Location, reader}, nil
}
