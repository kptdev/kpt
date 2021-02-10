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

// Pipeline declares a pipeline of functions used to generate, transform,
// or validate resources. A kpt package contains zero or one pipeline declration.
// The pipeline is defined in a separate file from the Kptfile.
// If a pipeline is not defined in the package, an implicit pipeline is assumed
// which uses the package itself and all subpackages as sources and has no functions.
// Whenever a pipeline includes another package as a source, the input from that
// source will be the hydrated output of the referenced package.
//
// TODO: Remove this definition after the Pipeline in kptfile package merged
type Pipeline struct {
	yaml.ResourceMeta `yaml:",inline"`

	// Following fields define the sequence of functions in the pipeline.
	// Input of the first function is the resolved sources.
	// Input of the second function is the output of the first function, and so on.
	// Order of operation: mutators, validators

	// Mutators defines a list of of KRM functions that mutate resources.
	Mutators []Function `yaml:"mutators,omitempty"`

	// Validators defines a list of KRM functions that validate resources.
	// Validators are not permitted to mutate resources.
	Validators []Function `yaml:"validators,omitempty"`
}

// String returns the string representation of Pipeline struct
// The string returned is the struct content in Go default format
func (p *Pipeline) String() string {
	return fmt.Sprintf("%+v", *p)
}

// validatePipeline will validate all fields in the Pipeline
// 'generators', 'transformers' and 'validators' share same schema and
// they are valid if all functions in them are ALL valid.
func validatePipeline(p *Pipeline) error {
	if p.APIVersion != kptAPIVersion {
		return fmt.Errorf("'apiVersion' %q is invalid, should be %q",
			p.APIVersion, kptAPIVersion)
	}
	if p.Kind != pipelineKind {
		return fmt.Errorf("'kind' %q is invalid, should be %q",
			p.Kind, pipelineKind)
	}
	fns := []Function{}
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
func fnChain(p *Pipeline) ([]kio.Filter, error) {
	fns := []Function{}
	fns = append(fns, p.Mutators...)
	// TODO: Validators cannot modify resources.
	fns = append(fns, p.Validators...)
	var runners []kio.Filter
	for i := range fns {
		fn := fns[i]
		r, err := newFnRunner(&fn)
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
	return p, validatePipeline(p)
}

// FromFile returns a Pipeline read from file
func FromFile(path string) (*Pipeline, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open path %s: %w", path, err)
	}
	return FromReader(f)
}
