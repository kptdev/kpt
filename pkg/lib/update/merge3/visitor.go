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
	"sigs.k8s.io/kustomize/kyaml/yaml/merge3"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

type Visitor struct {
	*merge3.Visitor
}

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

	values, err := m.getStrValues(nodes)
	if err != nil {
		return nil, err
	}

	if (values.Dest == "" || values.Dest == values.Origin) && values.Origin != values.Update {
		// if local is nil or is unchanged but there is new update
		return nodes.Updated(), nil
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

	// compare origin and update values to see if they have changed
	values, err := m.getStrValues(nodes)
	if err != nil {
		return nil, err
	}
	if values.Update != values.Origin {
		// value changed in update
		return nodes.Updated(), nil
	}

	// unchanged between origin and update, keep the dest
	return nodes.Dest(), nil
}

// NEW
// isCleared returns if the node has been present in `left` but explicitly removed in `right`
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
func (m *Visitor) getStrValues(nodes walk.Sources) (strValues, error) {
	var uStr, oStr, dStr string
	var err error

	for _, p := range []struct {
		rnode *yaml.RNode
		str   *string
	}{
		{nodes.Updated(), &uStr},
		{nodes.Origin(), &oStr},
		{nodes.Dest(), &dStr},
	} {
		rnode := p.rnode
		if rnode == nil || rnode.YNode() == nil {
			continue
		}
		s := rnode.YNode().Style
		defer func() {
			rnode.YNode().Style = s
		}()
		rnode.YNode().Style = yaml.FlowStyle | yaml.SingleQuotedStyle
		*p.str, err = rnode.String()
		if err != nil {
			return strValues{}, err
		}
	}

	return strValues{Origin: oStr, Update: uStr, Dest: dStr}, nil
}

type strValues struct {
	Origin string
	Update string
	Dest   string
}

var _ walk.Visitor = &Visitor{}
