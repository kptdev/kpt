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
	"bytes"
	"errors"
	"reflect"

	"gopkg.in/yaml.v3"
)

const (
	// NullNodeTag is the tag set for a yaml.Document that contains no data -- e.g. it isn't a
	// Map, Slice, Document, etc
	NullNodeTag = "!!null"
)

func IsEmpty(node *RNode) bool {
	return node == nil || node.YNode() == nil || node.YNode().Tag == NullNodeTag
}

func IsFieldEmpty(node *MapNode) bool {
	return node == nil || node.Value == nil || node.Value.YNode() == nil ||
		node.Value.YNode().Tag == NullNodeTag
}

// Parser parses values into configuration.
type Parser struct {
	Kind  string `yaml:"kind,omitempty"`
	Value string `yaml:"value,omitempty"`
}

func (p Parser) Filter(_ *RNode) (*RNode, error) {
	d := yaml.NewDecoder(bytes.NewBuffer([]byte(p.Value)))
	o := &RNode{value: &yaml.Node{}}
	return o, d.Decode(o.value)
}

// Parse parses a yaml string into an *RNode
func Parse(value string) (*RNode, error) {
	return Parser{Value: value}.Filter(nil)
}

// MustParse parses a yaml string into an *RNode and panics if there is an error
func MustParse(value string) *RNode {
	v, err := Parser{Value: value}.Filter(nil)
	if err != nil {
		panic(err)
	}
	return v
}

// NewScalarRNode returns a new Scalar *RNode containing the provided value.
func NewScalarRNode(value string) *RNode {
	return &RNode{
		value: &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: value,
		}}
}

// NewListRNode returns a new List *RNode containing the provided value.
func NewListRNode(values ...string) *RNode {
	seq := &RNode{value: &yaml.Node{Kind: yaml.SequenceNode}}
	for _, v := range values {
		seq.value.Content = append(seq.value.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: v,
		})
	}
	return seq
}

// NewRNode returns a new *RNode containing the provided value.
func NewRNode(value *yaml.Node) *RNode {
	value.Style = 0
	return &RNode{value: value}
}

// Filter may modify or walk the RNode.
// When possible, Filters should be serializable to yaml so that they can be described
// declaratively as data.
//
// Analogous to http://www.linfo.org/filters.html
type Filter interface {
	Filter(object *RNode) (*RNode, error)
}

type FilterFunc func(object *RNode) (*RNode, error)

func (f FilterFunc) Filter(object *RNode) (*RNode, error) {
	return f(object)
}

// RNode provides functions for manipulating Kubernetes Resources
// Objects unmarshalled into *yaml.Nodes
type RNode struct {
	// fieldPath contains the path from the root of the KubernetesObject to
	// this field.
	// Only field names are captured in the path.
	// e.g. a image field in a Deployment would be
	// 'spec.template.spec.containers.image'
	fieldPath []string

	// FieldValue contains the value.
	// FieldValue is always set:
	// field: field value
	// list entry: list entry value
	// object root: object root
	value *yaml.Node
}

// MapNode wraps a field key and value.
type MapNode struct {
	Key   *RNode
	Value *RNode
}

// ResourceMeta contains the metadata for a Resource.
type ResourceMeta struct {
	ApiVersion string `yaml:"apiVersion,omitempty"`
	Kind       string `yaml:"kind,omitempty"`
	ObjectMeta `yaml:"metadata,omitempty"`
}

func NewResourceMeta(name string, typeMeta ResourceMeta) ResourceMeta {
	return ResourceMeta{
		Kind:       typeMeta.Kind,
		ApiVersion: typeMeta.ApiVersion,
		ObjectMeta: ObjectMeta{Name: name},
	}
}

