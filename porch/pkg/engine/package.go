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

package engine

import (
	"context"
	"fmt"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
)

type PackageFetcher struct {
	repoOpener        RepositoryOpener
	referenceResolver ReferenceResolver
}

func (p *PackageFetcher) FetchRevision(ctx context.Context, packageRef *api.PackageRevisionRef, namespace string) (repository.PackageRevision, error) {
	repositoryName, err := parseUpstreamRepository(packageRef.Name)
	if err != nil {
		return nil, err
	}
	var resolved configapi.Repository
	if err := p.referenceResolver.ResolveReference(ctx, namespace, repositoryName, &resolved); err != nil {
		return nil, fmt.Errorf("cannot find repository %s/%s: %w", namespace, repositoryName, err)
	}

	repo, err := p.repoOpener.OpenRepository(ctx, &resolved)
	if err != nil {
		return nil, err
	}

	revisions, err := repo.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{KubeObjectName: packageRef.Name})
	if err != nil {
		return nil, err
	}

	var revision repository.PackageRevision
	for _, rev := range revisions {
		if rev.KubeObjectName() == packageRef.Name {
			revision = rev
			break
		}
	}
	if revision == nil {
		return nil, fmt.Errorf("cannot find package revision %q", packageRef.Name)
	}

	return revision, nil
}

func (p *PackageFetcher) FetchResources(ctx context.Context, packageRef *api.PackageRevisionRef, namespace string) (*api.PackageRevisionResources, error) {
	revision, err := p.FetchRevision(ctx, packageRef, namespace)
	if err != nil {
		return nil, err
	}

	resources, err := revision.GetResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot read contents of package %q: %w", packageRef.Name, err)
	}
	return resources, nil
}
