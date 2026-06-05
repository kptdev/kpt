// Copyright 2019 The Kubernetes Authors.
// Copyright 2026 The kpt Authors
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

// copied from sigs.k8s.io/kustomize/kyaml@v0.21.1/yaml/merge3/visitor.go with modifications

package merge3

import (
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

type Visitor struct{}

func (m *Visitor) VisitMap(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	if m.isCleared(nodes.Origin(), nodes.Updated()) || m.isCleared(nodes.Origin(), nodes.Dest()) { // MODIFIED
		// explicitly cleared from either dest or update
		return walk.ClearNode, nil
	}
	if nodes.Dest() == nil && nodes.Updated() == nil {
		// implicitly cleared missing from both dest and update
		return walk.ClearNode, nil
	}

	if nodes.Dest() == nil {
		// not cleared, but missing from the dest
		// initialize a new value that can be recursively merged
		return yaml.NewRNode(&yaml.Node{Kind: yaml.MappingNode}), nil
	}

	// recursively merge the dest with the original and updated
	return nodes.Dest(), nil
}

func (m *Visitor) visitAList(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	if yaml.IsMissingOrNull(nodes.Updated()) && !yaml.IsMissingOrNull(nodes.Origin()) {
		// implicitly cleared from update -- element was deleted
		return walk.ClearNode, nil
	}
	if yaml.IsMissingOrNull(nodes.Dest()) {
		// not cleared, but missing from the dest
		// initialize a new value that can be recursively merged
		return yaml.NewRNode(&yaml.Node{Kind: yaml.SequenceNode}), nil
	}

	// recursively merge the dest with the original and updated
	return nodes.Dest(), nil
}

func (m *Visitor) VisitScalar(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	if m.isCleared(nodes.Origin(), nodes.Updated()) || m.isCleared(nodes.Origin(), nodes.Dest()) { // MODIFIED
		// explicitly cleared from either dest or update
		return nil, nil
	}
	if yaml.IsMissingOrNull(nodes.Updated()) != yaml.IsMissingOrNull(nodes.Origin()) {
		// value added or removed in update
		return nodes.Updated(), nil
	}
	if yaml.IsMissingOrNull(nodes.Updated()) && yaml.IsMissingOrNull(nodes.Origin()) {
		// value absent in both origin and update
		return nodes.Dest(), nil
	}

	if nodes.Updated().YNode().Value != nodes.Origin().YNode().Value {
		// value changed in update
		return nodes.Updated(), nil
	}

	// unchanged between origin and update, keep the dest
	return nodes.Dest(), nil
}

func (m *Visitor) visitNAList(nodes walk.Sources) (*yaml.RNode, error) {
	if m.isCleared(nodes.Origin(), nodes.Updated()) || m.isCleared(nodes.Origin(), nodes.Dest()) { // MODIFIED
		// explicitly cleared from either dest or update
		return walk.ClearNode, nil
	}

	if yaml.IsMissingOrNull(nodes.Updated()) != yaml.IsMissingOrNull(nodes.Origin()) {
		// value added or removed in update
		return nodes.Updated(), nil
	}
	if yaml.IsMissingOrNull(nodes.Updated()) && yaml.IsMissingOrNull(nodes.Origin()) {
		// value not present in source or dest
		return nodes.Dest(), nil
	}

	if !m.isNodeContentEqual(nodes.Origin().YNode(), nodes.Updated().YNode()) {
		// value changed in update
		return nodes.Updated(), nil
	}

	// unchanged between origin and update, keep the dest
	return nodes.Dest(), nil
}

// NEW
// isCleared returns if the node has not tagged null in `left` but explicitly removed in `right`
func (*Visitor) isCleared(left, right *yaml.RNode) bool {
	return !left.IsTaggedNull() && right.IsTaggedNull()
}

func (m *Visitor) VisitList(nodes walk.Sources, s *openapi.ResourceSchema, kind walk.ListKind) (*yaml.RNode, error) {
	if kind == walk.AssociativeList {
		return m.visitAList(nodes, s)
	}
	// non-associative list
	return m.visitNAList(nodes)
}

// SIMPLIFIED
// isNodeContentEqual compares the nodes structurally (kind, tag, value and children),
// avoiding YAML serialization. Presentation style and comments are ignored.
func (m *Visitor) isNodeContentEqual(a, b *yaml.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value {
		return false
	}
	if len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !m.isNodeContentEqual(a.Content[i], b.Content[i]) {
			return false
		}
	}
	return true
}

var _ walk.Visitor = &Visitor{}
