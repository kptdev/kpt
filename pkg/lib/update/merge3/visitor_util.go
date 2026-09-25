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

import kyaml "sigs.k8s.io/kustomize/kyaml/yaml"

func IsExplicitNull(n *kyaml.RNode) bool {
	return n.IsTaggedNull() && n.YNode().Value == "null"
}

func IsImplicitNull(n *kyaml.RNode) bool {
	return n.IsTaggedNull() && n.YNode().Value == ""
}

// IsCleared returns if the node has not tagged null in `left` but explicitly removed in `right`
func IsCleared(left, right *kyaml.RNode) bool {
	return !left.IsTaggedNull() && right.IsTaggedNull()
}

// PersistedNull returns dest's null value as an RNode that survives kyaml's
// generic tagged-null clearing (see yaml.MakePersistentNullNode).
func PersistedNull(dest *kyaml.RNode) *kyaml.RNode {
	value := ""
	if dest != nil {
		value = dest.YNode().Value
	}
	return kyaml.MakePersistentNullNode(value)
}
