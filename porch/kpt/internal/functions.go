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

package internal

import (
	"context"
	"fmt"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var functions map[string]framework.ResourceListProcessorFunc = map[string]framework.ResourceListProcessorFunc{
	"gcr.io/kpt-fn/set-labels:v0.1.5": setLabels,
}

func Eval(ctx context.Context, pkg filesys.FileSystem, fn v1.Function, opts fn.EvalOptions) error {
	rw := &kio.LocalPackageReadWriter{
		IncludeSubpackages: true,
		PackagePath:        "/",
		FileSystem: filesys.FileSystemOrOnDisk{
			FileSystem: pkg,
		},
	}

	rl := framework.ResourceList{}

	if fn.ConfigMap != nil {
		if cm, err := NewConfigMap(fn.ConfigMap); err != nil {
			return err
		} else {
			rl.FunctionConfig = cm
		}
	}

	// Read input
	if items, err := rw.Read(); err != nil {
		return fmt.Errorf("failed to read fn eval input: %w", err)
	} else {
		rl.Items = items
	}

	if err := eval(fn.Image, &rl); err != nil {
		return fmt.Errorf("function evaluation failed; %w", err)
	}

	// Return error on error
	if rl.Results.ExitCode() != 0 {
		return rl.Results
	}

	// Write Output
	if err := rw.Write(rl.Items); err != nil {
		return fmt.Errorf("failed to write fn eval output: %w", err)
	}

	return nil
}

func eval(image string, rl *framework.ResourceList) error {
	// Evaluate
	if f, ok := functions[image]; ok {
		return f(rl)
	} else {
		return fmt.Errorf("unsupported kpt function %q", image)
	}
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
