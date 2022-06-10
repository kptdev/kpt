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

package git

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5/plumbing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type gitPackageRevision struct {
	parent   *gitRepository
	path     string
	revision string
	updated  time.Time
	ref      *plumbing.Reference // ref is the Git reference at which the package exists
	tree     plumbing.Hash       // Cached tree of the package itself, some descendent of commit.Tree()
	commit   plumbing.Hash       // Current version of the package (commit sha)
	tasks    []v1alpha1.Task
}

var _ repository.PackageRevision = &gitPackageRevision{}

// Kubernetes resource names requirements do not allow to encode arbitrary directory
// path: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
// Because we need a resource names that are stable over time, and avoid conflict, we
// compute a hash of the package path and revision.
// For implementation convenience (though this is temporary) we prepend the repository
// name in order to aide package discovery on the server. With improvements to caching
// layer, the prefix will be removed (this may happen without notice) so it should not
// be relied upon by clients.
func (p *gitPackageRevision) KubeObjectName() string {
	hash := sha1.Sum([]byte(fmt.Sprintf("%s:%s:%s", p.parent.name, p.path, p.revision)))
	return p.parent.name + "-" + hex.EncodeToString(hash[:])
}

func (p *gitPackageRevision) Key() repository.PackageRevisionKey {
	return repository.PackageRevisionKey{
		Repository: p.parent.name,
		Package:    p.path,
		Revision:   p.revision,
	}
}

func (p *gitPackageRevision) uid() types.UID {
	return types.UID(fmt.Sprintf("uid:%s:%s", p.path, p.revision))
}

func (p *gitPackageRevision) GetPackageRevision() *v1alpha1.PackageRevision {
	key := p.Key()

	_, lock, _ := p.GetUpstreamLock()

	// TODO: Use kpt definition of UpstreamLock in the package revision status
	// when https://github.com/GoogleContainerTools/kpt/issues/3297 is complete.
	// Until then, we have to translate from one type to another.
	lockCopy := &v1alpha1.UpstreamLock{
		Type: v1alpha1.OriginType(lock.Type),
		Git: &v1alpha1.GitLock{
			Repo:      lock.Git.Repo,
			Directory: lock.Git.Directory,
			Commit:    lock.Git.Commit,
			Ref:       lock.Git.Ref,
		},
	}

	return &v1alpha1.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.KubeObjectName(),
			Namespace:       p.parent.namespace,
			UID:             p.uid(),
			ResourceVersion: p.commit.String(),
			CreationTimestamp: metav1.Time{
				Time: p.updated,
			},
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    key.Package,
			Revision:       key.Revision,
			RepositoryName: key.Repository,

			Lifecycle: p.Lifecycle(),
			Tasks:     p.tasks,
		},
		Status: v1alpha1.PackageRevisionStatus{
			UpstreamLock: lockCopy,
		},
	}
}

func (p *gitPackageRevision) GetResources(ctx context.Context) (*v1alpha1.PackageRevisionResources, error) {
	resources := map[string]string{}

	tree, err := p.parent.repo.TreeObject(p.tree)
	if err == nil {
		// Files() iterator iterates recursively over all files in the tree.
		fit := tree.Files()
		defer fit.Close()
		for {
			file, err := fit.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, fmt.Errorf("failed to load package resources: %w", err)
			}

			content, err := file.Contents()
			if err != nil {
				return nil, fmt.Errorf("failed to read package file contents: %q, %w", file.Name, err)
			}

			// TODO: decide whether paths should include package directory or not.
			resources[file.Name] = content
			//resources[path.Join(p.path, file.Name)] = content
		}
	}

	key := p.Key()

	return &v1alpha1.PackageRevisionResources{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionResources",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.KubeObjectName(),
			Namespace:       p.parent.namespace,
			UID:             p.uid(),
			ResourceVersion: p.commit.String(),
			CreationTimestamp: metav1.Time{
				Time: p.updated,
			},
			OwnerReferences: []metav1.OwnerReference{}, // TODO: should point to repository resource
		},
		Spec: v1alpha1.PackageRevisionResourcesSpec{
			PackageName:    key.Package,
			Revision:       key.Revision,
			RepositoryName: key.Repository,

			Resources: resources,
		},
	}, nil
}

func (p *gitPackageRevision) GetUpstreamLock() (kptfile.Upstream, kptfile.UpstreamLock, error) {
	resources, err := p.GetResources(context.Background())
	if err != nil {
		return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("cannot determine package lock; cannot retrieve resources: %w", err)
	}

	// get the upstream package URL from the Kptfile
	var kf *kptfile.KptFile
	if contents, found := resources.Spec.Resources[kptfile.KptFileName]; found {
		kf, err = pkg.DecodeKptfile(strings.NewReader(contents))
		if err != nil {
			return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("cannot decode Kptfile: %w", err)
		}
	}

	if kf.Upstream == nil || kf.UpstreamLock == nil || kf.Upstream.Git == nil {
		// upstream information is not in Kptfile yet (this can happen when we clone from the upstream for the first time),
		// so we get it from the parent repo instead.
		return p.findParent()
	}
	return *kf.Upstream, *kf.UpstreamLock, nil
}

func (p *gitPackageRevision) findParent() (kptfile.Upstream, kptfile.UpstreamLock, error) {
	repo, err := p.parent.getRepo()
	if err != nil {
		return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("cannot determine package lock: %w", err)
	}

	if p.ref == nil {
		return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("cannot determine package lock; package has no ref")
	}

	ref, err := refInRemoteFromRefInLocal(p.ref.Name())
	if err != nil {
		return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("cannot determine package lock for %q: %v", p.ref, err)
	}

	return kptfile.Upstream{
			Type: kptfile.GitOrigin,
			Git: &kptfile.Git{
				Repo:      repo,
				Directory: p.path,
				Ref:       ref.Short(),
			},
		}, kptfile.UpstreamLock{
			Type: kptfile.GitOrigin,
			Git: &kptfile.GitLock{
				Repo:      repo,
				Directory: p.path,
				Ref:       ref.Short(),
				Commit:    p.commit.String(),
			},
		}, nil
}

func (p *gitPackageRevision) Lifecycle() v1alpha1.PackageRevisionLifecycle {
	switch ref := p.ref; {
	case ref == nil:
		return v1alpha1.PackageRevisionLifecyclePublished
	case isDraftBranchNameInLocal(ref.Name()):
		return v1alpha1.PackageRevisionLifecycleDraft
	case isProposedBranchNameInLocal(ref.Name()):
		return v1alpha1.PackageRevisionLifecycleProposed
	default:
		return v1alpha1.PackageRevisionLifecyclePublished
	}
}
