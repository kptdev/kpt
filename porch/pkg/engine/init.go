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

	"github.com/GoogleContainerTools/kpt/pkg/kptpkg"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type initPackageMutation struct {
	kptpkg.DefaultInitializer
	name string
	task *api.Task
}

var _ mutation = &initPackageMutation{}

func (m *initPackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.TaskResult, error) {
	ctx, span := tracer.Start(ctx, "initPackageMutation::Apply", trace.WithAttributes())
	defer span.End()

	fs := filesys.MakeFsInMemory()
	// virtual fs expected a rooted filesystem
	pkgPath := "/"

	if m.task.Init.Subpackage != "" {
		pkgPath = "/" + m.task.Init.Subpackage
	}
	if err := fs.Mkdir(pkgPath); err != nil {
		return repository.PackageResources{}, nil, err
	}
	err := m.Initialize(printer.WithContext(ctx, &fake.Printer{}), fs, kptpkg.InitOptions{
		PkgPath:  pkgPath,
		PkgName:  m.name,
		Desc:     m.task.Init.Description,
		Keywords: m.task.Init.Keywords,
		Site:     m.task.Init.Site,
	})
	if err != nil {
		return repository.PackageResources{}, nil, fmt.Errorf("failed to initialize pkg %q: %w", m.name, err)
	}

	result, err := readResources(fs)
	if err != nil {
		return repository.PackageResources{}, nil, err
	}

	return result, &api.TaskResult{Task: m.task}, nil
}
