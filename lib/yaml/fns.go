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

package yaml

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v3"
)

// Append creates an ElementAppender
func Append(elements ...*yaml.Node) ElementAppender {
	return ElementAppender{Elements: elements}
}

// ElementAppender adds all element to a SequenceNode's Content.
// Returns Elements[0] if len(Elements) == 1, otherwise returns nil.
type ElementAppender struct {
	Kind string `yaml:"kind,omitempty"`

	// Elem is the value to append.
	Elements []*yaml.Node `yaml:"elements,omitempty"`
}

func (a ElementAppender) Filter(rn *RNode) (*RNode, error) {
	if err := ErrorIfInvalid(rn, yaml.SequenceNode); err != nil {
		return nil, err
	}
	for i := range a.Elements {
		rn.YNode().Content = append(rn.Content(), a.Elements[i])
	}
	if len(a.Elements) == 1 {
		return NewRNode(a.Elements[0]), nil
	}
	return nil, nil
}

// Clear returns a FieldClearer
func Clear(name string) FieldClearer {
	return FieldClearer{Name: name}
}

// FieldClearer removes the field or map key.
// Returns a RNode with the removed field or map entry.
type FieldClearer struct {
	Kind string `yaml:"kind,omitempty"`

	// Name is the name of the field or key in the map.
	Name string `yaml:"name,omitempty"`

	IfEmpty bool `yaml:"ifEmpty,omitempty"`
}

func (c FieldClearer) Filter(rn *RNode) (*RNode, error) {
	if err := ErrorIfInvalid(rn, yaml.MappingNode); err != nil {
		return nil, err
	}

	for i := 0; i < len(rn.Content()); i += 2 {

		// if name matches, remove these 2 elements from the list because
		// they are treated as a fieldName/fieldValue pair.
		if rn.Content()[i].Value == c.Name {
			if c.IfEmpty {
				if len(rn.Content()[i+1].Content) > 0 {
					continue
				}
			}

			// save the item we are about to remove
			removed := NewRNode(rn.Content()[i+1])
			if len(rn.YNode().Content) > i+2 {
				// remove from the middle of the list
				rn.YNode().Content = append(
					rn.Content()[:i],
					rn.Content()[i+2:len(rn.YNode().Content)]...)
			} else {
				// remove from the end of the list
				rn.YNode().Content = rn.Content()[:i]
			}

			// return the removed field name and value
			return removed, nil
		}
	}
	// nothing removed
	return nil, nil
}

func MatchElement(field, value string) ElementMatcher {
	return ElementMatcher{FieldName: field, FieldValue: value}
}

// ElementMatcher returns the first element from a Sequence matching the
// specified field's value.
type ElementMatcher struct {
	Kind string `yaml:"kind,omitempty"`

	// FieldName will attempt to match this field in each list element.
	// Optional.  Leave empty for lists of primitives (ScalarNode).
	FieldName string `yaml:"name,omitempty"`

	// FieldValue will attempt to match each element field to this value.
	// For lists of primitives, this will be used to match the primitive value.
	FieldValue string `yaml:"value,omitempty"`

	// Create will create the Element if it is not found
	Create *RNode `yaml:"create,omitempty"`
}

func (e ElementMatcher) Filter(rn *RNode) (*RNode, error) {
	if err := ErrorIfInvalid(rn, yaml.SequenceNode); err != nil {
		return nil, err
	}

	// SequenceNode Content is a slice of ScalarNodes.  Each ScalarNode has a
	// YNode containing the primitive data.
	if len(e.FieldName) == 0 {
		for i := range rn.Content() {
			if rn.Content()[i].Value == e.FieldValue {
				return &RNode{value: rn.Content()[i]}, nil
			}
		}
		if e.Create != nil {
			return rn.Pipe(Append(e.Create.YNode()))
		}
		return nil, nil
	}

	// SequenceNode Content is a slice of MappingNodes.  Each MappingNode has Content
	// with a slice of key-value pairs containing the fields.
	for i := range rn.Content() {
		// cast the entry to a RNode so we can operate on it
		elem := NewRNode(rn.Content()[i])

		field, err := elem.Pipe(MatchField(e.FieldName, e.FieldValue))
		if IsFoundOrError(field, err) {
			return elem, err
		}
	}

	// create the element
	if e.Create != nil {
		return rn.Pipe(Append(e.Create.YNode()))
	}

	return nil, nil
}

