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
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel/trace"
)

type editPackageMutation struct {
	task              *api.Task
	namespace         string
	repositoryName    string
	packageName       string
	repoOpener        RepositoryOpener
	referenceResolver ReferenceResolver
}

var _ mutation = &editPackageMutation{}

func (m *editPackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.TaskResult, error) {
	ctx, span := tracer.Start(ctx, "editPackageMutation::Apply", trace.WithAttributes())
	defer span.End()

	sourceRef := m.task.Edit.Source

	revision, err := (&PackageFetcher{
		repoOpener:        m.repoOpener,
		referenceResolver: m.referenceResolver,
	}).FetchRevision(ctx, sourceRef, m.namespace)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to fetch package %q: %w", sourceRef.Name, err)
	}

	// We only allow edit to create new revision from the same package.
	if revision.Key().Package != m.packageName ||
		revision.Key().Repository != m.repositoryName {
		return repository.PackageResources{}, nil, fmt.Errorf("source revision must be from same package %s/%s", m.repositoryName, m.packageName)
	}

	// We only allow edit to create new revisions from published packages.
	if !api.LifecycleIsPublished(revision.Lifecycle()) {
		return repository.PackageResources{}, nil, fmt.Errorf("source revision must be published")
	}

	sourceResources, err := revision.GetResources(ctx)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("cannot read contents of package %q: %w", sourceRef.Name, err)
	}

	return repository.PackageResources{
		Contents: sourceResources.Spec.Resources,
	}, &api.TaskResult{Task: m.task}, nil
}
