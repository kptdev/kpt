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
	"io"
	"io/ioutil"
	"os"
	"path"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	kptAPIVersion string = "kpt.dev/v1alpha1"
	pipelineKind  string = "Pipeline"
	defaultName   string = "pipeline"

	sourceAllSubPkgs string = "./*"
	sourceCurrentPkg string = "."
)

var defaultSources []string = []string{sourceAllSubPkgs}

// Pipeline declares a pipeline of functions used to generate, transform,
// or validate resources. A kpt package contains zero or one pipeline declration.
// The pipeline is defined in a separate file from the Kptfile.
// If a pipeline is not defined in the package, an implicit pipeline is assumed
// which uses the package itself and all subpackages as sources and has no functions.
// Whenever a pipeline includes another package as a source, the input from that
// source will be the hydrated output of the referenced package.
type Pipeline struct {
	yaml.ResourceMeta `yaml:",inline"`
	//  1. Sources to resolve as input to the pipeline. Possible values:
	//  a) A slash-separated, OS-agnostic relative package path which may include '.' and '..' e.g. './base', '../foo'
	//     The source package is resolved recursively.
	//  b) Resources in this package using '.'. Meta resources such as the Kptfile, Pipeline, and function configs
	//     are excluded.
	//  c) Resources in this package AND all resolved subpackages using './*'
	//
	// Resultant list of resources are ordered:
	// - According to the order of sources specified in this array.
	// - When using './*': Subpackages are resolved in alphanumerical order before package resources.
	//
	// When omitted, defaults to './*'.
	Sources []string `yaml:"sources,omitempty"`

	// 2. Sequence of functions to run. Input of the first function is the resolved sources.
	// Input of the second function is the output of the first function, and so on.
	// Order of operations: generators, transformers, validators
	//
	// When omitted, defaults to NO-OP.
	//
	// 2.a  Sequence of KRM functions that generate resources.
	Generators []Function `yaml:"generators,omitempty"`

	// 2.b Sequence of KRM functions that transform resources.
	Transformers []Function `yaml:"transformers,omitempty"`

	// 2.c Sequence of KRM functions that validate resources.
	// Validators are not permitted to mutate resources.
	Validators []Function `yaml:"validators,omitempty"`
}

// String returns the string representation of Pipeline struct
// The string returned is the struct content in Go default format
func (p *Pipeline) String() string {
	return fmt.Sprintf("%+v", *p)
}

// Validate will validate all fields in the Pipeline
// 'generators', 'transformers' and 'validators' share same schema and
// they are valid if all functions in them are ALL valid.
func (p *Pipeline) Validate() error {
	if p.APIVersion != kptAPIVersion {
		return fmt.Errorf("'apiVersion' %q is invalid, should be %q",
			p.APIVersion, kptAPIVersion)
	}
	if p.Kind != pipelineKind {
		return fmt.Errorf("'kind' %q is invalid, should be %q",
			p.Kind, pipelineKind)
	}
	for i, s := range p.Sources {
		if s == sourceAllSubPkgs {
			continue
		}
		if err := validatePath(s); err != nil {
			return fmt.Errorf("'sources[%d]': path %q is invalid: %w", i, s, err)
		}
		p.Sources[i] = path.Clean(s)
	}
	fns := []Function{}
	fns = append(fns, p.Generators...)
	fns = append(fns, p.Transformers...)
	fns = append(fns, p.Validators...)
	for _, f := range fns {
		err := f.Validate()
		if err != nil {
			return fmt.Errorf("function %q is invalid: %w", f.Image, err)
		}
	}
	return nil
}

// fnChain returns a slice of function runners from the
// functions and configs defined in pipeline.
func (p *Pipeline) fnChain() ([]kio.Filter, error) {
	fns := []Function{}
	fns = append(fns, p.Generators...)
	fns = append(fns, p.Transformers...)
	// TODO: Validators cannot modify resources.
	fns = append(fns, p.Validators...)
	var runners []kio.Filter
	for _, fn := range fns {
		r, err := fn.runner()
		if err != nil {
			return nil, err
		}
		runners = append(runners, r)
	}
	return runners, nil
}

// New returns a pointer to a new default Pipeline.
// The default Pipeline should be:
// apiVersion: kpt.dev/v1alpha1
// kind: Pipeline
// metadata:
//   name: pipeline
// sources:
//   - './*'
func New() *Pipeline {
	return &Pipeline{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptAPIVersion,
				Kind:       pipelineKind,
			},
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: defaultName,
				},
			},
		},
		Sources: defaultSources,
	}
}

// fromBytes returns a Pipeline parsed from bytes
func fromBytes(b []byte) (*Pipeline, error) {
	p := New()
	err := yaml.Unmarshal(b, p)
	if err != nil {
		return nil, fmt.Errorf("failed to construct pipeline from bytes: %w", err)
	}
	return p, nil
}

// FromReader returns a Pipeline parsed from the content in reader
func FromReader(r io.Reader) (*Pipeline, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to construct pipeline from reader: %w", err)
	}
	p, err := fromBytes(b)
	if err != nil {
		return nil, err
	}
	return p, p.Validate()
}

// FromFile returns a Pipeline read from file
func FromFile(path string) (*Pipeline, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open path %s: %w", path, err)
	}
	return FromReader(f)
}
