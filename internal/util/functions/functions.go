// Copyright 2019 Google LLC
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

package functions

import (
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/container"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/exec"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/starlark"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/runfn"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func RunFunctions(path string, functions []kptfile.Function) error {
	rw := &kio.LocalPackageReadWriter{
		PackagePath:        path,
		IncludeSubpackages: true,
	}

	var fltrs []kio.Filter
	for i := range functions {
		f := functions[i]
		var e exec.Filter
		e.FunctionConfig = yaml.NewRNode(&f.Config)
		fltrs = append(fltrs, &container.Filter{
			ContainerSpec: runtimeutil.ContainerSpec{
				Image: f.Image,
			},
			Exec: e,
		})
	}
	if len(fltrs) == 0 {
		return nil
	}

	return kio.Pipeline{Inputs: []kio.Reader{rw}, Filters: fltrs, Outputs: []kio.Writer{rw}}.
		Execute()
}

// ReconcileFunctions runs functions specified by the Kptfile
func ReconcileFunctions(path string) error {
	k, err := kptfileutil.ReadFile(path)
	if err != nil {
		// do nothing if the package doesn't have a Kptfile
		return nil
	}
	if k.Functions.AutoRunStarlark {
		err := runfn.RunFns{
			EnableStarlark: k.Functions.AutoRunStarlark,
			// TODO: make auto-running containers an option
			DisableContainers: true,
			Path:              path,
		}.Execute()
		if err != nil {
			return err
		}
	}

	if len(k.Functions.StarlarkFunctions) > 0 {
		var fltrs []kio.Filter
		for _, fn := range k.Functions.StarlarkFunctions {
			fltrs = append(fltrs, &starlark.Filter{
				Name: fn.Name,
				Path: filepath.Join(path, fn.Path),
			})
		}
		rw := &kio.LocalPackageReadWriter{PackagePath: path}
		err = kio.Pipeline{
			Inputs:  []kio.Reader{rw},
			Filters: fltrs,
			Outputs: []kio.Writer{rw},
		}.Execute()
		if err != nil {
			return err
		}
	}

	return nil
}
