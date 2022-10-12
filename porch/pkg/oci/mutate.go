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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strconv"
	"time"

	"github.com/GoogleContainerTools/kpt/pkg/oci"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func (r *ociRepository) CreatePackageRevision(ctx context.Context, obj *api.PackageRevision) (repository.PackageDraft, error) {
	base := empty.Image

	packageName := obj.Spec.PackageName
	ociRepo, err := name.NewRepository(path.Join(r.spec.Registry, packageName))
	if err != nil {
		return nil, err
	}

	meta := &meta.PackageMeta{
		Tasks: []api.Task{},
	}

	// digestName := ImageDigestName{}
	return &ociPackageDraft{
		packageName: packageName,
		parent:      r,
		base:        base,
		tag:         ociRepo.Tag(obj.Spec.Revision),
		meta:        meta,
		lifecycle:   v1alpha1.PackageRevisionLifecycleDraft,
	}, nil
}

func (r *ociRepository) UpdatePackageRevision(ctx context.Context, old repository.PackageRevision) (repository.PackageDraft, error) {
	oldPackage := old.(*ociPackageRevision)
	packageName := oldPackage.packageName
	revision := oldPackage.revision
	// digestName := oldPackage.digestName

	ociRepo, err := name.NewRepository(path.Join(r.spec.Registry, packageName))
	if err != nil {
		return nil, err
	}

	// TODO: Authentication must be set up correctly. Do we use:
	// * Provided Service account?
	// * Workload identity?
	// * Caller credentials (is this possible with k8s apiserver)?
	options := []remote.Option{
		remote.WithAuthFromKeychain(gcrane.Keychain),
		remote.WithContext(ctx),
	}

	ref := ociRepo.Tag(revision)

	base, err := remote.Image(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("error fetching image %q: %w", ref, err)
	}

	newMeta := oldPackage.meta.DeepCopy()
	return &ociPackageDraft{
		packageName: packageName,
		parent:      r,
		base:        base,
		tag:         ociRepo.Tag(revision),
		meta:        newMeta,
	}, nil
}

type ociPackageDraft struct {
	packageName string

	created time.Time

	parent *ociRepository

	base    v1.Image
	tag     name.Tag
	changes []mutation

	lifecycle v1alpha1.PackageRevisionLifecycle
	meta      *meta.PackageMeta
}

type mutation struct {
	Layer v1.Layer
	Task  *api.Task
}

var _ repository.PackageDraft = (*ociPackageDraft)(nil)

func (p *ociPackageDraft) UpdateResources(ctx context.Context, new *api.PackageRevisionResources, task *api.Task) error {
	ctx, span := tracer.Start(ctx, "ociPackageDraft::UpdateResources", trace.WithAttributes())
	defer span.End()

	buf := bytes.NewBuffer(nil)
	writer := tar.NewWriter(buf)

	// TODO: write only changes.
	for k, v := range new.Spec.Resources {
		b := ([]byte)(v)
		blen := len(b)

		if err := writer.WriteHeader(&tar.Header{
			Name:       k,
			Size:       int64(blen),
			Mode:       0644,
			ModTime:    p.created,
			AccessTime: p.created,
			ChangeTime: p.created,
		}); err != nil {
			return fmt.Errorf("failed to write oci package tar header: %w", err)
		}

		if n, err := writer.Write(b); err != nil {
			return fmt.Errorf("failed to write oci package tar contents: %w", err)
		} else if n != blen {
			return fmt.Errorf("failed to write complete oci package tar content: %d of %d", n, blen)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize oci package tar content: %w", err)
	}

	layer := stream.NewLayer(io.NopCloser(buf), stream.WithCompressionLevel(gzip.BestCompression))
	if err := remote.WriteLayer(p.tag.Repository, layer, remote.WithAuthFromKeychain(gcrane.Keychain)); err != nil {
		return fmt.Errorf("failed to write remote layer: %w", err)
	}

	digest, err := layer.Digest()
	if err != nil {
		return fmt.Errorf("failed to get layer digets: %w", err)
	}

	remoteLayer, err := remote.Layer(
		p.tag.Context().Digest(digest.String()),
		remote.WithAuthFromKeychain(gcrane.Keychain))
	if err != nil {
		return fmt.Errorf("failed to create remote layer from digest: %w", err)
	}

	p.changes = append(p.changes, mutation{
		Layer: remoteLayer,
		Task:  task,
	})
	p.meta.Tasks = append(p.meta.Tasks, *task)

	return nil
}

func (p *ociPackageDraft) UpdateLifecycle(ctx context.Context, newLifecycle api.PackageRevisionLifecycle, labels map[string]string, annotations map[string]string) error {
	p.lifecycle = newLifecycle
	p.meta.Labels = labels
	p.meta.Annotations = annotations
	return nil
}

func (p *ociPackageDraft) addEmptyLayer(ctx context.Context) (v1.Layer, error) {
	ctx, span := tracer.Start(ctx, "ociPackageDraft::addEmptyLayer", trace.WithAttributes())
	defer span.End()

	buf := bytes.NewBuffer(nil)
	writer := tar.NewWriter(buf)

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize oci package tar content: %w", err)
	}

	layer := stream.NewLayer(io.NopCloser(buf), stream.WithCompressionLevel(gzip.BestCompression))
	if err := remote.WriteLayer(p.tag.Repository, layer, remote.WithAuthFromKeychain(gcrane.Keychain), remote.WithContext(ctx)); err != nil {
		return nil, fmt.Errorf("failed to write remote layer: %w", err)
	}

	digest, err := layer.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to get layer digets: %w", err)
	}

	remoteLayer, err := remote.Layer(
		p.tag.Context().Digest(digest.String()),
		remote.WithAuthFromKeychain(gcrane.Keychain), remote.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create remote layer from digest: %w", err)
	}

	return remoteLayer, nil
}

