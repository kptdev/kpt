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
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type gitPackageRevision struct {
	repo          *gitRepository // repo is repo containing the package
	path          string         // the path to the package from the repo root
	revision      string
	workspaceName v1alpha1.WorkspaceName
	updated       time.Time
	updatedBy     string
	ref           *plumbing.Reference // ref is the Git reference at which the package exists
	tree          plumbing.Hash       // Cached tree of the package itself, some descendent of commit.Tree()
	commit        plumbing.Hash       // Current version of the package (commit sha)
	tasks         []v1alpha1.Task
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
	// The published package revisions on the main branch will have the same workspaceName
	// as the most recently published package revision, so we need to ensure it has a unique
	// and unchanging name.
	var s string
	if p.revision == string(p.repo.branch) {
		s = p.revision
	} else {
		s = string(p.workspaceName)
	}
	hash := sha1.Sum([]byte(fmt.Sprintf("%s:%s:%s", p.repo.name, p.path, s)))
	return p.repo.name + "-" + hex.EncodeToString(hash[:])
}

func (p *gitPackageRevision) KubeObjectNamespace() string {
	return p.repo.namespace
}

func (p *gitPackageRevision) UID() types.UID {
	return p.uid()
}

func (p *gitPackageRevision) Key() repository.PackageRevisionKey {
	// if the repository has been registered with a directory, then the
	// package name is the package path relative to the registered directory
	packageName := p.path
	if p.repo.directory != "" {
		pn, err := filepath.Rel(p.repo.directory, packageName)
		if err != nil {
			klog.Errorf("error computing package name relative to registered directory: %w", err)
		}
		packageName = strings.TrimLeft(pn, "./")
	}

	return repository.PackageRevisionKey{
		Repository:    p.repo.name,
		Package:       packageName,
		Revision:      p.revision,
		WorkspaceName: p.workspaceName,
	}
}

func (p *gitPackageRevision) uid() types.UID {
	var s string
	if p.revision == string(p.repo.branch) {
		s = p.revision
	} else {
		s = string(p.workspaceName)
	}
	return types.UID(fmt.Sprintf("uid:%s:%s", p.path, s))
}

func (p *gitPackageRevision) GetPackageRevision(ctx context.Context) (*v1alpha1.PackageRevision, error) {
	key := p.Key()

	_, lock, _ := p.GetUpstreamLock(ctx)
	lockCopy := &v1alpha1.UpstreamLock{}

	// TODO: Use kpt definition of UpstreamLock in the package revision status
	// when https://github.com/GoogleContainerTools/kpt/issues/3297 is complete.
	// Until then, we have to translate from one type to another.
	if lock.Git != nil {
		lockCopy = &v1alpha1.UpstreamLock{
			Type: v1alpha1.OriginType(lock.Type),
			Git: &v1alpha1.GitLock{
				Repo:      lock.Git.Repo,
				Directory: lock.Git.Directory,
				Commit:    lock.Git.Commit,
				Ref:       lock.Git.Ref,
			},
		}
	}

	kf, _ := p.GetKptfile(ctx)

	status := v1alpha1.PackageRevisionStatus{
		UpstreamLock: lockCopy,
		Deployment:   p.repo.deployment,
		Conditions:   repository.ToApiConditions(kf),
	}

	if v1alpha1.LifecycleIsPublished(p.Lifecycle()) {
		if !p.updated.IsZero() {
			status.PublishedAt = metav1.Time{Time: p.updated}
		}
		if p.updatedBy != "" {
			status.PublishedBy = p.updatedBy
		}
	}

	return &v1alpha1.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.KubeObjectName(),
			Namespace:       p.repo.namespace,
			UID:             p.uid(),
			ResourceVersion: p.commit.String(),
			CreationTimestamp: metav1.Time{
				Time: p.updated,
			},
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    key.Package,
			RepositoryName: key.Repository,
			Lifecycle:      p.Lifecycle(),
			Tasks:          p.tasks,
			ReadinessGates: repository.ToApiReadinessGates(kf),
			WorkspaceName:  key.WorkspaceName,
			Revision:       key.Revision,
		},
		Status: status,
	}, nil
}

func (p *gitPackageRevision) GetResources(ctx context.Context) (*v1alpha1.PackageRevisionResources, error) {
	resources, err := p.repo.getResources(p.tree)
	if err != nil {
		return nil, fmt.Errorf("failed to load package resources: %w", err)
	}

	key := p.Key()

	return &v1alpha1.PackageRevisionResources{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionResources",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.KubeObjectName(),
			Namespace:       p.repo.namespace,
			UID:             p.uid(),
			ResourceVersion: p.commit.String(),
			CreationTimestamp: metav1.Time{
				Time: p.updated,
			},
			OwnerReferences: []metav1.OwnerReference{}, // TODO: should point to repository resource
		},
		Spec: v1alpha1.PackageRevisionResourcesSpec{
			PackageName:    key.Package,
			WorkspaceName:  key.WorkspaceName,
			Revision:       key.Revision,
			RepositoryName: key.Repository,

			Resources: resources,
		},
	}, nil
}

