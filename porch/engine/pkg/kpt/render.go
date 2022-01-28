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

	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func NewPlaceholderRenderer() fn.Renderer {
	return &renderer{}
}

type renderer struct {
}

var _ fn.Renderer = &renderer{}

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
