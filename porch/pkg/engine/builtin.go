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

	"github.com/GoogleContainerTools/kpt/internal/builtins"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type builtinEvalMutation struct {
	function string
	runner   fn.FunctionRunner
}

func newPackageContextGeneratorMutation(packageConfig *builtins.PackageConfig) (mutation, error) {
	runner := &builtins.PackageContextGenerator{
		PackageConfig: packageConfig,
	}

	return &builtinEvalMutation{
		function: fnruntime.FuncGenPkgContext,
		runner:   runner,
	}, nil
}

var _ mutation = &builtinEvalMutation{}

func (m *builtinEvalMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.TaskResult, error) {
	ff := &runtimeutil.FunctionFilter{
		Run:     m.runner.Run,
		Results: &yaml.RNode{},
	}

	pr := &packageReader{
		input: resources,
		extra: map[string]string{},
	}

	result := repository.PackageResources{
		Contents: map[string]string{},
	}

	pipeline := kio.Pipeline{
		Inputs:  []kio.Reader{pr},
		Filters: []kio.Filter{ff},
		Outputs: []kio.Writer{&packageWriter{
			output: result,
		}},
	}

	if err := pipeline.Execute(); err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to evaluate function %q: %w", m.function, err)
	}

	for k, v := range pr.extra {
		result.Contents[k] = v
	}

	return result, &api.TaskResult{}, nil
}
