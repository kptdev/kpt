// Copyright 2020 Google LLC
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

// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package pipeline

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	sourceAllSubPkgs string = "./*"
	sourceCurrentPkg string = "."
)

// ValidatePipeline will validate all fields in the Pipeline
// 'generators', 'transformers' and 'validators' share same schema and
// they are valid if all functions in them are ALL valid.
func ValidatePipeline(p *v1alpha2.Pipeline) error {
	fns := []v1alpha2.Function{}
	fns = append(fns, p.Mutators...)
	fns = append(fns, p.Validators...)
	for i := range fns {
		f := fns[i]
		err := validateFunction(&f)
		if err != nil {
			return fmt.Errorf("function %q is invalid: %w", f.Image, err)
		}
	}
	return nil
}

// fnChain returns a slice of function runners from the
// functions and configs defined in pipeline.
func fnChain(p *v1alpha2.Pipeline, pkgPath string) ([]kio.Filter, error) {
	fns := []v1alpha2.Function{}
	fns = append(fns, p.Mutators...)
	// TODO: Validators cannot modify resources.
	fns = append(fns, p.Validators...)
	var runners []kio.Filter
	for i := range fns {
		fn := fns[i]
		r, err := newFnRunner(&fn, pkgPath)
		if err != nil {
			return nil, err
		}
		runners = append(runners, r)
	}
	return runners, nil
}

// fromBytes returns a Pipeline parsed from bytes
func fromBytes(b []byte) (*v1alpha2.Pipeline, error) {
	p := &v1alpha2.Pipeline{}
	err := yaml.Unmarshal(b, p)
	if err != nil {
		return nil, fmt.Errorf("failed to construct pipeline from bytes: %w", err)
	}
	return p, nil
}

func fromString(s string) (*v1alpha2.Pipeline, error) {
	return fromBytes([]byte(s))
}
