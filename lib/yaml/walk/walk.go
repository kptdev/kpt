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

package walk

import (
	"fmt"

	"lib.kpt.dev/yaml"
)

// Visitor is invoked by walk with source and destination node pairs
type Visitor interface {
	// SetMapField is called by walk when a field is missing or null on either the source
	// or destination.
	//
	// If a non-nil, non-empty RNode is returned it will be set on the destination.
	// If an empty RNode (e.g. null) is returned, the field will be cleared from the destination
	// If a nil RNode is returned, no action will be taken
	SetMapField(source, dest *yaml.MapNode) (*yaml.RNode, error)

	// SetScalarValue is called by walk on each Scalar node.
	//
	// If a non-nil, non-empty RNode is returned it will be set on the destination.
	// If an empty RNode (e.g. null) is returned, the field will be cleared from the destination
	// If a nil RNode is returned, no action will be taken
	SetScalarValue(source, dest *yaml.RNode) (*yaml.RNode, error)

	// SetElement is called by walk on elements in source lists that contain elements with an
	// associative key.  SetElement is only called on elements which are missing from the
	// destination list.  See yaml.AssociativeSequenceKeys for the list of recognized keys.
	// There is no way to delete destination elements in associative lists.
	// TODO: Support removing elements from associative lists -- maybe use SMP syntax
	//
	// If a non-nil, non-empty RNode is returned it will be set on the destination.
	// If an empty RNode (e.g. null) is returned, the field will be cleared from the destination
	// If a nil RNode is returned, no action will be taken
	SetElement(source, dest *yaml.RNode) (*yaml.RNode, error)

	// SetList is called by walk on each non-associative list node.
	//
	// If a non-nil, non-empty RNode is returned it will be set on the destination.
	// If an empty RNode (e.g. null) is returned, the field will be cleared from the destination
	// If a nil RNode is returned, no action will be taken
	SetList(source, dest *yaml.RNode) (*yaml.RNode, error)

	// SetComments is called by walk on each source node, and may be used to copy comments
	// from the source to the destination.  Destination should be updated by the function.
	SetComments(source, dest *yaml.RNode) error
}

// NoOp is returned if GrepFilter should do nothing after calling Set
var NoOp *yaml.RNode = nil

// DefaultVisitor is a no-op visitor.
// It may be embedded anonymously to keep forwards compatibility when new functions are
// added to the interface.
type DefaultVisitor struct{}

func (DefaultVisitor) SetMapField(source, dest *yaml.MapNode) (*yaml.RNode, error) {
	return nil, nil
}

func (DefaultVisitor) SetScalarValue(source, dest *yaml.RNode) (*yaml.RNode, error) {
	return nil, nil
}

func (DefaultVisitor) SetElement(source, dest *yaml.RNode) (*yaml.RNode, error) {
	return nil, nil
}

func (DefaultVisitor) SetList(source, dest *yaml.RNode) (*yaml.RNode, error) {
	return nil, nil
}

func (DefaultVisitor) SetComments(source, dest *yaml.RNode) error {
	return nil
}

// GrepFilter walks the Source RNode and modifies the RNode provided to GrepFilter.
type Filter struct {
	// Visitor is invoked by GrepFilter
	Visitor

	// Source is the RNode to walk.  All Source fields and associative list elements
	// will be visited.
	Source *yaml.RNode

	// Path is the field path to the current Source Node.
	Path []string
}

// GrepFilter implements yaml.GrepFilter
func (l Filter) Filter(dest *yaml.RNode) (*yaml.RNode, error) {
	// invoke the handler for the corresponding node type
	switch dest.YNode().Kind {
	case yaml.MappingNode:
		if err := yaml.ErrorIfAnyInvalid(yaml.MappingNode, l.Source, dest); err != nil {
			return nil, err
		}
		return dest, l.walkMap(dest)
	case yaml.SequenceNode:
		if err := yaml.ErrorIfAnyInvalid(yaml.SequenceNode, l.Source, dest); err != nil {
			return nil, err
		}
		if l.Source.IsAssociative() {
			return dest, l.walkAssociativeSequence(dest)
		} else {
			return dest, l.walkNonAssociativeSequence(dest)
		}
	case yaml.ScalarNode:
		if err := yaml.ErrorIfAnyInvalid(yaml.ScalarNode, l.Source, dest); err != nil {
			return nil, err
		}
		return dest, l.walkScalar(dest)
	default:
		return dest, fmt.Errorf("unsupported Node Kind")
	}
}
