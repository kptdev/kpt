// Copyright 2026 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/kptdev/krm-functions-sdk/go/fn"
	"sigs.k8s.io/kustomize/api/filters/replacement"
	"sigs.k8s.io/kustomize/api/types"
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

func (a *Runner) Run(r io.Reader, w io.Writer) error {
	input, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	rl, err := fn.ParseResourceList(input)
	if err != nil {
		return fmt.Errorf("parsing ResourceList: %w", err)
	}
	if _, err := applyReplacements(rl); err != nil {
		return err
	}
	out, err := rl.ToYAML()
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}
func applyReplacements(rl *fn.ResourceList) (bool, error) {
	r := &Replacements{}
	return r.Process(rl)
}

type Replacements struct {
	Replacements []types.Replacement `json:"replacements,omitempty" yaml:"replacements,omitempty"`
}

func (r *Replacements) Config(functionConfig *fn.KubeObject) error {
	if functionConfig.IsEmpty() {
		return fmt.Errorf("FunctionConfig is missing. Expect `ApplyReplacements`")
	}
	if functionConfig.GetKind() != fnConfigKind || functionConfig.GetAPIVersion() != fnConfigAPIVersion {
		return fmt.Errorf("received functionConfig of kind %s and apiVersion %s, only functionConfig of kind %s and apiVersion %s is supported",
			functionConfig.GetKind(), functionConfig.GetAPIVersion(), fnConfigKind, fnConfigAPIVersion)
	}
	r.Replacements = []types.Replacement{}
	if err := functionConfig.As(r); err != nil {
		return fmt.Errorf("unable to convert functionConfig to replacements:\n%w", err)
	}
	return nil
}

func (r *Replacements) Process(rl *fn.ResourceList) (bool, error) {
	if err := r.Config(rl.FunctionConfig); err != nil {
		rl.LogResult(err)
		return false, err
	}
	transformedItems, err := r.Transform(rl.Items)
	if err != nil {
		rl.LogResult(err)
		return false, err
	}
	rl.Items = transformedItems
	return true, nil
}

func (r *Replacements) Transform(items []*fn.KubeObject) ([]*fn.KubeObject, error) {
	var transformedItems []*fn.KubeObject
	var nodes []*yaml.RNode
	for _, obj := range items {
		objRN, err := yaml.Parse(obj.String())
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, objRN)
	}
	transformedNodes, err := replacement.Filter{
		Replacements: r.Replacements,
	}.Filter(nodes)
	if err != nil {
		return nil, err
	}
	for _, n := range transformedNodes {
		obj, err := fn.ParseKubeObject([]byte(n.MustString()))
		if err != nil {
			return nil, err
		}
		transformedItems = append(transformedItems, obj)
	}
	return transformedItems, nil
}
