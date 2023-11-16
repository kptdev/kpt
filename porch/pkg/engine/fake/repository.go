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

package fake

import (
	"context"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
)

// Implementation of the repository.Repository interface for testing.
// TODO(mortent): Implement stub functionality for all functions from the interface.
type Repository struct {
	PackageRevisions []repository.PackageRevision
	Packages         []repository.Package
}

var _ repository.Repository = &Repository{}

func (r *Repository) Close() error {
	return nil
}

func (r *Repository) Version(ctx context.Context) (string, error) {
	return "foo", nil
}

func (r *Repository) ListPackageRevisions(_ context.Context, filter repository.ListPackageRevisionFilter) ([]repository.PackageRevision, error) {
	var revs []repository.PackageRevision
	for _, rev := range r.PackageRevisions {
		if filter.KubeObjectName != "" && filter.KubeObjectName == rev.KubeObjectName() {
			revs = append(revs, rev)
		}
		if filter.Package != "" && filter.Package == rev.Key().Package {
			revs = append(revs, rev)
		}
		if filter.Revision != "" && filter.Revision == rev.Key().Revision {
			revs = append(revs, rev)
		}
		if filter.WorkspaceName != "" && filter.WorkspaceName == rev.Key().WorkspaceName {
			revs = append(revs, rev)
		}
	}
	return revs, nil
}

func (r *Repository) CreatePackageRevision(_ context.Context, pr *v1alpha1.PackageRevision) (repository.PackageDraft, error) {
	return nil, nil
}

func (r *Repository) DeletePackageRevision(context.Context, repository.PackageRevision) error {
	return nil
}

func (r *Repository) UpdatePackageRevision(context.Context, repository.PackageRevision) (repository.PackageDraft, error) {
	return nil, nil
}

func (r *Repository) ListPackages(context.Context, repository.ListPackageFilter) ([]repository.Package, error) {
	return r.Packages, nil
}

func (r *Repository) CreatePackage(_ context.Context, pr *v1alpha1.Package) (repository.Package, error) {
	return nil, nil
}

func (r *Repository) DeletePackage(_ context.Context, pr repository.Package) error {
	return nil
}
