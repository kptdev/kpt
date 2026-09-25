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

package merge3

import (
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

// NullPreservingVisitor is a copy of our Visitor that preserves nulls.
// This behavior is necessary when the merged packages themselves are patches.
type NullPreservingVisitor struct{}

var _ walk.Visitor = &NullPreservingVisitor{}

func (m *NullPreservingVisitor) VisitMap(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	if IsCleared(nodes.Origin(), nodes.Updated()) {
		// upstream set this field to null; keep that null
		return PersistedNull(nodes.Updated()), nil
	}
	if IsCleared(nodes.Origin(), nodes.Dest()) {
		return PersistedNull(nodes.Dest()), nil
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
	// Tagged nulls are IsMissingOrNull, so a field that is null on every side
	// is walked as an empty map. Keep dest's null instead of dropping it.
	// A concrete updated value still replaces a null origin.
	if nodes.Dest().IsTaggedNull() {
		if yaml.IsMissingOrNull(nodes.Updated()) {
			return PersistedNull(nodes.Dest()), nil
		}
		return yaml.NewRNode(&yaml.Node{Kind: yaml.MappingNode}), nil
	}

	// recursively merge the dest with the original and updated
	return nodes.Dest(), nil
}

func (m *NullPreservingVisitor) visitAList(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	if IsCleared(nodes.Origin(), nodes.Updated()) {
		// upstream set this list to null; keep that null
		return PersistedNull(nodes.Updated()), nil
	}
	if IsCleared(nodes.Origin(), nodes.Dest()) {
		return PersistedNull(nodes.Dest()), nil
	}
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

func (m *NullPreservingVisitor) VisitScalar(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	if IsCleared(nodes.Origin(), nodes.Updated()) {
		// upstream set this field to null; keep that null
		return PersistedNull(nodes.Updated()), nil
	}
	if IsCleared(nodes.Origin(), nodes.Dest()) {
		return PersistedNull(nodes.Dest()), nil
	}
	if yaml.IsMissingOrNull(nodes.Updated()) != yaml.IsMissingOrNull(nodes.Origin()) {
		// value added or removed in update
		return KeepNull(nodes.Updated()), nil
	}
	if yaml.IsMissingOrNull(nodes.Updated()) && yaml.IsMissingOrNull(nodes.Origin()) {
		// value absent in both origin and update
		return KeepNull(nodes.Dest()), nil
	}

	if nodes.Updated().YNode().Value != nodes.Origin().YNode().Value {
		// value changed in update
		return nodes.Updated(), nil
	}

	// unchanged between origin and update, keep the dest
	return KeepNull(nodes.Dest()), nil
}

func (m *NullPreservingVisitor) visitNAList(nodes walk.Sources) (*yaml.RNode, error) {
	if IsCleared(nodes.Origin(), nodes.Updated()) {
		// upstream set this list to null; keep that null
		return PersistedNull(nodes.Updated()), nil
	}
	if IsCleared(nodes.Origin(), nodes.Dest()) {
		return PersistedNull(nodes.Dest()), nil
	}

	if yaml.IsMissingOrNull(nodes.Updated()) != yaml.IsMissingOrNull(nodes.Origin()) {
		// value added or removed in update
		return KeepNull(nodes.Updated()), nil
	}
	if yaml.IsMissingOrNull(nodes.Updated()) && yaml.IsMissingOrNull(nodes.Origin()) {
		// value not present in source or dest
		return KeepNull(nodes.Dest()), nil
	}

	if !m.isNodeContentEqual(nodes.Origin().YNode(), nodes.Updated().YNode()) {
		// value changed in update
		return nodes.Updated(), nil
	}

	// unchanged between origin and update, keep the dest
	return KeepNull(nodes.Dest()), nil
}

func (m *NullPreservingVisitor) VisitList(nodes walk.Sources, s *openapi.ResourceSchema, kind walk.ListKind) (*yaml.RNode, error) {
	if kind == walk.AssociativeList {
		return m.visitAList(nodes, s)
	}
	// non-associative list
	return m.visitNAList(nodes)
}

// SIMPLIFIED
// isNodeContentEqual compares the nodes structurally (kind, tag, value and children),
// avoiding YAML serialization. Presentation style and comments are ignored.
func (m *NullPreservingVisitor) isNodeContentEqual(a, b *yaml.Node) bool {
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
