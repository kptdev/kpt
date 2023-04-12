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

package functiondiscovery

import (
	"context"
	"fmt"

	api "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/oci"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var tracer = otel.Tracer("functioncontroller")

func (r *FunctionReconciler) listFunctions(ctx context.Context, subject *api.Repository) ([]*api.Function, error) {
	log := log.FromContext(ctx)

	ctx, span := tracer.Start(ctx, "FunctionReconciler::listFunctions", trace.WithAttributes())
	defer span.End()

	// Repository whose content type is not Function contains no Function resources.
	if subject.Spec.Content != api.RepositoryContentFunction {
		log.Info("repository doesn't contain functions")
		return nil, nil
	}

	if subject.Spec.Oci == nil {
		return nil, fmt.Errorf("expected spec.oci to be set")
	}

	registry := subject.Spec.Oci.Registry
	if registry == "" {
		return nil, fmt.Errorf("spec.oci.registry is not set")
	}

	repo, err := oci.OpenRepository(subject.Name, subject.Namespace, subject.Spec.Content, subject.Spec.Oci, subject.Spec.Deployment, r.ociStorage)
	if err != nil {
		return nil, err
	}

	functionRepository, ok := (repo).(repository.FunctionRepository)
	if !ok {
		return nil, fmt.Errorf("repository was not of expected type, expected FunctionRepository, got %T", repo)
	}

	functions, err := functionRepository.ListFunctions(ctx)
	if err != nil {
		return nil, err
	}

	var functionObjects []*api.Function
	for _, f := range functions {
		functionObject, err := f.GetCRD()
		if err != nil {
			return nil, err
		}
		functionObject.Namespace = subject.Namespace
		// TODO: Set ownerRef?
		// TODO: Set labels?
		functionObjects = append(functionObjects, functionObject)
	}

	return functionObjects, nil
}
