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

package wasm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/v1/match"

	"github.com/GoogleContainerTools/kpt/pkg/oci"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type Client struct {
	*oci.Storage
}

func NewClient(cacheDir string) (*Client, error) {
	store, err := oci.NewStorage(cacheDir)
	if err != nil {
		return nil, err
	}
	return &Client{Storage: store}, nil
}

func (r *Client) PushWasm(ctx context.Context, wasmFile string, imageName string) error {
	tag, err := name.NewTag(imageName)
	if err != nil {
		return fmt.Errorf("unable to parse tag %q: %v", imageName, err)
	}

	options := []remote.Option{
		remote.WithAuthFromKeychain(gcrane.Keychain),
		remote.WithContext(ctx),
	}

	rmt, err := remote.Get(tag, options...)
	if err != nil && !isManifestNotFoundErr(err) {
		return fmt.Errorf("unexpected error when trying to check the remote registry: %w", err)
	}

	// If there is an existing remote image, we ensure its media type is an image index type.
	// TODO: if the media type is not an image index type, we can do something smarter than
	// simply failing.
	if err == nil {
		if rmt.MediaType != types.OCIImageIndex && rmt.MediaType != types.DockerManifestList {
			return fmt.Errorf("unexpected media type: %s, expect either %s or %s", rmt.MediaType, types.OCIImageIndex, types.DockerManifestList)
		}
	}

	// Compress the wasm file.
	tarReader, err := wasmFileToTar(wasmFile)
	if err != nil {
		return fmt.Errorf("unable to compress %v: %w", wasmFile, err)
	}

	// Construct the image by building its layer.
	layer := stream.NewLayer(io.NopCloser(tarReader), stream.WithCompressionLevel(gzip.BestCompression))
	if err = remote.WriteLayer(tag.Repository, layer, options...); err != nil {
		return fmt.Errorf("failed to write remote layer: %w", err)
	}
	img, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		return fmt.Errorf("failed to append image layers: %w", err)
	}
	img = mutate.MediaType(img, types.DockerManifestSchema2)
	img, err = mutate.CreatedAt(img, v1.Time{Time: time.Now()})
	if err != nil {
		return fmt.Errorf("failed to set created time for the image: %w", err)
	}
	hash, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to get digest of the image: %w", err)
	}

	// Push the image to remote
	if err = remote.Write(tag.Repository.Digest(hash.String()), img, options...); err != nil {
		return fmt.Errorf("failed to push image %s: %w", tag, err)
	}
	fmt.Printf("digest of the image: %v\n", hash.String())

	// Construct the image index.
	var index v1.ImageIndex = empty.Index
	if rmt != nil {
		index, err = rmt.ImageIndex()
		if err != nil {
			return fmt.Errorf("unable to parse an image as image index: %w", err)
		}
	}

	wasmPlatform := v1.Platform{
		Architecture: "wasm",
		OS:           "js",
	}
	index = mutate.RemoveManifests(index, match.Platforms(wasmPlatform))
	index = mutate.AppendManifests(index, mutate.IndexAddendum{
		Add: img,
		Descriptor: v1.Descriptor{
			MediaType: "application/vnd.docker.distribution.manifest.v2+json",
			Digest:    hash,
			Platform:  &wasmPlatform,
		},
	})
	index = mutate.IndexMediaType(index, types.DockerManifestList)

	// Push the image index to remote.
	err = remote.WriteIndex(tag, index, options...)
	if err != nil {
		return fmt.Errorf("unable to write image index: %w", err)
	}
	indexHash, err := index.Digest()
	if err != nil {
		return fmt.Errorf("unable to get digest for the image index: %w", err)
	}
	fmt.Printf("digest of the image index: %v\n", indexHash.String())
	return nil
}

func wasmFileToTar(filename string) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	writer := tar.NewWriter(buf)

	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read from file %v: %w", filename, err)
	}
	blen := len(b)

	if err = writer.WriteHeader(&tar.Header{
		Name: filepath.Base(filename),
		Size: int64(blen),
		Mode: 0644,
	}); err != nil {
		return nil, fmt.Errorf("failed to write tar header: %w", err)
	}

	if _, err = writer.Write(b); err != nil {
		return nil, fmt.Errorf("failed to write tar contents: %w", err)
	}
	return buf, nil
}

func (r *Client) LoadWasm(ctx context.Context, imageName string) (io.ReadCloser, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, err
	}

	fetcher := func() (io.ReadCloser, error) {
		options := []remote.Option{
			remote.WithContext(ctx),
			remote.WithAuthFromKeychain(gcrane.Keychain),
			remote.WithPlatform(v1.Platform{
				Architecture: "wasm",
				OS:           "js",
			}),
		}
		ociImage, err := remote.Image(ref, options...)
		if err != nil {
			return nil, fmt.Errorf("unable to get remote image: %w", err)
		}

		reader := mutate.Extract(ociImage)
		return reader, nil
	}

	// We need the per-digest cache here because otherwise we have to make a network request to look up the manifest in remote.Image
	// (this could be cached by the go-containerregistry library, for some reason it is not...)
	// TODO: Is there then any real reason to _also_ have the image-layer cache?
	f, err := oci.WithCacheFile(filepath.Join(r.GetCacheDir(), "wasm", ref.String()), fetcher)
	if err != nil {
		return nil, err
	}

	wrapper := &tarReadCloser{
		Reader: tar.NewReader(f),
		closer: f,
	}

	wasmBytesReader, err := loadWasmFromTar(wrapper)
	if err != nil {
		return nil, err
	}
	return wasmBytesReader, nil
}

type tarReadCloser struct {
	*tar.Reader
	closer io.Closer
}

func (w *tarReadCloser) Close() error {
	return w.closer.Close()
}

func loadWasmFromTar(tarReadCloser *tarReadCloser) (io.ReadCloser, error) {
	for {
		hdr, err := tarReadCloser.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		path := hdr.Name
		fileType := hdr.FileInfo().Mode().Type()
		switch fileType {
		case fs.ModeDir:
			// Ignored
		case fs.ModeSymlink:
			// We probably don't want to support this; feels high-risk, low-reward
			return nil, fmt.Errorf("package cannot contain symlink (%q)", path)
		case 0:
			// there is only one file in it
			return tarReadCloser, nil
		default:
			return nil, fmt.Errorf("package cannot unsupported entry type for %q (%v)", path, fileType)
		}
	}

	return nil, nil
}

func isManifestNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	var terr *transport.Error
	isTransportError := errors.As(err, &terr)
	if !isTransportError {
		return false
	}
	if terr.StatusCode != http.StatusNotFound {
		return false
	}
	foundManifestUnknown := false
	for _, e := range terr.Errors {
		if e.Code == transport.ManifestUnknownErrorCode {
			foundManifestUnknown = true
		}
	}
	return foundManifestUnknown
}
