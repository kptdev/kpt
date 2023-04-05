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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

// Registry manages a single oci registry.  Our registries serve a single named image, with multiple tags.
type Registry struct {
	name    string
	rootDir string

	uploadsDir   string
	manifestsDir string
	blobsDir     string

	// Basic auth
	username string
	password string
}

// metaDir is how we differentiate the image data from a child image directory
// (e.g. foo/_image_/blobs holds the blobs for foo, foo/bar/_image_/blobs holds the blobs for foo/bar)
const metaDir = "_image_"

// NewRegistry constructs an instance of Registry
func NewRegistry(name string, rootDir string, options ...RegistryOption) (*Registry, error) {
	r := &Registry{
		name:    name,
		rootDir: rootDir,
	}
	for _, option := range options {
		if err := option.apply(r); err != nil {
			return nil, err
		}
	}

	r.manifestsDir = filepath.Join(rootDir, metaDir, "manifests")
	if err := os.MkdirAll(r.manifestsDir, 0755); err != nil {
		return nil, err
	}

	r.blobsDir = filepath.Join(rootDir, metaDir, "blobs")
	if err := os.MkdirAll(r.blobsDir, 0755); err != nil {
		return nil, err
	}

	r.uploadsDir = filepath.Join(rootDir, metaDir, "uploads")
	if err := os.MkdirAll(r.uploadsDir, 0755); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Registry) CreateManifest(ctx context.Context, tag string, b []byte) error {
	{
		p := filepath.Join(r.manifestsDir, tag)

		if err := os.WriteFile(p, b, 0644); err != nil {
			return err
		}
	}

	// TODO: Likely we want to write the tags in a different way

	hash := sha256.Sum256(b)
	ref := "sha256:" + hex.EncodeToString(hash[:])

	{
		p := filepath.Join(r.manifestsDir, ref)

		if err := os.WriteFile(p, b, 0644); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) ReadManifest(ctx context.Context, tag string) ([]byte, error) {
	p := filepath.Join(r.manifestsDir, tag)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Registry) StartUpload(ctx context.Context) (string, error) {
	id := uuid.New().String()

	p := filepath.Join(r.uploadsDir, id)

	f, err := os.Create(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return id, nil
}

func (r *Registry) UploadPosition(ctx context.Context, uuid string) (int64, error) {
	// TODO: Verify uuid
	p := filepath.Join(r.uploadsDir, uuid)

	stat, err := os.Stat(p)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

func (r *Registry) AppendUpload(ctx context.Context, uuid string, pos int64, in io.Reader) (int64, error) {
	// TODO: Verify uuid
	p := filepath.Join(r.uploadsDir, uuid)

	f, err := os.OpenFile(p, os.O_RDWR, 0666)
	if err != nil {
		return 0, err
	}
	shouldClose := true
	defer func() {
		if shouldClose {
			f.Close()
		}
	}()

	endPos, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("error seeking to end: %w", err)
	}
	if pos != endPos {
		return 0, fmt.Errorf("position is not end of file (%d vs %d)", endPos, pos)
	}

	n, err := io.Copy(f, in)
	if err != nil {
		return 0, fmt.Errorf("error writing content to upload file: %w", err)
	}

	if err := f.Close(); err != nil {
		return n, fmt.Errorf("error closing uploaded file: %w", err)
	}
	shouldClose = false

	return n, nil
}

func (r *Registry) CompleteUpload(ctx context.Context, uuid string, digest string) (string, error) {
	log := klog.FromContext(ctx)

	// TODO: Verify uuid
	p := filepath.Join(r.uploadsDir, uuid)

	f, err := os.OpenFile(p, os.O_RDONLY, 0666)
	if err != nil {
		return "", err
	}
	shouldClose := true
	defer func() {
		if shouldClose {
			f.Close()
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("error hashing uploaded file: %w", err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("error closing uploaded file: %w", err)
	}
	shouldClose = false

	actualHash := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualHash != digest {
		// TODO: return bad request or similar?
		return "", fmt.Errorf("digest mismatch; disk hash is %q, request hash is  %q", actualHash, digest)
	}

	blobPath := filepath.Join(r.blobsDir, actualHash)
	if err := os.Rename(p, blobPath); err != nil {
		return "", fmt.Errorf("error renaming uploaded file %q -> %q: %w", p, blobPath, err)
	}
	log.Info("created blob", "path", blobPath)

	return actualHash, nil
}

// ServeBlob returns the HTTP response for a blob.
// The return type is optimized for serving for HTTP (we might want an internal method in the future)
func (r *Registry) ServeBlob(ctx context.Context, blob string) (Response, error) {
	// TODO: Verify blob

	blobPath := filepath.Join(r.blobsDir, blob)

	stat, err := os.Stat(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			klog.Infof("file %q not file", blobPath)
			return ErrorResponse(http.StatusNotFound), nil
		}
		return nil, err
	}

	// TODO: Additional headers

	return &FileResponse{Stat: stat, Path: blobPath, ContentType: "application/octet-stream"}, nil
}

type Tags struct {
	Name      string                  `json:"name"`
	Manifests map[string]ManifestInfo `json:"manifest"`
	Tags      []string                `json:"tags"`
	Children  []string                `json:"child"`
}

type ManifestInfo struct {
	ImageSizeBytes string   `json:"imageSizeBytes"`
	MediaType      string   `json:"mediaType"`
	TimeCreatedMs  string   `json:"timeCreatedMs"`
	TimeUploadedMS string   `json:"timeUploadedMs"`
	Tags           []string `json:"tag"`
}

func (r *Registry) ListTags(ctx context.Context) (*Tags, error) {
	tags := &Tags{Name: r.name}

	// TODO: We may need to optimize how manifests are stored as this is pretty expensive
	includeManifestsInResponse := true

	// gcr includes children (other registries)
	includeChildrenInResponse := true

	if includeManifestsInResponse {
		tags.Manifests = make(map[string]ManifestInfo)
	}

	{
		files, err := os.ReadDir(r.manifestsDir)
		if err != nil {
			return nil, fmt.Errorf("error reading tags directory %q: %w", r.manifestsDir, err)
		}
		for _, f := range files {
			tag := f.Name()
			if !strings.HasPrefix(tag, "sha256:") {
				tags.Tags = append(tags.Tags, tag)
			}

			if includeManifestsInResponse {
				p := filepath.Join(r.manifestsDir, tag)
				b, err := os.ReadFile(p)
				if err != nil {
					return nil, fmt.Errorf("error reading %q: %w", p, err)
				}
				stat, err := os.Stat(p)
				if err != nil {
					return nil, fmt.Errorf("error from os.Stat(%q): %w", p, err)
				}
				unixStat := stat.Sys().(*syscall.Stat_t)

				hash := sha256.Sum256(b)
				ref := "sha256:" + hex.EncodeToString(hash[:])
				existing, found := tags.Manifests[ref]
				if !found {
					existing = ManifestInfo{}
					ctime := time.Unix(int64(unixStat.Ctim.Sec), int64(unixStat.Ctim.Nsec))
					// TODO: What should these values really be?
					existing.TimeCreatedMs = strconv.FormatInt(ctime.UnixMilli(), 10)
					existing.TimeUploadedMS = strconv.FormatInt(ctime.UnixMilli(), 10)
					// TODO: ImageSizeBytes
					// TODO: MediaType
				}
				if !strings.HasPrefix(tag, "sha256:") {
					existing.Tags = append(existing.Tags, tag)
				}
				tags.Manifests[ref] = existing
			}
		}
	}

	if includeChildrenInResponse {
		files, err := os.ReadDir(r.rootDir)
		if err != nil {
			return nil, fmt.Errorf("error reading directory %q: %w", r.rootDir, err)
		}
		for _, f := range files {
			if f.IsDir() {
				name := f.Name()
				if name != metaDir {
					tags.Children = append(tags.Children, name)
				}
			}
		}
	}

	return tags, nil
}

// RegistryOption is implemented by configuration settings for git repository.
type RegistryOption interface {
	apply(*Registry) error
}

type optionBasicAuth struct {
	username, password string
}

func (o *optionBasicAuth) apply(s *Registry) error {
	s.username, s.password = o.username, o.password
	return nil
}

func WithBasicAuth(username, password string) RegistryOption {
	return &optionBasicAuth{
		username: username,
		password: password,
	}
}
