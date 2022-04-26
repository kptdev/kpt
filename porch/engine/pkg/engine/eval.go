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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type evalFunctionMutation struct {
	runtime fn.FunctionRuntime
	task    *api.Task
}

func (m *evalFunctionMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	ctx, span := tracer.Start(ctx, "evalFunctionMutation::Apply", trace.WithAttributes())
	defer span.End()

	e := m.task.Eval

	// TODO: Apply should accept filesystem instead of PackageResources

	x := FunctionExecution{
		Runtime:       m.runtime,
		FunctionImage: e.Image,
		Input:         resources,
	}
	if m.task.Eval.ConfigMap != nil {
		if cm, err := kpt.NewConfigMap(m.task.Eval.ConfigMap); err != nil {
			return repository.PackageResources{}, nil, fmt.Errorf("failed to create function config: %w", err)
		} else {
			x.FunctionConfig = cm
		}
	} else if len(m.task.Eval.Config.Raw) != 0 {
		// raw is JSON (we expect), but we take advantage of the fact that YAML is a superset of JSON
		config, err := yaml.Parse(string(m.task.Eval.Config.Raw))
		if err != nil {
			return repository.PackageResources{}, nil, fmt.Errorf("error parsing function config: %w", err)
		}
		x.FunctionConfig = config
	}

	output, err := x.Run(ctx)
	if err != nil {
		return repository.PackageResources{}, nil, err
	}
	return *output, m.task, nil
}

type FunctionExecution struct {
	Runtime        fn.FunctionRuntime
	FunctionImage  string
	Input          repository.PackageResources
	FunctionConfig *yaml.RNode
}

func (f *FunctionExecution) Run(ctx context.Context) (*repository.PackageResources, error) {
	ctx, span := tracer.Start(ctx, "runFunction", trace.WithAttributes(attribute.String("functionImage", f.FunctionImage)))
	defer span.End()

	klog.Infof("running function %s", f.FunctionImage)
	runner, err := f.Runtime.GetRunner(ctx, &v1.Function{
		Image: f.FunctionImage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create function runner: %w", err)
	}

	ff := &runtimeutil.FunctionFilter{
		Run:            runner.Run,
		FunctionConfig: f.FunctionConfig,
		Results:        &yaml.RNode{},
	}

	pr := &packageReader{
		input: f.Input,
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
		return nil, fmt.Errorf("failed to evaluate function: %w", err)
	}

	for k, v := range result.Contents {
		result.Contents[k] = v
	}

	// Return extras. TODO: Apply should accept FS.
	for k, v := range pr.extra {
		result.Contents[k] = v
	}

	if _, found := result.Contents["Kptfile"]; !found {
		// TODO: Something more useful
		result.Contents["Kptfile"] = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: generated
info:
  description: generated
`
	}

	return &result, nil
}
