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

package engine

import (
	"context"
	"fmt"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"go.opentelemetry.io/otel/trace"
)

type editPackageMutation struct {
	task              *api.Task
	name              string
	namespace         string
	cad               CaDEngine
	referenceResolver ReferenceResolver
}

var _ mutation = &editPackageMutation{}

func (m *editPackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	ctx, span := tracer.Start(ctx, "editPackageMutation::Apply", trace.WithAttributes())
	defer span.End()

	sourceRef := m.task.Edit.Source

	sourceResources, err := (&PackageFetcher{
		cad:               m.cad,
		referenceResolver: m.referenceResolver,
	}).FetchResources(ctx, sourceRef, m.namespace)
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to fetch resources for package %q: %w", sourceRef.Name, err)
	}

	// Update Kptfile
	if err := kpt.UpdateKptfileName(m.name, sourceResources.Spec.Resources); err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to update package name %q: %w", sourceRef.Name, err)
	}

	return repository.PackageResources{
		Contents: sourceResources.Spec.Resources,
	}, &api.Task{}, nil
}