// Finish round of updates.
func (p *ociPackageDraft) Close(ctx context.Context) (repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "ociPackageDraft::Close", trace.WithAttributes())
	defer span.End()

	ref := p.tag
	option := remote.WithAuthFromKeychain(gcrane.Keychain)

	klog.Infof("pushing %s", ref)

	if len(p.changes) == 0 {
		p.changes = append(p.changes, mutation{})
	}

	var addendums []mutate.Addendum
	for i, change := range p.changes {
		isLast := len(p.changes) == (i + 1)

		annotation := &meta.Annotation{
			Task: change.Task,
		}

		layer := change.Layer
		if layer == nil {
			l, err := p.addEmptyLayer(ctx)
			if err != nil {
				return nil, err
			}
			layer = l
		}

		addendum := mutate.Addendum{
			Layer: layer,
			History: v1.History{
				Author:  "kpt",
				Created: v1.Time{Time: p.created},
			},
		}

		if isLast {
			if p.lifecycle != "" {
				if addendum.Annotations == nil {
					addendum.Annotations = make(map[string]string)
				}
				addendum.Annotations[annotationKeyLifecycle] = string(p.lifecycle)
			}
			if true {
				// TODO: Lazy?
				annotation.Labels = &p.meta.Labels
				annotation.Annotations = &p.meta.Annotations
			}

		}

		annotationJSON, err := json.Marshal(*annotation)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotation: %w", err)
		}
		// TODO: Use an annotation?
		addendum.History.CreatedBy = "kpt:" + string(annotationJSON)

		addendums = append(addendums, addendum)
	}

	base := p.base
	if base == nil {
		base = empty.Image
	}
	img, err := mutate.Append(base, addendums...)
	if err != nil {
		return nil, fmt.Errorf("failed to append image layers: %w", err)
	}

	// TODO: We have a race condition here; there's no way to indicate that we want to create / not update an existing tag
	if err := remote.Write(ref, img, option); err != nil {
		return nil, fmt.Errorf("failed to push image %s: %w", ref, err)
	}

	// TODO: remote.Write should return the digest etc that was pushed
	desc, err := remote.Get(ref, option)
	if err != nil {
		return nil, fmt.Errorf("error getting metadata for %s: %w", ref, err)
	}
	klog.Infof("desc %s", string(desc.Manifest))

	digestName := oci.ImageDigestName{
		Image:  ref.Name(),
		Digest: desc.Digest.String(),
	}

	configFile, err := p.parent.storage.CachedConfigFile(ctx, digestName)
	if err != nil {
		return nil, fmt.Errorf("error getting config file: %w", err)
	}

	return p.parent.buildPackageRevision(ctx, digestName, p.packageName, p.tag.TagStr(), configFile.Created.Time)
}

func constructResourceVersion(t time.Time) string {
	return strconv.FormatInt(t.UnixNano(), 10)
}

func constructUID(ref string) types.UID {
	return types.UID("uid:" + ref)
}

func (r *ociRepository) DeletePackageRevision(ctx context.Context, old repository.PackageRevision) error {
	oldPackage := old.(*ociPackageRevision)
	packageName := oldPackage.packageName
	revision := oldPackage.revision

	ociRepo, err := name.NewRepository(path.Join(r.spec.Registry, packageName))
	if err != nil {
		return err
	}

	// TODO: Authentication must be set up correctly. Do we use:
	// * Provided Service account?
	// * Workload identity?
	// * Caller credentials (is this possible with k8s apiserver)?
	options := []remote.Option{
		remote.WithAuthFromKeychain(gcrane.Keychain),
		remote.WithContext(ctx),
	}

	ref := ociRepo.Tag(revision)

	klog.Infof("deleting %s", ref)

	if err := remote.Delete(ref, options...); err != nil {
		return fmt.Errorf("error deleting image %q: %w", ref, err)
	}

	return nil
}

func (r *ociRepository) CreatePackage(ctx context.Context, obj *v1alpha1.Package) (repository.Package, error) {
	return nil, fmt.Errorf("CreatePackage not supported for OCI packages")
}

func (r *ociRepository) DeletePackage(ctx context.Context, obj repository.Package) error {
	return fmt.Errorf("DeletePackage not supported for OCI packages")
}
