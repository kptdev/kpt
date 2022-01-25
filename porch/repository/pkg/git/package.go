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
	"path"
	"time"

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
	draft    *plumbing.Reference
	tree     plumbing.Hash
	sha      plumbing.Hash // Current version of the package (commit sha)
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
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.Name(),
			Namespace:       p.parent.namespace,
			UID:             p.uid(),
			ResourceVersion: p.sha.String(),
			CreationTimestamp: metav1.Time{
				Time: p.updated,
			},
		},
		Spec: v1alpha1.PackageRevisionSpec{
			PackageName:    p.path,
			Revision:       p.revision,
			RepositoryName: p.parent.name,
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
			resources[path.Join(p.path, file.Name)] = content
		}
	}
	return &v1alpha1.PackageRevisionResources{
		ObjectMeta: metav1.ObjectMeta{
			Name:            p.Name(),
			Namespace:       p.parent.namespace,
			UID:             p.uid(),
			ResourceVersion: p.sha.String(),
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
