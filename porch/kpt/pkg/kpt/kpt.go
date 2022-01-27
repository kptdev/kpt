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

package kpt

import (
	"context"
	"fmt"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/kpt/pkg/kpt/internal"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewPlaceholderRenderer() fn.Renderer {
	return &renderer{}
}

func NewPlaceholderEvaluator() fn.Evaluator {
	return &evaluator{}
}

type evaluator struct {
}

var _ fn.Evaluator = &evaluator{}

type renderer struct {
}

var _ fn.Renderer = &renderer{}

func (k *evaluator) Eval(ctx context.Context, pkg filesys.FileSystem, fn v1.Function, opts fn.EvalOptions) error {
	rw := &kio.LocalPackageReadWriter{
		IncludeSubpackages: true,
		PackagePath:        "/",
		FileSystem: filesys.FileSystemOrOnDisk{
			FileSystem: pkg,
		},
	}

	var config kio.ResourceNodeSlice
	if fn.ConfigMap != nil {
		if cm, err := NewConfigMap(fn.ConfigMap); err != nil {
			return err
		} else {
			config = kio.ResourceNodeSlice{cm}
		}
	}

	return k.OldEval(rw, fn.Image, config, rw)
}

func (k *evaluator) OldEval(input kio.Reader, function string, config kio.Reader, output kio.Writer) error {
	var err error
	rl := framework.ResourceList{}

	// Read input
	if rl.Items, err = input.Read(); err != nil {
		return fmt.Errorf("failed to read fn eval input: %w", err)
	}

	// Function config
	if fc, err := config.Read(); err != nil {
		return fmt.Errorf("failed to read fn eval config: %w", err)
	} else {
		switch count := len(fc); count {
		case 0:
			// ok; no config
		case 1:
			rl.FunctionConfig = fc[0]
		default:
			return fmt.Errorf("invalid function config containing %d resources; expected at most one", count)
		}
	}

	// Evaluate
	if err := internal.Eval(function, &rl); err != nil {
		klog.Errorf("kpt fn eval failed: %v", err)
		return fmt.Errorf("kpt fn eval failed: %w", err)
	}

	// Return error on error
	if rl.Results.ExitCode() != 0 {
		return rl.Results
	}

	// Write Output
	if err := output.Write(rl.Items); err != nil {
		return fmt.Errorf("failed to write fn eval output: %w", err)
	}

	return nil
}

func (r *renderer) Render(ctx context.Context, pkg filesys.FileSystem, opts fn.RenderOptions) error {
	rw := &kio.LocalPackageReadWriter{
		PackagePath:        "/",
		IncludeSubpackages: true,
		FileSystem: filesys.FileSystemOrOnDisk{
			FileSystem: pkg,
		},
	}

	// Currently a noop rendering. TODO: Implement
	nodes, err := rw.Read()
	if err != nil {
		return err
	}

	for _, n := range nodes {
		ann := n.GetAnnotations()
		ann["porch.kpt.dev/rendered"] = "yes"
		n.SetAnnotations(ann)
	}

	return rw.Write(nodes)
}

func NewConfigMap(data map[string]string) (*yaml.RNode, error) {
	node := yaml.NewMapRNode(&data)
	if node == nil {
		return nil, nil
	}
	// create a ConfigMap only for configMap config
	configMap := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {}
`)
	if err := configMap.PipeE(yaml.SetField("data", node)); err != nil {
		return nil, err
	}
	return configMap, nil
}
