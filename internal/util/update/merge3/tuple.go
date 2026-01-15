// Copyright 2025 The kpt Authors
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
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge3"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

// tuple wraps an original, updated, and dest tuple for a given Resource
type tuple struct {
	original,
	updated,
	dest *yaml.RNode
}

// merge performs a 3-way merge on the tuple
func (t *tuple) merge() (*yaml.RNode, error) {
	return walk.Walker{
		// same as in merge3.Merge()
		Visitor:            merge3.Visitor{},
		VisitKeysAsScalars: true,
		Sources:            []*yaml.RNode{t.dest, t.original, t.updated},

		// added
		InferAssociativeLists: false,
	}.Walk()
}

type tuplelist []*tuple

// tuples combines nodes with the same GVK + N + NS
type tuples struct {
	tuplelist

	matcher filters.ResourceMatcher
}

// addOriginal adds an original node to the list, returning an error if such a Resource had already been added
func (ts *tuples) addOriginal(node *yaml.RNode) error {
	for i := range ts.tuplelist {
		t := ts.tuplelist[i]
		if ts.matcher.IsSameResource(addedNode(t), node) {
			if t.original != nil {
				return duplicateError("original", node.GetAnnotations()[kioutil.PathAnnotation])
			}
			t.original = node
			return nil
		}
	}
	ts.tuplelist = append(ts.tuplelist, &tuple{original: node})
	return nil
}

// addUpdated adds an updated node to the list, combining it with an existing matching Resource if found
func (ts *tuples) addUpdated(node *yaml.RNode) error {
	for i := range ts.tuplelist {
		t := ts.tuplelist[i]
		if ts.matcher.IsSameResource(addedNode(t), node) {
			if t.updated != nil {
				return duplicateError("updated", node.GetAnnotations()[kioutil.PathAnnotation])
			}
			t.updated = node
			return nil
		}
	}
	ts.tuplelist = append(ts.tuplelist, &tuple{updated: node})
	return nil
}

// addDest adds a dest node to the list, combining it with an existing matching Resource if found
func (ts *tuples) addDest(node *yaml.RNode) error {
	for i := range ts.tuplelist {
		t := ts.tuplelist[i]
		if ts.matcher.IsSameResource(addedNode(t), node) {
			if t.dest != nil {
				return duplicateError("dest", node.GetAnnotations()[kioutil.PathAnnotation])
			}
			t.dest = node
			return nil
		}
	}
	ts.tuplelist = append(ts.tuplelist, &tuple{dest: node})
	return nil
}

// addedNode returns one on the existing added nodes in the tuple
func addedNode(t *tuple) *yaml.RNode {
	if t.original != nil {
		return t.original
	}
	if t.updated != nil {
		return t.updated
	}
	return t.dest
}

// duplicateError returns duplicate resources error
func duplicateError(source, filePath string) error {
	return fmt.Errorf(`found duplicate %q resources in file %q, please refer to "update" documentation for the fix`, source, filePath)
}
