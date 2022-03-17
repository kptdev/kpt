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
	"fmt"
	"io"
	"strings"
	"time"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
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
	tree     plumbing.Hash       // tree of the package itself, some descendent of commit.Tree()
	commit   plumbing.Hash       // Current version of the package (commit sha)
}

var _ repository.PackageRevision = &gitPackageRevision{}

func (p *gitPackageRevision) Name() string {
	return p.parent.name + ":" + p.path + ":" + p.revision
}

func (p *gitPackageRevision) uid() types.UID {
	return types.UID(fmt.Sprintf("uid:%s:%s", p.path, p.revision))
}

func (p *gitPackageRevision) GetPackageRevision() (*v1alpha1.PackageRevision, error) {
	return &v1alpha1.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.Name(),
			Namespace:       p.parent.namespace,
			UID:             p.uid(),
			ResourceVersion: p.commit.String(),
			CreationTimestamp: metav1.Time{
				Time: p.updated,
			},
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    p.path,
			Revision:       p.revision,
			RepositoryName: p.parent.name,
			Type:           p.getPackageRevisionType(),
			Tasks:          []v1alpha1.Task{},
		},
		Status: v1alpha1.PackageRevisionStatus{},
	}, nil
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
	return &v1alpha1.PackageRevisionResources{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevisionResources",
			APIVersion: v1alpha1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.Name(),
			Namespace:       p.parent.namespace,
			UID:             p.uid(),
			ResourceVersion: p.commit.String(),
			CreationTimestamp: metav1.Time{
				Time: p.updated,
			},
			OwnerReferences: []metav1.OwnerReference{}, // TODO: should point to repository resource
		},
		Spec: v1alpha1.PackageRevisionResourcesSpec{
			Resources: resources,
		},
	}, nil
}

func (p *gitPackageRevision) GetUpstreamLock() (kptfile.Upstream, kptfile.UpstreamLock, error) {
	repo, err := p.parent.getRepo()
	if err != nil {
		return kptfile.Upstream{}, kptfile.UpstreamLock{}, fmt.Errorf("cannot determine package lock: %w", err)
	}

	return kptfile.Upstream{
			Type: kptfile.GitOrigin,
			Git: &kptfile.Git{
				Repo:      repo,
				Directory: p.path,
				Ref:       p.revision,
			},
		}, kptfile.UpstreamLock{
			Type: kptfile.GitOrigin,
			Git: &kptfile.GitLock{
				Repo:      repo,
				Directory: p.path,
				Ref:       p.revision,
				Commit:    p.commit.String(),
			},
		}, nil
}

func (p *gitPackageRevision) getPackageRevisionType() v1alpha1.PackageRevisionType {
	switch {
	case p.ref == nil || p.ref.Name().IsTag():
		return v1alpha1.PackageRevisionTypeFinal
	case strings.HasPrefix(p.ref.Name().String(), refDraftPrefix):
		return v1alpha1.PackageRevisionTypeDraft
	case strings.HasPrefix(p.ref.Name().String(), refProposedPrefix):
		return v1alpha1.PackageRevisionTypeProposed
	default:
		return v1alpha1.PackageRevisionTypeFinal
	}
}

func (p *gitPackageRevision) isDraft() bool {
	return p.ref != nil && strings.HasPrefix(p.ref.Name().String(), refDraftPrefix)
}
