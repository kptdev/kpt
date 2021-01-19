package setters

import (
	"fmt"
	"strings"

	"github.com/go-openapi/spec"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Set sets resource field values from an OpenAPI setter
type Set struct {
	// Name is the name of the setter to set on the object.  i.e. matches the x-k8s-cli.setter.name
	// of the setter that should have its value applied to fields which reference it.
	Name string

	// Count is the number of fields that were updated by calling Filter
	Count int

	// SettersSchema is openapi schema of all the setters in the packages from Kptfile
	SettersSchema *spec.Schema
}

// Filter implements Set as a yaml.Filter
func (s *Set) Filter(object *yaml.RNode) (*yaml.RNode, error) {
	return object, accept(s, object, s.SettersSchema)
}

func (s *Set) visitMapping(_ *yaml.RNode, p string, _ *SetterInfos) error {
	return nil
}

// isMatch checks if the the name of the input setter matches with any of the
// setters present in the setterInfos
// e.g. is input setter name is "image" and the comment on the yaml node is
// ${image}-${tag} there is a match
func (s *Set) isMatch(setterInfos *SetterInfos) bool {
	if setterInfos == nil {
		return false
	}
	nameMatch := false
	for k := range setterInfos.SetterDefinitions {
		if s.Name == k {
			nameMatch = true
		}
	}
	return nameMatch
}

// visitSequence will perform setters for sequences
func (s *Set) visitSequence(object *yaml.RNode, p string, setterInfos *SetterInfos) error {
	if !s.isMatch(setterInfos) {
		return nil
	}
	s.Count++
	// set the values on the sequences
	var elements []*yaml.Node
	if len(setterInfos.SetterDefinitions) > 1 {
		return nil
	}
	var listValues []string
	for _, schema := range setterInfos.SetterDefinitions {
		listValues = ListValues(fmt.Sprintf("%v", schema.Default), " ")
	}
	for i := range listValues {
		v := listValues[i]
		n := yaml.NewScalarRNode(v).YNode()
		n.Style = yaml.DoubleQuotedStyle
		elements = append(elements, n)
	}
	object.YNode().Content = elements
	object.YNode().Style = yaml.FoldedStyle
	return nil
}

// visitScalar
func (s *Set) visitScalar(object *yaml.RNode, p string, setterInfos *SetterInfos) error {
	if !s.isMatch(setterInfos) {
		return nil
	}
	s.Count++
	// perform a direct set of the field if it matches
	s.set(object, setterInfos)
	return nil
}

// set applies the value from ext to field if its name matches s.Name
func (s *Set) set(field *yaml.RNode, setterInfos *SetterInfos) {
	// check full setter
	fieldValue := setterInfos.SetterPattern
	for setterName, setterSchema := range setterInfos.SetterDefinitions {
		// this has a full setter, set its value
		fieldValue = strings.ReplaceAll(fieldValue, fmt.Sprintf("${%s}", setterName), fmt.Sprintf("%v", setterSchema.Default))

		// format the node so it is quoted if it is a string. If there is
		// type information on the setter schema, we use it. Otherwise we
		// fall back to the field schema if it exists.
		if len(setterSchema.Type) > 0 {
			yaml.FormatNonStringStyle(field.YNode(), *setterSchema)
		}
	}
	field.YNode().Value = fieldValue
}

// SetAll applies the set filter for all yaml nodes and only returns the nodes whose
// corresponding file has at least one node with input setter
func SetAll(s *Set) kio.Filter {
	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		filesToUpdate := sets.String{}
		// for each node record the set fields count before and after filter is applied and
		// store the corresponding file paths if there is an increment in setters count
		for i := range nodes {
			preCount := s.Count
			_, err := s.Filter(nodes[i])
			if err != nil {
				return nil, errors.Wrap(err)
			}
			if s.Count > preCount {
				path, _, err := kioutil.GetFileAnnotations(nodes[i])
				if err != nil {
					return nil, errors.Wrap(err)
				}
				filesToUpdate.Insert(path)
			}
		}
		var nodesInUpdatedFiles []*yaml.RNode
		// return only the nodes whose corresponding file has at least one node with input setter
		for i := range nodes {
			path, _, err := kioutil.GetFileAnnotations(nodes[i])
			if err != nil {
				return nil, errors.Wrap(err)
			}
			if filesToUpdate.Has(path) {
				nodesInUpdatedFiles = append(nodesInUpdatedFiles, nodes[i])
			}
		}
		return nodesInUpdatedFiles, nil
	})
}

// ListValues takes a list in the form of string and returns the list values based on delimiter
// returns nil, if the input is not enclosed in []
func ListValues(setterValue, delimiter string) []string {
	if !strings.HasPrefix(setterValue, "[") || !strings.HasSuffix(setterValue, "]") {
		return nil
	}
	commaSepVals := strings.TrimSuffix(strings.TrimPrefix(setterValue, "["), "]")
	listValues := strings.Split(commaSepVals, delimiter)
	for i := range listValues {
		listValues[i] = strings.TrimSpace(listValues[i])
	}
	return listValues
}