func Get(name string) FieldMatcher {
	return FieldMatcher{Name: name}
}

func MatchField(name, value string) FieldMatcher {
	return FieldMatcher{Name: name, Value: NewScalarRNode(value)}
}

func Match(value string) FieldMatcher {
	return FieldMatcher{Value: NewScalarRNode(value)}
}

// FieldMatcher returns the value of a named field or map entry.
type FieldMatcher struct {
	Kind string `yaml:"kind,omitempty"`

	// Name of the field to return
	Name string `yaml:"path,omitempty"`

	// YNode of the field to return.
	// Optional.  Will only need to match field name if unset.
	Value *RNode `yaml:"value,omitempty"`

	StringValue string `yaml:"stringValue,omitempty"`

	// Create will cause the field to be created with this value
	// if it is set.
	Create *RNode `yaml:"create,omitempty"`
}

func (f FieldMatcher) Filter(rn *RNode) (*RNode, error) {
	if f.StringValue != "" && f.Value == nil {
		f.Value = NewScalarRNode(f.StringValue)
	}

	if f.Name == "" {
		if err := ErrorIfInvalid(rn, yaml.ScalarNode); err != nil {
			return nil, err
		}
		if rn.value.Value == f.Value.YNode().Value {
			return rn, nil
		} else {
			return nil, nil
		}
	}

	if err := ErrorIfInvalid(rn, yaml.MappingNode); err != nil {
		return nil, err
	}

	for i := 0; i < len(rn.Content()); IncrementFieldIndex(&i) {
		isMatchingField := rn.Content()[i].Value == f.Name
		if isMatchingField {
			requireMatchFieldValue := f.Value != nil
			if !requireMatchFieldValue || rn.Content()[i+1].Value == f.Value.YNode().Value {
				return NewRNode(rn.Content()[i+1]), nil
			}
		}
	}

	if f.Create != nil {
		return rn.Pipe(SetField(f.Name, f.Create))
	}

	return nil, nil
}

func Lookup(path ...string) PathGetter {
	return PathGetter{Path: path}
}

func LookupCreate(kind yaml.Kind, path ...string) PathGetter {
	return PathGetter{Path: path, Create: kind}
}

// PathGetter returns the RNode under Path.
type PathGetter struct {
	Kind string `yaml:"kind,omitempty"`

	// Path is a slice of parts leading to the RNode to lookup.
	// Each path part may be one of:
	// * FieldMatcher -- e.g. "spec"
	// * Map Key -- e.g. "app.k8s.io/version"
	// * List Entry -- e.g. "[name=nginx]" or "[=-jar]"
	//
	// Map Keys and Fields are equivalent.
	// See FieldMatcher for more on Fields and Map Keys.
	//
	// List Entries are specified as map entry to match [fieldName=fieldValue].
	// See Elem for more on List Entries.
	//
	// Examples:
	// * spec.template.spec.container with matching name: [name=nginx]
	// * spec.template.spec.container.argument matching a value: [=-jar]
	Path []string `yaml:"path,omitempty"`

	// Create will cause missing path parts to be created as they are walked.
	//
	// * The leaf Node (final path) will be created with a Kind matching Create
	// * Intermediary Nodes will be created as either a MappingNodes or
	//   SequenceNodes as appropriate for each's Path location.
	Create yaml.Kind `yaml:"create,omitempty"`

	// Style is the style to apply to created value Nodes.
	// Created key Nodes keep an unspecified Style.
	Style yaml.Style `yaml:"style,omitempty"`
}

