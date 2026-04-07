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

package applyreplacements

import (
	"fmt"
	"io"

	"github.com/kptdev/kpt/internal/builtins/registry"
	"sigs.k8s.io/kustomize/api/filters/replacement"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	ImageName          = "ghcr.io/kptdev/krm-functions-catalog/apply-replacements"
	fnConfigKind       = "ApplyReplacements"
	fnConfigAPIVersion = "fn.kpt.dev/v1alpha1"
)

//nolint:gochecknoinits
func init() { Register() }

func Register() {
	registry.Register(&Runner{})
}

type Runner struct{}

func (a *Runner) ImageName() string { return ImageName }

func (a *Runner) Run(r io.Reader, w io.Writer, _ io.Writer) error {
	return framework.Execute(
		framework.ResourceListProcessorFunc(Process),
		&kio.ByteReadWriter{
			Reader:                r,
			Writer:                w,
			KeepReaderAnnotations: true,
		},
	)
}

func Process(rl *framework.ResourceList) error {
	rep := &Replacements{}

	if rl.FunctionConfig == nil {
		return fmt.Errorf("FunctionConfig is missing. Expect `ApplyReplacements`")
	}

	meta, err := rl.FunctionConfig.GetMeta()
	if err != nil {
		return fmt.Errorf("reading functionConfig metadata: %w", err)
	}
	if meta.Kind != fnConfigKind || meta.APIVersion != fnConfigAPIVersion {
		return fmt.Errorf("received functionConfig of kind %s and apiVersion %s, only functionConfig of kind %s and apiVersion %s is supported",
			meta.Kind, meta.APIVersion, fnConfigKind, fnConfigAPIVersion)
	}

	if err := yaml.Unmarshal([]byte(rl.FunctionConfig.MustString()), rep); err != nil {
		return fmt.Errorf("unable to convert functionConfig to replacements: %w", err)
	}

	transformed, err := replacement.Filter{
		Replacements: rep.Replacements,
	}.Filter(rl.Items)
	if err != nil {
		return err
	}
	rl.Items = transformed
	return nil
}

type Replacements struct {
	Replacements []types.Replacement `json:"replacements,omitempty" yaml:"replacements,omitempty"`
}
