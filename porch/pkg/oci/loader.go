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
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/oci"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

const (
	annotationKeyLifecycle = "kpt.dev/lifecycle"
	annotationKeyRevision  = "kpt.dev/revision"
)

var tracer = otel.Tracer("oci")

func (r *ociRepository) getLifecycle(ctx context.Context, imageRef oci.ImageDigestName) (v1alpha1.PackageRevisionLifecycle, error) {
	ctx, span := tracer.Start(ctx, "ociRepository::getLifecycle", trace.WithAttributes(
		attribute.Stringer("image", imageRef),
	))
	defer span.End()

	ociImage, err := r.storage.ToRemoteImage(ctx, imageRef)
	if err != nil {
		return "", err
	}

	manifest, err := r.storage.CachedManifest(ctx, ociImage)
	if err != nil {
		return "", fmt.Errorf("error fetching manifest for image: %w", err)
	}

	lifecycleValue := manifest.Annotations[annotationKeyLifecycle]
	switch lifecycleValue {
	case "", string(v1alpha1.PackageRevisionLifecycleDraft):
		return v1alpha1.PackageRevisionLifecycleDraft, nil
	case string(v1alpha1.PackageRevisionLifecycleProposed):
		return v1alpha1.PackageRevisionLifecycleProposed, nil
	case string(v1alpha1.PackageRevisionLifecyclePublished):
		return v1alpha1.PackageRevisionLifecyclePublished, nil
	case string(v1alpha1.PackageRevisionLifecycleDeletionProposed):
		return v1alpha1.PackageRevisionLifecycleDeletionProposed, nil
	default:
		return "", fmt.Errorf("unknown package revision lifecycle %q", lifecycleValue)
	}
}

func (r *ociRepository) getRevisionNumber(ctx context.Context, imageRef oci.ImageDigestName) (string, error) {
	ctx, span := tracer.Start(ctx, "ociRepository::getRevision", trace.WithAttributes(
		attribute.Stringer("image", imageRef),
	))
	defer span.End()

	ociImage, err := r.storage.ToRemoteImage(ctx, imageRef)
	if err != nil {
		return "", err
	}

	manifest, err := r.storage.CachedManifest(ctx, ociImage)
	if err != nil {
		return "", fmt.Errorf("error fetching manifest for image: %w", err)
	}

	return manifest.Annotations[annotationKeyRevision], nil
}

func (r *ociRepository) loadTasks(ctx context.Context, imageRef oci.ImageDigestName) ([]api.Task, error) {
	ctx, span := tracer.Start(ctx, "ociRepository::loadTasks", trace.WithAttributes(
		attribute.Stringer("image", imageRef),
	))
	defer span.End()

	configFile, err := r.storage.CachedConfigFile(ctx, imageRef)
	if err != nil {
		return nil, fmt.Errorf("error fetching config for image: %w", err)
	}

	var tasks []api.Task
	for i := range configFile.History {
		history := &configFile.History[i]
		command := history.CreatedBy
		if strings.HasPrefix(command, "kpt:") {
			task := api.Task{}
			b := []byte(strings.TrimPrefix(command, "kpt:"))
			if err := json.Unmarshal(b, &task); err != nil {
				klog.Warningf("failed to unmarshal task command %q: %w", command, err)
				continue
			}
			tasks = append(tasks, task)
		} else {
			klog.Warningf("unknown task command in history %q", command)
		}
	}

	return tasks, nil
}

func LookupImageTag(ctx context.Context, s *oci.Storage, imageName oci.ImageTagName) (*oci.ImageDigestName, error) {
	ctx, span := tracer.Start(ctx, "Storage::LookupImageTag", trace.WithAttributes(
		attribute.Stringer("image", imageName),
	))
	defer span.End()

	ociImage, err := s.ToRemoteImage(ctx, imageName)
	if err != nil {
		return nil, err
	}

	digest, err := ociImage.Digest()
	if err != nil {
		return nil, err
	}

	return &oci.ImageDigestName{
		Image:  imageName.Image,
		Digest: digest.String(),
	}, nil
}

func LoadResources(ctx context.Context, s *oci.Storage, imageName *oci.ImageDigestName) (*repository.PackageResources, error) {
	ctx, span := tracer.Start(ctx, "Storage::LoadResources", trace.WithAttributes(
		attribute.Stringer("image", imageName),
	))
	defer span.End()

	if imageName.Digest == "" {
		// New package; no digest yet
		return &repository.PackageResources{
			Contents: map[string]string{},
		}, nil
	}

	fetcher := func() (io.ReadCloser, error) {
		ociImage, err := s.ToRemoteImage(ctx, imageName)
		if err != nil {
			return nil, err
		}

		reader := mutate.Extract(ociImage)
		return reader, nil
	}

	// We need the per-digest cache here because otherwise we have to make a network request to look up the manifest in remote.Image
	// (this could be cached by the go-containerregistry library, for some reason it is not...)
	// TODO: Is there then any real reason to _also_ have the image-layer cache?
	f, err := oci.WithCacheFile(filepath.Join(s.GetCacheDir(), "resources", imageName.Digest), fetcher)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tarReader := tar.NewReader(f)

	// TODO: Check hash here?  Or otherwise handle error?
	resources, err := loadResourcesFromTar(ctx, tarReader)
	if err != nil {
		return nil, err
	}
	return resources, nil
}

func loadResourcesFromTar(ctx context.Context, tarReader *tar.Reader) (*repository.PackageResources, error) {
	resources := &repository.PackageResources{
		Contents: map[string]string{},
	}

	for {
		hdr, err := tarReader.Next()
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
			b, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("error reading %q from image: %w", path, err)
			}
			resources.Contents[path] = string(b)

		default:
			return nil, fmt.Errorf("package cannot unsupported entry type for %q (%v)", path, fileType)

		}
	}

	return resources, nil
}