func (l PathGetter) Filter(rn *RNode) (*RNode, error) {
	var err error
	fieldPath := append([]string{}, rn.FieldPath()...)
	match := rn

	// iterate over path until encountering an error or missing value
	l.cleanPath()
	for i := range l.Path {
		var part, nextPart string
		part = l.Path[i]
		if len(l.Path) > i+1 {
			nextPart = l.Path[i+1]
		}
		if IsListIndex(part) {
			match, err = l.doElem(match, part)
		} else {
			fieldPath = append(fieldPath, part)
			match, err = l.doField(match, part, l.getKind(nextPart))
		}
		if IsMissingOrError(match, err) {
			return nil, err
		}
		match.AppendToFieldPath(fieldPath...)
	}
	return match, nil
}

func (l PathGetter) doElem(rn *RNode, part string) (*RNode, error) {
	var match *RNode
	name, value, err := SplitIndexNameValue(part)
	if err != nil {
		return nil, err
	}
	if !IsCreate(l.Create) {
		return rn.Pipe(MatchElement(name, value))
	}

	var elem *RNode
	primitiveElement := len(name) == 0
	if primitiveElement {
		// append a ScalarNode
		elem = NewScalarRNode(value)
		elem.YNode().Style = l.Style
		match = elem
	} else {
		// append a MappingNode
		match = NewRNode(&yaml.Node{Kind: yaml.ScalarNode, Value: value, Style: l.Style})
		elem = NewRNode(&yaml.Node{
			Kind:    yaml.MappingNode,
			Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: name}, match.YNode()},
			Style:   l.Style,
		})
	}
	// Append the Node
	return rn.Pipe(ElementMatcher{FieldName: name, FieldValue: value, Create: elem})
}

func (l PathGetter) doField(
	rn *RNode, name string, kind yaml.Kind) (*RNode, error) {
	if !IsCreate(l.Create) {
		return rn.Pipe(Get(name))
	}
	return rn.Pipe(FieldMatcher{Name: name, Create: &RNode{value: &yaml.Node{Kind: kind, Style: l.Style}}})
}

func (l *PathGetter) cleanPath() {
	var p []string
	for _, elem := range l.Path {
		elem = strings.TrimSpace(elem)
		if len(elem) == 0 {
			continue
		}
		p = append(p, elem)
	}
	l.Path = p
}

func (l PathGetter) getKind(nextPart string) yaml.Kind {
	if IsListIndex(nextPart) {
		// if nextPart is of the form [a=b], then it is an index into a Sequence
		// so the current part must be a SequenceNode
		return yaml.SequenceNode
	}
	if nextPart == "" {
		// final name in the path, use the l.Create defined Kind
		return l.Create
	}

	// non-sequence intermediate Node
	return yaml.MappingNode
}

func SetField(name string, value *RNode) FieldSetter {
	return FieldSetter{Name: name, Value: value}
}

func Set(value *RNode) FieldSetter {
	return FieldSetter{Value: value}
}

// FieldSetter sets a field or map entry to a value.
type FieldSetter struct {
	Kind string `yaml:"kind,omitempty"`

	// Name is the name of the field or key to lookup in a MappingNode.
	// If Name is unspecified, and the input is a ScalarNode, FieldSetter will set the
	// value on the ScalarNode.
	Name string `yaml:"name,omitempty"`

	// YNode is the value to set.
	// Optional if Kind is set.
	Value *RNode `yaml:"value,omitempty"`

	StringValue string `yaml:"stringValue,omitempty"`
}

