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

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/kpt/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"k8s.io/klog/v2"
)

type renderPackageMutation struct {
	kpt kpt.Kpt
}

var _ mutation = &renderPackageMutation{}

func (m *renderPackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	extra := map[string]string{}
	r := &packageReader{
		input: resources,
		extra: extra,
	}
	w := &packageWriter{
		output: repository.PackageResources{
			Contents: map[string]string{},
		},
	}

	if err := m.kpt.Render(r, w); err != nil {
		return repository.PackageResources{}, nil, err
	}

	for k, v := range extra {
		if _, ok := w.output.Contents[k]; ok {
			klog.Warningf("package rendering overwrote non-krm content: %q", k)
		}
		w.output.Contents[k] = v
	}

	// TODO: There are internal tasks not represented in the API; Update the Apply interface to enable them.
	return w.output, &api.Task{
		Type: "eval",
		Eval: &api.FunctionEvalTaskSpec{
			Image:     "render",
			ConfigMap: map[string]string{},
		},
	}, nil
}
