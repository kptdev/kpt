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

// Package merge contains libraries for merging fields from one RNode to another
// RNode
package merge

import (
	"lib.kpt.dev/yaml"
	"lib.kpt.dev/yaml/walk"
)

func Merge(src, dest *yaml.RNode) (*yaml.RNode, error) {
	return dest.Pipe(walk.Filter{Source: src, Visitor: Merger{}})
}

func MergeStrings(srcStr, destStr string) (string, error) {
	src, err := yaml.Parse(srcStr)
	if err != nil {
		return "", err
	}
	dest, err := yaml.Parse(destStr)
	if err != nil {
		return "", err
	}

	result, err := dest.Pipe(walk.Filter{Source: src, Visitor: Merger{}})
	if err != nil {
		return "", err
	}
	return result.String()
}

// Merger implements walk.Visitor and merges fields from a source into a destination.
//
// - Fields specified in both the source and destination are merged
// - Fields specified only in the destination are kept in the destination
// - Fields specified only in the source are copied to the destination
// - Fields may be cleared from the destination by setting the value to null in the source
//
// - List elements are merged using an associative key.  If an associative key is found
//   in the the list element fields.
// - Lists containing elements without an associative key are replaced in the
//   destination by the source list.
//
// - Comments are merged from the source if they are present, otherwise the source comments
//   are kept.
type Merger struct {
	// for forwards compatibility when new functions are added to the interface
	walk.DefaultVisitor
}

var _ walk.Visitor = Merger{}

// SetMapField copies the dest field to the source field
func (m Merger) SetMapField(source, dest *yaml.MapNode) (*yaml.RNode, error) {
	if yaml.IsFieldEmpty(dest) {
		return source.Value, nil
	}
	return m.copyRNode(source.Value, dest.Value)
}

// SetScalarValue copies the dest value to source value
func (m Merger) SetScalarValue(source, dest *yaml.RNode) (*yaml.RNode, error) {
	return m.copyRNode(source, dest)
}

// SetElement copies the dest element to the source element
func (m Merger) SetElement(source, dest *yaml.RNode) (*yaml.RNode, error) {
	n, e := m.copyRNode(source, dest)
	return n, e
}

// SetList copies the dest list to the source list, replacing all elements
func (m Merger) SetList(source, dest *yaml.RNode) (*yaml.RNode, error) {
	return m.copyRNode(source, dest)
}

// SetComments copies the dest comments to the source comments if they are present
// on the source.
func (m Merger) SetComments(source, dest *yaml.RNode) error {
	if source.YNode().FootComment != "" {
		dest.YNode().FootComment = source.YNode().FootComment
	}
	if source.YNode().HeadComment != "" {
		dest.YNode().HeadComment = source.YNode().HeadComment
	}
	if source.YNode().LineComment != "" {
		dest.YNode().LineComment = source.YNode().LineComment
	}
	return nil
}

func (m Merger) copyRNode(source, dest *yaml.RNode) (*yaml.RNode, error) {
	// if the field doesn't exist in the dest, return the source value directly
	if yaml.IsEmpty(dest) {
		return source, nil
	}

	// not present in the source one way or another
	if source == nil {
		return dest, nil
	}

	// if the field exists in the dest, copy the source values to the dest node
	m.copy(source.YNode(), dest.YNode())
	if err := m.SetComments(source, dest); err != nil {
		return nil, err
	}

	return walk.NoOp, nil
}

func (m Merger) copy(source, dest *yaml.Node) {
	dest.Value = source.Value
	dest.Content = source.Content
	dest.Kind = source.Kind
	dest.Tag = source.Tag
}
