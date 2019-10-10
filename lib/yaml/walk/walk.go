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
	"os"
	"strings"

	"lib.kpt.dev/yaml"
)

// Filter walks the Source RNode and modifies the RNode provided to GrepFilter.
type Walker struct {
	// Visitor is invoked by GrepFilter
	Visitor

	// Source is the RNode to walk.  All Source fields and associative list elements
	// will be visited.
	Sources Sources

	// Path is the field path to the current Source Node.
	Path []string
}

func (l Walker) Kind() yaml.Kind {
	for _, s := range l.Sources {
		if !yaml.IsEmpty(s) {
			return s.YNode().Kind
		}
	}
	return 0
}

// GrepFilter implements yaml.GrepFilter
func (l Walker) Walk() (*yaml.RNode, error) {
	// invoke the handler for the corresponding node type
	switch l.Kind() {
	case yaml.MappingNode:
		if err := yaml.ErrorIfAnyInvalidAndNonNull(yaml.MappingNode, l.Sources...); err != nil {
			return nil, err
		}
		return l.walkMap()
	case yaml.SequenceNode:
		if err := yaml.ErrorIfAnyInvalidAndNonNull(yaml.SequenceNode, l.Sources...); err != nil {
			return nil, err
		}
		if yaml.IsAssociative(l.Sources) {
			return l.walkAssociativeSequence()
		} else {
			return l.walkNonAssociativeSequence()
		}
	case yaml.ScalarNode:
		if err := yaml.ErrorIfAnyInvalidAndNonNull(yaml.ScalarNode, l.Sources...); err != nil {
			return nil, err
		}
		return l.walkScalar()
	default:
		return nil, nil
	}
}

const (
	DestIndex = iota
	OriginIndex
	UpdatedIndex
)

type Sources []*yaml.RNode

// Dest returns the destination node
func (s Sources) Dest() *yaml.RNode {
	if len(s) <= DestIndex {
		return nil
	}
	return s[DestIndex]
}

// Origin returns the origin node
func (s Sources) Origin() *yaml.RNode {
	if len(s) <= OriginIndex {
		return nil
	}
	return s[OriginIndex]
}

// Updated returns the updated node
func (s Sources) Updated() *yaml.RNode {
	if len(s) <= UpdatedIndex {
		return nil
	}
	return s[UpdatedIndex]
}

func (s Sources) String() string {
	var values []string
	for i := range s {
		str, err := s[i].String()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		values = append(values, str)
	}
	return strings.Join(values, "\n")
}

// setDestNode sets the destination source node
func (s Sources) setDestNode(node *yaml.RNode, err error) (*yaml.RNode, error) {
	if err != nil {
		return nil, err
	}
	s[0] = node
	return node, nil
}

type FieldSources []*yaml.MapNode

// Dest returns the destination node
func (s FieldSources) Dest() *yaml.MapNode {
	if len(s) <= DestIndex {
		return nil
	}
	return s[DestIndex]
}

// Origin returns the origin node
func (s FieldSources) Origin() *yaml.MapNode {
	if len(s) <= OriginIndex {
		return nil
	}
	return s[OriginIndex]
}

// Updated returns the updated node
func (s FieldSources) Updated() *yaml.MapNode {
	if len(s) <= UpdatedIndex {
		return nil
	}
	return s[UpdatedIndex]
}
