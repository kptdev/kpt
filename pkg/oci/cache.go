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
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// CachedConfigFile fetches ConfigFile making use of the cache
// TODO: This should be incorporated into the go-containerregistry, along with replacing a fetcher
func (r *Storage) CachedConfigFile(ctx context.Context, imageName imageName) (*v1.ConfigFile, error) {
	ociImage, err := r.ToRemoteImage(ctx, imageName)
	if err != nil {
		return nil, err
	}

	manifest, err := r.CachedManifest(ctx, ociImage)
	if err != nil {
		return nil, fmt.Errorf("error fetching manifest for image: %w", err)
	}

	digest := manifest.Config.Digest

	fetcher := func() (io.ReadCloser, error) {
		b, err := ociImage.RawConfigFile()
		if err != nil {
			return nil, err
		}
		r := bytes.NewReader(b)
		return io.NopCloser(r), nil
	}

	f, err := WithCacheFile(filepath.Join(r.cacheDir, "config", digest.String()), fetcher)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config, err := v1.ParseConfigFile(f)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}
	return config, nil
}

// CachedManifest fetches Manifest making use of the cache
// TODO: This should be incorporated into the go-containerregistry, along with replacing a fetcher
func (r *Storage) CachedManifest(_ context.Context, image v1.Image) (*v1.Manifest, error) {
	digest, err := image.Digest()
	if err != nil {
		return nil, fmt.Errorf("cannot get image manifest: %w", err)
	}

	fetcher := func() (io.ReadCloser, error) {
		b, err := image.RawManifest()
		if err != nil {
			return nil, err
		}
		r := bytes.NewReader(b)
		return io.NopCloser(r), nil
	}

	f, err := WithCacheFile(filepath.Join(r.cacheDir, "manifest", digest.String()), fetcher)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	manifest, err := v1.ParseManifest(f)
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest: %w", err)
	}
	return manifest, nil
}
