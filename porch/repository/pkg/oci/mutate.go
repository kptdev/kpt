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

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

func (r *ociRepository) CreatePackageRevision(ctx context.Context, obj *api.PackageRevision) (repository.PackageDraft, error) {
	base := empty.Image
	return r.createDraft(
		obj.Spec.PackageName,
		obj.Spec.Revision,
		base,
		ImageDigestName{},
	)
}

func (r *ociRepository) UpdatePackage(ctx context.Context, old repository.PackageRevision) (repository.PackageDraft, error) {
	oldPackage := old.(*ociPackageRevision)
	packageName := oldPackage.packageName
	revision := oldPackage.revision
	digestName := oldPackage.digestName

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

	return r.createDraft(
		packageName,
		revision,
		base,
		digestName,
	)
}

func (r *ociRepository) createDraft(packageName, revision string, base v1.Image, digestName ImageDigestName) (*ociPackageDraft, error) {
	ociRepo, err := name.NewRepository(path.Join(r.spec.Registry, packageName))
	if err != nil {
		return nil, err
	}

	return &ociPackageDraft{
		packageName: packageName,
		parent:      r,
		tasks:       []api.Task{},
		base:        base,
		tag:         ociRepo.Tag(revision),
	}, nil
}

type ociPackageDraft struct {
	packageName string

	created time.Time

	parent *ociRepository

	tasks []v1alpha1.Task

	base      v1.Image
	tag       name.Tag
	addendums []mutate.Addendum
}

var _ repository.PackageDraft = (*ociPackageDraft)(nil)

func (p *ociPackageDraft) UpdateResources(ctx context.Context, new *api.PackageRevisionResources, task *api.Task) error {
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

	taskJSON, err := json.Marshal(*task)
	if err != nil {
		return fmt.Errorf("failed to marshal task %T to json: %w", task, err)
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

	p.addendums = append(p.addendums, mutate.Addendum{
		Layer: remoteLayer,
		History: v1.History{
			Author:    "kool kat",
			Created:   v1.Time{Time: p.created},
			CreatedBy: "kpt:" + string(taskJSON),
		},
	})

	p.tasks = append(p.tasks, *task)

	return nil
}

// Finish round of updates.
func (p *ociPackageDraft) Close(ctx context.Context) (repository.PackageRevision, error) {
	ref := p.tag
	option := remote.WithAuthFromKeychain(gcrane.Keychain)

	klog.Infof("pushing %s", ref)

	img, err := mutate.Append(p.base, p.addendums...)
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

	digestName := ImageDigestName{
		Image:  ref.Name(),
		Digest: desc.Digest.String(),
	}

	configFile, err := p.parent.storage.cachedConfigFile(ctx, digestName)
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
