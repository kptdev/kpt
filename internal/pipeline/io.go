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
//
// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package pipeline

import (
	"io"
	"io/ioutil"
	"os"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	kptAPIVersion string = "kpt.dev/v1alpha1"
	pipelineKind  string = "Pipeline"
	defaultName   string = "pipeline"
)

var defaultSources []string = []string{"./*"}

// newEmptyPipeline returns a new empty Pipeline
func newEmptyPipeline() *Pipeline {
	return &Pipeline{}
}

// NewPipeline returns a pointer to a new default Pipeline.
// The default Pipeline should be:
// apiVersion: kpt.dev/v1alpha1
// kind: Pipeline
// metadata:
//   name: pipeline
// sources:
//   - '.*'
func NewPipeline() *Pipeline {
	p := newEmptyPipeline()
	p.APIVersion = kptAPIVersion
	p.Kind = pipelineKind
	p.Name = defaultName
	p.Sources = defaultSources
	return p
}

// FromBytes returns a Pipeline parsed from bytes
func FromBytes(b []byte) (*Pipeline, error) {
	p := newEmptyPipeline()
	err := yaml.Unmarshal(b, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// FromString returns a Pipeline parsed from string
func FromString(s string) (*Pipeline, error) {
	return FromBytes([]byte(s))
}

// FromReader returns a Pipeline parsed from the content in reader
func FromReader(r io.Reader) (*Pipeline, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p, err := FromBytes(b)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// FromFile returns a Pipeline read from file
func FromFile(path string) (*Pipeline, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return FromReader(f)
}
