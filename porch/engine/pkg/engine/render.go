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
	iofs "io/fs"
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/fn"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type renderPackageMutation struct {
	renderer fn.Renderer
	runner   fn.FunctionRunner
}

var _ mutation = &renderPackageMutation{}

func (m *renderPackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	fs := &kpt.MemFS{}

	if err := writeResources(fs, resources); err != nil {
		return repository.PackageResources{}, nil, err
	}

	if err := m.renderer.Render(ctx, fs, fn.RenderOptions{
		Runner: m.runner,
	}); err != nil {
		return repository.PackageResources{}, nil, err
	}

	result, err := readResources(fs)
	if err != nil {
		return repository.PackageResources{}, nil, err
	}

	// TODO: There are internal tasks not represented in the API; Update the Apply interface to enable them.
	return result, &api.Task{
		Type: "eval",
		Eval: &api.FunctionEvalTaskSpec{
			Image:     "render",
			ConfigMap: map[string]string{},
		},
	}, nil
}

// TODO: Implement filesystem abstraction directly rather than on top of PackageResources
func writeResources(fs filesys.FileSystem, resources repository.PackageResources) error {
	for k, v := range resources.Contents {
		if err := fs.MkdirAll(path.Dir(k)); err != nil {
			return err
		}
		if err := fs.WriteFile(k, []byte(v)); err != nil {
			return err
		}
	}
	return nil
}

func readResources(fs filesys.FileSystem) (repository.PackageResources, error) {
	contents := map[string]string{}

	if err := fs.Walk("", func(path string, info iofs.FileInfo, err error) error {
		if info.Mode().IsRegular() {
			data, err := fs.ReadFile(path)
			if err != nil {
				return err
			}
			contents[strings.TrimPrefix(path, "/")] = string(data)
		}
		return nil
	}); err != nil {
		return repository.PackageResources{}, err
	}

	return repository.PackageResources{
		Contents: contents,
	}, nil
}
