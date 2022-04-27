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

package fake

import (
	"context"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
)

// Implementation of the repository.Repository interface for testing.
// TODO(mortent): Implement stub functionality for all functions from the interface.
type Repository struct {
	PackageRevisions []repository.PackageRevision
}

func (r *Repository) ListPackageRevisions(context.Context, repository.ListPackageRevisionFilter) ([]repository.PackageRevision, error) {
	return r.PackageRevisions, nil
}

func (r *Repository) CreatePackageRevision(_ context.Context, pr *v1alpha1.PackageRevision) (repository.PackageDraft, error) {
	return nil, nil
}

func (r *Repository) DeletePackageRevision(context.Context, repository.PackageRevision) error {
	return nil
}

func (r *Repository) UpdatePackage(context.Context, repository.PackageRevision) (repository.PackageDraft, error) {
	return nil, nil
}
