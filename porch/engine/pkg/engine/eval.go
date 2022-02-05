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

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type evalFunctionMutation struct {
	runtime fn.FunctionRuntime
	task    *api.Task
}

func (m *evalFunctionMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	e := m.task.Eval

	// TODO: Apply should accept filesystem instead of PackageResources

	runner, err := m.runtime.GetRunner(ctx, &v1.Function{
		Image: e.Image,
	})
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to create function runner: %w", err)
	}

	var functionConfig *yaml.RNode
	if m.task.Eval.ConfigMap != nil {
		if cm, err := kpt.NewConfigMap(m.task.Eval.ConfigMap); err != nil {
			return repository.PackageResources{}, nil, fmt.Errorf("failed to create function config: %w", err)
		} else {
			functionConfig = cm
		}
	}

	ff := &runtimeutil.FunctionFilter{
		Run:            runner.Run,
		FunctionConfig: functionConfig,
		Results:        &yaml.RNode{},
	}

	pr := &packageReader{
		input: resources,
		extra: map[string]string{},
	}

	// r := &kio.LocalPackageReader{
	// 	PackagePath:        "/",
	// 	IncludeSubpackages: true,
	// 	FileSystem:         filesys.FileSystemOrOnDisk{FileSystem: fs},
	// 	WrapBareSeqNode:    true,
	// }

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
		return repository.PackageResources{}, nil, fmt.Errorf("failed to evaluate function: %w", err)
	}

	// Return extras. TODO: Apply should accept FS.
	for k, v := range pr.extra {
		result.Contents[k] = v
	}

	return result, m.task, nil
}
