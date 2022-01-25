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

package oci

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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

	cache := cache.NewFilesystemCache(filepath.Join(cacheDir, "layers"))

	return &Storage{
		imageCache: cache,
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

func (r *Storage) createOptions(ctx context.Context) []google.Option {
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

// toRemoteImage builds a remote image reference for the given name, including caching and authentication.
func (r *Storage) toRemoteImage(ctx context.Context, imageName imageName) (v1.Image, error) {
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
	//imageRef = imageRef.Context().Digest(digest)

	ociImage, err := remote.Image(imageRef, options...)
	if err != nil {
		return nil, err
	}

	ociImage = cache.Image(ociImage, r.imageCache)

	return ociImage, nil
}
