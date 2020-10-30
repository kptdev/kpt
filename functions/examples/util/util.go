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

// Package util provides utilities for developing kpt-functions.
package util

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Template is a template to be parsed
type Template struct {
	// Input is the input to a template.  Typically the API.
	Input interface{}

	// Name is the name of the template.  Used if there is an error.
	Name string

	// Template is the string template to be parsed.
	Template string
}

// MustParseAll parses the Resources from a slice of templates, exiting non-0 if
// there is an error.
func MustParseAll(inputs ...Template) []*yaml.RNode {
	r, err := ParseAll(inputs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	return r
}

// ParseAll parses the Resources from a slice of templates
func ParseAll(inputs ...Template) ([]*yaml.RNode, error) {
	var results []*yaml.RNode
	for i := range inputs {
		r, err := inputs[i].parse()
		if err != nil {
			return nil, err
		}
		results = append(results, r...)
	}
	return results, nil
}

func (tm Template) parse() ([]*yaml.RNode, error) {
	t, err := template.New(tm.Name).Parse(tm.Template)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	if err := t.Execute(b, tm.Input); err != nil {
		return nil, err
	}
	values := strings.Split(b.String(), "\n---\n")
	var ret []*yaml.RNode
	for i := range values {
		v, err := yaml.Parse(values[i])
		if err != nil {
			return nil, err
		}
		ret = append(ret, v)
	}
	return ret, nil
}

// SetFieldSetter
func SetSetter(n *yaml.RNode, o string) error {
	if o == "" {
		// no-op
		return nil
	}
	fm := fieldmeta.FieldMeta{}
	if err := fm.Read(n); err != nil {
		return err
	}
	fm.Extensions.SetBy = o
	return fm.Write(n)
}

func SetSetters(object *yaml.RNode, o string) error {
	return setSetters(object, o, true, false, "")
}

func setSetters(object *yaml.RNode, o string, root, meta bool, assc string) error {
	switch object.YNode().Kind {
	case yaml.DocumentNode:
		return setSetters(yaml.NewRNode(object.YNode().Content[0]), o, true, false, assc)
	case yaml.MappingNode:
		return object.VisitFields(func(node *yaml.MapNode) error {
			// special case unique scalars
			if node.Key.YNode().Value == assc {
				return nil
			}
			if root && node.Key.YNode().Value == "apiVersion" {
				return nil
			}
			if root && node.Key.YNode().Value == "kind" {
				return nil
			}
			if meta && node.Key.YNode().Value == "name" {
				return nil
			}
			if meta && node.Key.YNode().Value == "namespace" {
				return nil
			}
			if root && node.Key.YNode().Value == "metadata" {
				return setSetters(node.Value, o, false, true, "")
			}
			// no longer an associative key candidate
			return setSetters(node.Value, o, false, false, "")
		})
	case yaml.SequenceNode:
		// never set owner for associative keys -- the keys are shared across
		// all owners of the element
		key := object.GetAssociativeKey()
		return object.VisitElements(func(node *yaml.RNode) error {
			return setSetters(node, o, false, false, key)
		})
	case yaml.ScalarNode:
		return SetSetter(object, o)
	default:
		return nil
	}
}