func (p *gitPackageRevision) GetKptfile(ctx context.Context) (kptfile.KptFile, error) {
	resources, err := p.repo.getResources(p.tree)
	if err != nil {
		return kptfile.KptFile{}, fmt.Errorf("error loading package resources: %w", err)
	}
	kfString, found := resources[kptfile.KptFileName]
	if !found {
		return kptfile.KptFile{}, fmt.Errorf("packagerevision does not have a Kptfile")
	}
	kf, err := pkg.DecodeKptfile(strings.NewReader(kfString))
	if err != nil {
		return kptfile.KptFile{}, fmt.Errorf("error decoding Kptfile: %w", err)
	}
	return *kf, nil
}

// GetUpstreamLock returns the upstreamLock info present in the Kptfile of the package.
func (p *gitPackageRevision) GetUpstreamLock(ctx context.Context) (kptfile.Upstream, kptfile.UpstreamLock, error) {
	kf, err := p.GetKptfile(ctx)
	if err != nil {
		return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("cannot determine package lock; cannot retrieve resources: %w", err)
	}

	if kf.Upstream == nil || kf.UpstreamLock == nil || kf.Upstream.Git == nil {
		// the package doesn't have any upstream package.
		return kptfile.Upstream{}, kptfile.UpstreamLock{}, nil
	}
	return *kf.Upstream, *kf.UpstreamLock, nil
}

// GetLock returns the self version of the package. Think of it as the Git commit information
// that represent the package revision of this package. Please note that it uses Upstream types
// to represent this information but it has no connection with the associated upstream package (if any).
func (p *gitPackageRevision) GetLock() (kptfile.Upstream, kptfile.UpstreamLock, error) {
	repo, err := p.repo.getRepo()
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
		return p.checkPublishedLifecycle()
	case isDraftBranchNameInLocal(ref.Name()):
		return v1alpha1.PackageRevisionLifecycleDraft
	case isProposedBranchNameInLocal(ref.Name()):
		return v1alpha1.PackageRevisionLifecycleProposed
	default:
		return p.checkPublishedLifecycle()
	}

}

func (p *gitPackageRevision) checkPublishedLifecycle() v1alpha1.PackageRevisionLifecycle {
	if p.repo.deletionProposedCache == nil {
		if err := p.repo.UpdateDeletionProposedCache(); err != nil {
			klog.Errorf("failed to update deletionProposed cache: %v", err)
			return v1alpha1.PackageRevisionLifecyclePublished
		}
	}

	branchName := createDeletionProposedName(p.path, p.revision)
	if _, found := p.repo.deletionProposedCache[branchName]; found {
		return v1alpha1.PackageRevisionLifecycleDeletionProposed
	}

	return v1alpha1.PackageRevisionLifecyclePublished
}

func (p *gitPackageRevision) UpdateLifecycle(ctx context.Context, new v1alpha1.PackageRevisionLifecycle) error {
	old := p.Lifecycle()
	if !v1alpha1.LifecycleIsPublished(old) {
		return fmt.Errorf("cannot update lifecycle for draft package revision")
	}
	refSpecs := newPushRefSpecBuilder()
	deletionProposedBranch := createDeletionProposedName(p.path, p.revision)

	if old == v1alpha1.PackageRevisionLifecyclePublished {
		if new != v1alpha1.PackageRevisionLifecycleDeletionProposed {
			return fmt.Errorf("invalid new lifecycle value: %q", new)
		}

		// Push the package revision into a deletionProposed branch.
		p.repo.deletionProposedCache[deletionProposedBranch] = true
		refSpecs.AddRefToPush(p.commit, deletionProposedBranch.RefInLocal())
	}
	if old == v1alpha1.PackageRevisionLifecycleDeletionProposed {
		if new != v1alpha1.PackageRevisionLifecyclePublished {
			return fmt.Errorf("invalid new lifecycle value: %q", new)
		}

		// Delete the deletionProposed branch
		delete(p.repo.deletionProposedCache, deletionProposedBranch)
		ref := plumbing.NewHashReference(deletionProposedBranch.RefInLocal(), p.commit)
		refSpecs.AddRefToDelete(ref)
	}

	if err := p.repo.pushAndCleanup(ctx, refSpecs); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return err
		}
	}

	return nil
}

// TODO: Define a type `gitPackage` to implement the Repository.Package interface
