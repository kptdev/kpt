// Copyright 2022 The kpt Authors
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

package oci

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"k8s.io/klog/v2"
)

// Storage provides helper functions specifically for OCI storage.
// It abstracts and simplifies the go-containerregistry library, but is agnostic to the contents of the images etc.
type Storage struct {
	imageCache cache.Cache
	cacheDir   string

	transport http.RoundTripper
}

// NewStorage creates a Storage for managing OCI images.
func NewStorage(cacheDir string) (*Storage, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir %q: %w", cacheDir, err)
	}

	c := cache.NewFilesystemCache(filepath.Join(cacheDir, "layers"))

	return &Storage{
		imageCache: c,
		cacheDir:   cacheDir,
		transport:  http.DefaultTransport,
	}, nil
}

// ImageTagName holds an image we know by tag (but tags are mutable, so this often implies lookups)
type ImageTagName struct {
	Image string
	Tag   string
}

func (i ImageTagName) String() string {
	return fmt.Sprintf("%s:%s", i.Image, i.Tag)
}

func (i ImageTagName) ociReference() (name.Reference, error) {
	imageRef, err := name.NewTag(i.Image + ":" + i.Tag)
	if err != nil {
		return nil, fmt.Errorf("cannot parse image name %q: %w", i, err)
	}
	return imageRef, nil
}

func ParseImageTagName(s string) (*ImageTagName, error) {
	t, err := name.NewTag(s)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q as tag: %w", s, err)
	}
	return &ImageTagName{
		Image: t.Repository.Name(),
		Tag:   t.TagStr(),
	}, nil
}

// ImageDigestName holds an image we know by digest (which is immutable and more cacheable)
type ImageDigestName struct {
	Image  string
	Digest string
}

func (i ImageDigestName) String() string {
	return fmt.Sprintf("%s:%s", i.Image, i.Digest)
}

func (i ImageDigestName) ociReference() (name.Reference, error) {
	imageRef, err := name.NewDigest(i.Image + "@" + i.Digest)
	if err != nil {
		return nil, fmt.Errorf("cannot parse image name %q: %w", i, err)
	}
	return imageRef, nil
}

func (r *Storage) CreateOptions(ctx context.Context) []google.Option {
	// TODO: Authentication must be set up correctly. Do we use:
	// * Provided Service account?
	// * Workload identity?
	// * Caller credentials (is this possible with k8s apiserver)?
	return []google.Option{
		google.WithAuthFromKeychain(gcrane.Keychain),
		google.WithContext(ctx),
	}
}

type imageName interface {
	ociReference() (name.Reference, error)
}

// ToRemoteImage builds a remote image reference for the given name, including caching and authentication.
func (r *Storage) ToRemoteImage(ctx context.Context, imageName imageName) (v1.Image, error) {
	// TODO: Authentication must be set up correctly. Do we use:
	// * Provided Service account?
	// * Workload identity?
	// * Caller credentials (is this possible with k8s apiserver)?
	options := []remote.Option{
		remote.WithAuthFromKeychain(gcrane.Keychain),
		remote.WithContext(ctx),
		remote.WithTransport(r.transport),
	}

	imageRef, err := imageName.ociReference()
	if err != nil {
		return nil, err
	}
	// TODO: Can we use a digest to save a lookup?
	// imageRef = imageRef.Context().Digest(digest)

	ociImage, err := remote.Image(imageRef, options...)
	if err != nil {
		return nil, err
	}

	ociImage = cache.Image(ociImage, r.imageCache)

	return ociImage, nil
}

func (r *Storage) GetCacheDir() string {
	return r.cacheDir
}

// WithCacheFile runs with a filesystem-backed cache.
// If cacheFilePath does not exist, it will be fetched with the function fetcher.
// The file contents are then processed with the function reader.
// TODO: We likely need some form of GC/LRU on the cache file paths.
// We can probably use FS access time (or we might need to touch the files when we access them)!
func WithCacheFile(cacheFilePath string, fetcher func() (io.ReadCloser, error)) (io.ReadCloser, error) {
	dir := filepath.Dir(cacheFilePath)

	f, err := os.Open(cacheFilePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("error opening cache file %q: %w", cacheFilePath, err)
		}
	} else {
		// TODO: Delete file if corrupt?
		return f, nil
	}

	r, err := fetcher()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %q: %w", dir, err)
	}

	tempFile, err := os.CreateTemp(dir, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create tempfile in directory %q: %w", dir, err)
	}
	defer func() {
		if tempFile != nil {
			if err := tempFile.Close(); err != nil {
				klog.Warningf("error closing temp file: %v", err)
			}

			if err := os.Remove(tempFile.Name()); err != nil {
				klog.Warningf("failed to write tempfile: %v", err)
			}
		}
	}()
	if _, err := io.Copy(tempFile, r); err != nil {
		return nil, fmt.Errorf("error caching data: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("error closing temp file: %w", err)
	}

	if err := os.Rename(tempFile.Name(), cacheFilePath); err != nil {
		return nil, fmt.Errorf("error renaming temp file %q -> %q: %w", tempFile.Name(), cacheFilePath, err)
	}

	tempFile = nil

	f, err = os.Open(cacheFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening cache file %q (after fetch): %w", cacheFilePath, err)
	}
	return f, nil
}
