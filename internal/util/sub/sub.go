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
	"github.com/GoogleContainerTools/kpt/internal/util/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var _ kio.Filter = &Sub{}

// Sub performs substitutions
type Sub struct {
	Modified int

	Remaining int

	Done int

	Value string

	Override bool

	Revert bool

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
				Override:     s.Override,
				Revert:       s.Revert,
			}
			if err := input[i].PipeE(fs); err != nil {
				return nil, err
			}
			if fs.Modified {
				// increment the count if the value was substituted
				s.Modified++
			}
			if fs.ContainsMarker {
				s.Remaining++
			}
			if fs.ContainsValue {
				s.Value = fs.Value
				s.Done++
			}
		}
	}

	return input, nil
}

var _ yaml.Filter = &FieldSub{}

// FieldSub substitutes a Marker value on a field
type FieldSub struct {
	// Modified will be true if a value was substituted
	Modified bool

	ContainsMarker bool

	ContainsValue bool

	// Value is the current substituted value
	Value string

	// Override if set to true will replace previously substituted values
	Override bool

	// Revert if set to true will undo previously substituted values
	Revert bool

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

	// check for a substitution for this field
	var fm = &fieldmeta.FieldMeta{}
	if err := fm.Read(field); err != nil {
		return nil, err
	}
	var s *fieldmeta.Substitution
	for i := range fm.Substitutions {
		if fm.Substitutions[i].Name == fs.Name {
			s = &fm.Substitutions[i]
			break
		}
	}
	if s == nil {
		// no substitutions for this field
		return object, nil
	}

	// record stats
	if strings.Contains(field.YNode().Value, s.Marker) {
		fs.ContainsMarker = true
	}
	if s.Value != "" {
		fs.Value = s.Value
		fs.ContainsValue = true
	}

	// undo or override previous substitutions
	if fs.Revert || fs.Override {
		// revert to the marker value
		if strings.Contains(field.YNode().Value, s.Value) {
			fs.Modified = true // modified the config
			field.YNode().Value = strings.ReplaceAll(field.YNode().Value, s.Value, s.Marker)
		}
	}
	if fs.Revert {
		s.Value = "" // value has been cleared and replaced with marker
		if err := fm.Write(field); err != nil {
			return nil, err
		}
		return object, nil
	}

	if !strings.Contains(field.YNode().Value, s.Marker) {
		// no substitutions necessary
		return object, nil
	}

	// replace the marker with the new value
	// be sure to set the tag so the yaml doesn't incorrectly quote ints, bools or floats
	fs.Modified = true // modified the config
	field.YNode().Value = strings.ReplaceAll(field.YNode().Value, fs.Marker, fs.StringValue)
	field.YNode().Tag = fs.Type.Tag()
	field.YNode().Style = 0

	// update the comment
	s.Value = fs.StringValue
	if err := fm.Write(field); err != nil {
		return nil, err
	}
	return object, nil
}
