// Copyright 2019 Google LLC
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

// Package sub substitutes variables into a package
package sub

import (
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var _ kio.Filter = &Sub{}

// Sub performs substitutions
type Sub struct {
	// Count is the number of substitutions that have been performed
	Count int

	// Substitution defines the substitution to perform
	kptfile.Substitution
}

func (s *Sub) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	for i := range input {
		// perform the substitutions on each Resource object
		for j := range s.Substitution.Paths {
			// perform the substitutions for each field of each object
			fs := &FieldSub{
				Path:         s.Substitution.Paths[j],
				Substitution: s.Substitution,
			}
			if err := input[i].PipeE(fs); err != nil {
				return nil, err
			}
			if fs.Found {
				// increment the count if the value was substituted
				s.Count++
			}
		}
	}

	return input, nil
}

var _ yaml.Filter = &FieldSub{}

// FieldSub substitutes a Marker value on a field
type FieldSub struct {
	// Found will be true if a value was substituted
	Found bool

	// Path is the path to the field to substitute
	Path kptfile.Path

	// Substitution defines the Marker and Value to substitute
	kptfile.Substitution
}

func (fs *FieldSub) Filter(object *yaml.RNode) (*yaml.RNode, error) {
	// get the field to substitute
	field, err := object.Pipe(yaml.Lookup(fs.Path.Path...))
	if err != nil {
		return nil, err
	}
	if field == nil {
		// object doesn't have the field -- no-op
		return object, nil
	}

	if !strings.Contains(field.YNode().Value, fs.Marker) {
		// field doesn't have the marker -- no-op
		return object, nil
	}

	// replace the marker with the new value
	fs.Found = true
	field.YNode().Value = strings.ReplaceAll(field.YNode().Value, fs.Marker, fs.StringValue)
	// be sure to set the tag so the yaml doesn't quote ints or bools
	field.YNode().Tag = fs.Type.Tag()
	field.YNode().Style = 0

	return object, nil
}
