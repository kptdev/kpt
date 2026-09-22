// Copyright 2022 The kpt Authors
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

package internal

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type SliceVariant struct {
	node *yaml.Node
}

func NewSliceVariant(s ...variant) *SliceVariant {
	node := buildSequenceNode()
	for _, v := range s {
		node.Content = append(node.Content, v.Node())
	}
	return &SliceVariant{node: node}
}

func (v *SliceVariant) GetKind() VariantKind {
	return VariantKindSlice
}

func (v *SliceVariant) Node() *yaml.Node {
	return v.node
}

func (v *SliceVariant) Clear() {
	v.node.Content = nil
}

func (v *SliceVariant) Elements() ([]*MapVariant, error) {
	return ExtractObjects(v.node.Content...)
}

func (v *SliceVariant) Add(node variant) {
	v.node.Content = append(v.node.Content, node.Node())
}