type ObjectMeta struct {
	Name        string            `yaml:"name,omitempty"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

var MissingMetaError = errors.New("missing Resource metadata")

func (rn *RNode) GetMeta() (ResourceMeta, error) {
	m := ResourceMeta{}
	b := &bytes.Buffer{}
	e := NewEncoder(b)
	if err := e.Encode(rn.YNode()); err != nil {
		return m, err
	}
	if err := e.Close(); err != nil {
		return m, err
	}
	d := yaml.NewDecoder(b)
	d.KnownFields(false) // only want to parse the metadata
	if err := d.Decode(&m); err != nil {
		return m, err
	}
	if reflect.DeepEqual(m, ResourceMeta{}) {
		return m, MissingMetaError
	}
	return m, nil
}

// Pipe sequentially invokes each Filter, and passes the result to the next
// Filter.
//
// Analogous to http://www.linfo.org/pipes.html
//
// * rn is provided as input to the first Filter.
// * if any Filter returns an error, immediately return the error
// * if any Filter returns a nil RNode, immediately return nil, nil
// * if all Filters succeed with non-empty results, return the final result
func (rn *RNode) Pipe(functions ...Filter) (*RNode, error) {
	// check if rn is nil to make chaining Pipe calls easier
	if rn == nil {
		return nil, nil
	}

	var err error
	var v *RNode
	if rn.value != nil && rn.value.Kind == yaml.DocumentNode {
		// the first node may be a DocumentNode containing a single MappingNode
		v = &RNode{value: rn.value.Content[0]}
	} else {
		v = rn
	}

	// return each fn in sequence until encountering an error or missing value
	for _, c := range functions {
		v, err = c.Filter(v)
		if err != nil || v == nil {
			return v, err
		}
	}
	return v, err
}

// Document returns the Node RNode for the value.  Does not unwrap the node if it is a
// DocumentNodes
func (rn *RNode) Document() *yaml.Node {
	return rn.value
}

// YNode returns the yaml.Node value.  If the yaml.Node value is a DocumentNode,
// YNode will return the DocumentNode Content entry instead of the DocumentNode.
func (rn *RNode) YNode() *yaml.Node {
	if rn == nil || rn.value == nil {
		return nil
	}
	if rn.value.Kind == yaml.DocumentNode {
		return rn.value.Content[0]
	}
	return rn.value
}

// SetYNode sets the yaml.Node value.
func (rn *RNode) SetYNode(node *yaml.Node) {
	if rn.value == nil || node == nil {
		rn.value = node
		return
	}
	*rn.value = *node
}

// SetYNode sets the value on a Document.
func (rn *RNode) AppendToFieldPath(parts ...string) {
	rn.fieldPath = append(rn.fieldPath, parts...)
}

// FieldPath returns the field path from the object root to rn.  Does not include list indexes.
func (rn *RNode) FieldPath() []string {
	return rn.fieldPath
}

// NewScalarRNode returns the yaml NewScalarRNode representation of the RNode value.
func (rn *RNode) String() (string, error) {
	if rn == nil || rn.value == nil {
		return "", nil
	}
	b := &bytes.Buffer{}
	e := NewEncoder(b)
	err := e.Encode(rn.value)
	e.Close()
	return b.String(), err
}

// Content returns the value node's Content field.
func (rn *RNode) Content() []*yaml.Node {
	return rn.YNode().Content
}

// Fields returns the list of fields for a ResourceNode containing a MappingNode
// value.
func (rn *RNode) Fields() ([]string, error) {
	if err := ErrorIfInvalid(rn, yaml.MappingNode); err != nil {
		return nil, err
	}
	var fields []string
	for i := 0; i < len(rn.Content()); i += 2 {
		fields = append(fields, rn.Content()[i].Value)
	}
	return fields, nil
}

// Field returns the fieldName, fieldValue pair for MappingNodes.  Returns nil for non-MappingNodes.
func (rn *RNode) Field(field string) *MapNode {
	if rn.YNode().Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(rn.Content()); IncrementFieldIndex(&i) {
		isMatchingField := rn.Content()[i].Value == field
		if isMatchingField {
			return &MapNode{Key: NewRNode(rn.Content()[i]), Value: NewRNode(rn.Content()[i+1])}
		}
	}
	return nil
}

// VisitFields calls fn for each field in rn.
func (rn *RNode) VisitFields(fn func(node *MapNode) error) error {
	// get the list of srcFieldNames
	srcFieldNames, err := rn.Fields()
	if err != nil {
		return err
	}

	// visit each field
	for _, fieldName := range srcFieldNames {
		if err := fn(rn.Field(fieldName)); err != nil {
			return err
		}
	}
	return nil
}

// Elements returns a list of elements for a ResourceNode containing a
// SequenceNode value.
func (rn *RNode) Elements() ([]*RNode, error) {
	if err := ErrorIfInvalid(rn, yaml.SequenceNode); err != nil {
		return nil, err
	}
	var elements []*RNode
	for i := 0; i < len(rn.Content()); i += 1 {
		elements = append(elements, NewRNode(rn.Content()[i]))
	}
	return elements, nil
}

// Element returns the element in the list which contains the field matching the value.
// Returns nil for non-SequenceNodes
func (rn *RNode) Element(key, value string) *RNode {
	if rn.YNode().Kind != yaml.SequenceNode {
		return nil
	}
	elem, err := rn.Pipe(MatchElement(key, value))
	if err != nil {
		return nil
	}
	return elem
}

// VisitElements calls fn for each element in the list.
func (rn *RNode) VisitElements(fn func(node *RNode) error) error {
	elements, err := rn.Elements()
	if err != nil {
		return err
	}

	for i := range elements {
		if err := fn(elements[i]); err != nil {
			return err
		}
	}
	return nil
}

// AssociativeSequencePaths is a map of paths to sequences that have associative keys.
// The order sets the precedence of the merge keys -- if multiple keys are present
// in the list, then the FIRST key which ALL elements have is used as the
// associative key.
var AssociativeSequenceKeys = []string{
	"mountPath", "devicePath", "ip", "type", "topologyKey", "name", "containerPort",
}

// IsAssociative returns true if the RNode is for an associative list.
func (rn *RNode) IsAssociative() bool {
	key := rn.GetAssociativeKey()
	return key != ""
}

// GetAssociativeKey returns the associative key used to merge the list, or "" if the
// list is not associative.
func (rn *RNode) GetAssociativeKey() string {
	// look for any associative keys in the first element
	for _, key := range AssociativeSequenceKeys {
		if checkKey(key, rn.Content()) {
			return key
		}
	}

	// element doesn't have an associative keys
	return ""
}

// checkKey returns true if all elems have the key
func checkKey(key string, elems []*Node) bool {
	for i := range elems {
		elem := NewRNode(elems[i])
		if elem.Field(key) == nil {
			return false
		}
	}
	return true
}
