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

package setters

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/go-openapi/spec"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// visitor is implemented by structs which need to walk the configuration.
// visitor is provided to accept to walk configuration
type visitor interface {
	// visitScalar is called for each scalar field value on a resource
	// node is the scalar field value
	// path is the path to the field; path elements are separated by '.'
	// oa is the OpenAPI schema for the field
	visitScalar(node *yaml.RNode, path string, setterInfos *SetterInfos) error

	// visitSequence is called for each sequence field value on a resource
	// node is the sequence field value
	// path is the path to the field
	// oa is the OpenAPI schema for the field
	visitSequence(node *yaml.RNode, path string, setterInfos *SetterInfos) error

	// visitMapping is called for each Mapping field value on a resource
	// node is the mapping field value
	// path is the path to the field
	// oa is the OpenAPI schema for the field
	visitMapping(node *yaml.RNode, path string, setterInfos *SetterInfos) error
}

// accept invokes the appropriate function on v for each field in object
func accept(v visitor, object *yaml.RNode, settersSchema *spec.Schema) error {
	// get the OpenAPI for the type if it exists
	return acceptImpl(v, object, "", nil, settersSchema)
}

// acceptImpl implements accept using recursion
func acceptImpl(v visitor, object *yaml.RNode, p string, setterInfos *SetterInfos, settersSchema *spec.Schema) error {
	switch object.YNode().Kind {
	case yaml.DocumentNode:
		// Traverse the child of the document
		return accept(v, yaml.NewRNode(object.YNode()), settersSchema)
	case yaml.MappingNode:
		if err := v.visitMapping(object, p, setterInfos); err != nil {
			return err
		}
		return object.VisitFields(func(node *yaml.MapNode) error {
			// Traverse each field value
			setterInfos := getSchemas(node.Key, settersSchema)
			return acceptImpl(v, node.Value, p+"."+node.Key.YNode().Value, setterInfos, settersSchema)
		})
	case yaml.SequenceNode:
		// get the schema for the sequence node, use the schema provided if not present
		// on the field
		if err := v.visitSequence(object, p, setterInfos); err != nil {
			return err
		}
		// get the schema for the elements
		setterInfos := getSchemas(object, settersSchema)
		return object.VisitElements(func(node *yaml.RNode) error {
			// Traverse each list element
			return acceptImpl(v, node, p, setterInfos, settersSchema)
		})
	case yaml.ScalarNode:
		// Visit the scalar field
		setterInfos := getSchemas(object, settersSchema)
		return v.visitScalar(object, p, setterInfos)
	}
	return nil
}

type SetterInfos struct {
	// SetterPattern is the pattern of setter in comment
	// e.g. ${image}-${tag}
	SetterPattern string

	// SetterDefinitions is the map of name to setters schema for the setters
	// present in SetterPattern
	SetterDefinitions map[string]*spec.Schema

	// SetterValues is the map of setter name to value provided by package
	// consumer
	SetterValues map[string]string
}

// getSchemas returns OpenAPI schemas for an RNode or field of the
// RNode.
// r is the Node with setter pattern comment to get the Schemas for
// s is the provided schema for all the setters in package
// field is the name of the field
func getSchemas(r *yaml.RNode, settersSchema *spec.Schema) *SetterInfos {
	comment := r.YNode().LineComment
	if comment == "" {
		return nil
	}
	return setterInfos(comment, settersSchema)
}

// setterInfos takes the setter pattern comment and settersSchema for all
// setters in package and returns the SetterInfos struct instance for the
// setters present in the setter pattern
func setterInfos(comment string, settersSchema *spec.Schema) *SetterInfos {
	comment = strings.TrimLeft(comment, "#")
	input := map[string]string{}
	err := json.Unmarshal([]byte(comment), &input)
	if err != nil {
		return nil
	}
	setterPattern := input[setterRef]
	if setterPattern == "" {
		return nil
	}
	// extract setter name tokens from pattern enclosed in ${}
	re := regexp.MustCompile(`\$\{([^}]*)\}`)
	markers := re.FindAllString(setterPattern, -1)
	if len(markers) == 0 {
		return nil
	}
	res := make(map[string]*spec.Schema)
	for _, marker := range markers {
		name := strings.TrimSuffix(strings.TrimPrefix(marker, "${"), "}")
		sch := settersSchema.Definitions[name]
		res[name] = &sch
	}
	setterInfos := &SetterInfos{
		SetterDefinitions: res,
		SetterPattern:     setterPattern,
	}
	return setterInfos
}
