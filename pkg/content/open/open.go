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

package open

import (
	"errors"
	"fmt"
	"io"
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

var (
	ErrUnknownReference = errors.New("unknown reference type")
)

type ContentResult struct {
	io.Closer
	Content       content.Content
	Reference     location.Reference
	ReferenceLock location.ReferenceLock
}

// Content returns an initialized content.Content for the given
// location.Reference address. The resolved location.ReferenceLock address
// is also returned for locations that support immutable history.
// Close must be called on the result Content to ensure proper cleanup of
// temporary resources.
func Content(ref location.Reference, opts ...Option) (ContentResult, error) {
	opt := makeOptions(opts...)

	for _, provider := range opt.providers {
		content, refNew, refLock, err := provider.Content(opt.ctx, ref)
		if err != nil {
			if errors.Is(err, ErrUnknownReference) {
				continue
			}
			return ContentResult{}, fmt.Errorf("error opening content %q: %w", ref, err)
		}
		return ContentResult{
			Closer:        content,
			Content:       content,
			Reference:     refNew,
			ReferenceLock: refLock,
		}, nil
	}

	switch ref := ref.(type) {
	case location.Dir:
		c, err := dir.Open(ref)
		if err != nil {
			return ContentResult{}, err
		}
		return ContentResult{
			Closer:    c,
			Content:   c,
			Reference: ref,
		}, nil
	case location.Git:
		c, lock, err := git.Open(opt.ctx, ref)
		if err != nil {
			return ContentResult{}, err
		}
		return ContentResult{
			Closer:        c,
			Content:       c,
			Reference:     ref,
			ReferenceLock: lock,
		}, nil
	case location.GitLock:
		c, lock, err := git.OpenLock(opt.ctx, ref)
		if err != nil {
			return ContentResult{}, err
		}
		return ContentResult{
			Closer:        c,
			Content:       c,
			Reference:     ref,
			ReferenceLock: lock,
		}, nil
	case location.Oci:
		// TODO(https://github.com/GoogleContainerTools/kpt/issues/2765) pass through options for build-in openers like oci

		c, lock, err := oci.Open(
			ref,
			remote.WithAuthFromKeychain(gcrane.Keychain),
			remote.WithContext(opt.ctx))
		if err != nil {
			return ContentResult{}, err
		}
		return ContentResult{
			Closer:        c,
			Content:       c,
			Reference:     ref,
			ReferenceLock: lock,
		}, nil
	case location.OciLock:
		c, lock, err := oci.OpenLock(
			ref,
			remote.WithAuthFromKeychain(gcrane.Keychain),
			remote.WithContext(opt.ctx))
		if err != nil {
			return ContentResult{}, err
		}
		return ContentResult{
			Closer:        c,
			Content:       c,
			Reference:     ref,
			ReferenceLock: lock,
		}, nil
	}

	// TODO(https://github.com/GoogleContainerTools/kpt/issues/2765) have an additional case
	// in the switch for custom Reference types that implement a ContentOpener extension

	return ContentResult{}, fmt.Errorf("error opening content %q: %w", ref, ErrUnknownReference)
}

type FSResult struct {
	io.Closer
	Content       content.Content
	Reference     location.Reference
	ReferenceLock location.ReferenceLock
	FS            fs.FS
}

func FS(ref location.Reference, opts ...Option) (FSResult, error) {
	r, err := Content(ref, opts...)
	if err != nil {
		return FSResult{}, err
	}
	fsys, err := content.FS(r.Content)
	if err != nil {
		r.Content.Close()
		return FSResult{}, err
	}
	return FSResult{
		Closer:        r.Closer,
		Content:       r.Content,
		Reference:     r.Reference,
		ReferenceLock: r.ReferenceLock,
		FS:            fsys,
	}, nil
}

type FileSystemResult struct {
	io.Closer
	Content       content.Content
	Reference     location.Reference
	ReferenceLock location.ReferenceLock
	paths.FileSystemPath
}

func FileSystem(ref location.Reference, opts ...Option) (FileSystemResult, error) {
	r, err := Content(ref, opts...)
	if err != nil {
		return FileSystemResult{}, err
	}
	fsp, err := content.FileSystem(r.Content)
	if err != nil {
		r.Content.Close()
		return FileSystemResult{}, err
	}
	return FileSystemResult{
		Closer:         r.Closer,
		Content:        r.Content,
		Reference:      r.Reference,
		ReferenceLock:  r.ReferenceLock,
		FileSystemPath: fsp,
	}, nil
}

type ReaderResult struct {
	io.Closer
	Content       content.Content
	Reference     location.Reference
	ReferenceLock location.ReferenceLock
	Reader        kio.Reader
}

func Reader(ref location.Reference, opts ...Option) (ReaderResult, error) {
	r, err := Content(ref, opts...)
	if err != nil {
		return ReaderResult{}, err
	}
	reader, err := content.Reader(r.Content)
	if err != nil {
		r.Close()
		return ReaderResult{}, err
	}
	return ReaderResult{
		Closer:        r.Closer,
		Content:       r.Content,
		Reference:     r.Reference,
		ReferenceLock: r.ReferenceLock,
		Reader:        reader,
	}, nil
}
