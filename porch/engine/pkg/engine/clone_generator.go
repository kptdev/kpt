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

	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func (m *clonePackageMutation) cloneFromGenerator(ctx context.Context, spec *api.GeneratorSpec, runtime fn.FunctionRuntime) (repository.PackageResources, error) {
	if len(spec.Config.Raw) == 0 {
		return repository.PackageResources{}, fmt.Errorf("config must be provided")
	}

	var config unstructured.Unstructured
	if err := config.UnmarshalJSON(spec.Config.Raw); err != nil {
		return repository.PackageResources{}, fmt.Errorf("error parsing generator config: %w", err)
	}

	functionImage, err := m.mapToFunctionImage(ctx, &config)
	if err != nil {
		return repository.PackageResources{}, err
	}

	x := FunctionExecution{
		Runtime:       runtime,
		FunctionImage: functionImage,
		Input:         repository.PackageResources{},
	}
	{
		// raw is JSON (we expect), but we take advantage of the fact that YAML is a superset of JSON
		configRNode, err := yaml.Parse(string(spec.Config.Raw))
		if err != nil {
			return repository.PackageResources{}, fmt.Errorf("error parsing function config: %w", err)
		}
		x.FunctionConfig = configRNode
	}

	output, err := x.Run(ctx)
	if err != nil {
		return repository.PackageResources{}, err
	}

	return *output, nil
}

func (m *clonePackageMutation) mapToFunctionImage(ctx context.Context, config *unstructured.Unstructured) (string, error) {
	gvk := config.GetObjectKind().GroupVersionKind()

	// TODO:: Look at config.kubernetes.io/function annotation?
       // e.g. https://github.com/GoogleContainerTools/kpt-functions-catalog/blob/b766c8d1e2d2027313fd50d24f6da28e47c2d247/examples/render-helm-chart-kustomize-inline-values/kustomization.yaml#L7-L11

	registry := m.cad.Registry()
	if registry == nil {
		// TODO: Remove this mapping; registry should probably be required / have a public default
		klog.Warningf("registry not configured, falling back to hard-coded mapping")
		switch gvk.GroupKind() {
		case schema.GroupKind{Group: "fn.kpt.dev", Kind: "RenderHelmChart"}:
			return "gcr.io/kpt-fn/render-helm-chart:unstable", nil
		}

		return "", fmt.Errorf("registry not configured; cannot get function for %v", gvk)
	}
	functionImage, err := registry.FunctionImageForGVK(ctx, gvk)
	if err != nil {
		return "", err
	}
	return functionImage, nil
}