// FieldSetter returns an Filter that sets the named field to the given value.
func (s FieldSetter) Filter(rn *RNode) (*RNode, error) {
	if s.StringValue != "" && s.Value == nil {
		s.Value = NewScalarRNode(s.StringValue)
	}

	if s.Name == "" {
		if err := ErrorIfInvalid(rn, yaml.ScalarNode); err != nil {
			return rn, err
		}
		rn.SetYNode(s.Value.YNode())
		return rn, nil
	}

	if s.Value == nil {
		return rn.Pipe(Clear(s.Name))
	}

	field, err := rn.Pipe(FieldMatcher{Name: s.Name})
	if err != nil {
		return nil, err
	}
	if field != nil {
		// need to def ref the Node since field is ephemeral
		field.SetYNode(s.Value.YNode())
		return field, nil
	}

	// create the field
	rn.YNode().Content = append(rn.YNode().Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: s.Name},
		s.Value.YNode())
	return s.Value, nil
}

// Tee calls the provided Filters, and returns its argument rather than the result
// of the filters.
// May be used to fork sub-filters from a call.
// e.g. locate field, set value; locate another field, set another value
func Tee(filters ...Filter) Filter {
	return TeePiper{Filters: filters}
}

// TeePiper Calls a slice of Filters and returns its input.
// May be used to fork sub-filters from a call.
// e.g. locate field, set value; locate another field, set another value
type TeePiper struct {
	Kind string `yaml:"kind,omitempty"`

	// Filters are the set of Filters run by TeePiper.
	Filters []Filter `yaml:"filters,omitempty"`
}

func (t TeePiper) Filter(rn *RNode) (*RNode, error) {
	_, err := rn.Pipe(t.Filters...)
	return rn, err
}

// IsCreate returns true if kind is specified
func IsCreate(kind yaml.Kind) bool {
	return kind != 0
}

// IsMissingOrError returns true if rn is NOT found or err is non-nil
func IsMissingOrError(rn *RNode, err error) bool {
	return rn == nil || err != nil
}

// IsFoundOrError returns true if rn is found or err is non-nil
func IsFoundOrError(rn *RNode, err error) bool {
	return rn != nil || err != nil
}

func ErrorIfAnyInvalid(kind yaml.Kind, rn ...*RNode) error {
	for i := range rn {
		if err := ErrorIfInvalid(rn[i], kind); err != nil {
			return err
		}
	}
	return nil
}

func ErrorIfInvalid(rn *RNode, kind yaml.Kind) error {
	if rn == nil || rn.YNode() == nil {
		return fmt.Errorf("missing value")
	}

	if rn.YNode().Kind != kind {
		s, _ := rn.String()
		return fmt.Errorf(
			"wrong Node Kind for %s expected: %v was %v: value: {%s}",
			strings.Join(rn.FieldPath(), "."),
			kind, rn.YNode().Kind, strings.TrimSpace(s))
	}

	if kind == yaml.MappingNode {
		if len(rn.YNode().Content)%2 != 0 {
			return fmt.Errorf("yaml MappingNodes must have even length contents: %v", spew.Sdump(rn))
		}
	}

	return nil
}

// IsListIndex returns true if p is an index into a Seq.
// e.g. [fieldName=fieldValue]
// e.g. [=primitiveValue]
func IsListIndex(p string) bool {
	return strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]")
}

// SplitIndexNameValue splits a lookup part Seq index into the field name
// and field value to match.
// e.g. splits [name=nginx] into (name, nginx)
// e.g. splits [=-jar] into ("", jar)
func SplitIndexNameValue(p string) (string, string, error) {
	elem := strings.TrimSuffix(p, "]")
	elem = strings.TrimPrefix(elem, "[")
	parts := strings.SplitN(elem, "=", 2)
	if len(parts) == 1 {
		return "", "", fmt.Errorf("list path element must contain fieldName=fieldValue for element to match")
	}
	return parts[0], parts[1], nil
}

// IncrementFieldIndex increments i to point to the next field name element in
// a slice of Contents.
func IncrementFieldIndex(i *int) {
	*i = *i + 2
}
