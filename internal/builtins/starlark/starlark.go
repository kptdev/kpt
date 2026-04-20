// Copyright 2026 The kpt Authors
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

package starlark

import (
	"io"

	"github.com/kptdev/kpt/internal/builtins/registry"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

const ImageName = "ghcr.io/kptdev/krm-functions-catalog/starlark"

//nolint:gochecknoinits
func init() { Register() }

func Register() { registry.Register(&Runner{}) }

type Runner struct{}

func (s *Runner) ImageName() string { return ImageName }

func (s *Runner) Run(r io.Reader, w io.Writer, _ io.Writer) error {
	return framework.Execute(
		framework.ResourceListProcessorFunc(Process),
		&kio.ByteReadWriter{
			Reader:                r,
			Writer:                w,
			KeepReaderAnnotations: true,
		},
	)
}
