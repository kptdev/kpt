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
	"fmt"
	"io"

	"github.com/kptdev/kpt/internal/builtins/registry"
	"github.com/kptdev/krm-functions-sdk/go/fn"
)

const ImageName = "ghcr.io/kptdev/krm-functions-catalog/starlark"

func Register() {
	registry.Register(&Runner{})
}

type Runner struct{}

func (s *Runner) ImageName() string { return ImageName }

func (s *Runner) Run(r io.Reader, w io.Writer) error {
	input, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	rl, err := fn.ParseResourceList(input)
	if err != nil {
		return fmt.Errorf("parsing ResourceList: %w", err)
	}
	if _, err := Process(rl); err != nil {
		return err
	}
	out, err := rl.ToYAML()
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}
