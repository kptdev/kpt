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

	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/pkg/kptpkg"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func NewInitializer() kptpkg.Initializer {
	return &initializer{}
}

type initializer struct{}

func (i *initializer) Initialize(ctx context.Context, fsys filesys.FileSystem, opts kptpkg.InitOptions) error {
	return (&kptpkg.DefaultInitializer{}).Initialize(printer.WithContext(ctx, &packagePrinter{}), fsys, opts)
}
