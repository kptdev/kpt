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

package filters

import (
	"fmt"

	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
	"lib.kpt.dev/yaml/merge3"
)

const (
	mergeSourceAnnotation = "kpt.dev/merge-source"
	mergeSourceOriginal   = "original"
	mergeSourceUpdated    = "updated"
	mergeSourceDest       = "dest"
)

// Merge3 performs a 3-way merge on the original, updated, and destination packages.
type Merge3 struct {
	OriginalPath   string
	UpdatedPath    string
	DestPath       string
	MatchFilesGlob []string
}

func (m Merge3) Merge() error {
	// Read the destination package.  The ReadWriter will take take of deleting files
	// for removed resources.
	var inputs []kio.Reader
	dest := &kio.LocalPackageReadWriter{
		PackagePath:    m.DestPath,
		MatchFilesGlob: m.MatchFilesGlob,
		SetAnnotations: map[string]string{mergeSourceAnnotation: mergeSourceDest},
	}
	inputs = append(inputs, dest)

	// Read the original package
	inputs = append(inputs, kio.LocalPackageReader{
		PackagePath:    m.OriginalPath,
		MatchFilesGlob: m.MatchFilesGlob,
		SetAnnotations: map[string]string{mergeSourceAnnotation: mergeSourceOriginal},
	})

	// Read the updated package
	inputs = append(inputs, kio.LocalPackageReader{
		PackagePath:    m.UpdatedPath,
		MatchFilesGlob: m.MatchFilesGlob,
		SetAnnotations: map[string]string{mergeSourceAnnotation: mergeSourceUpdated},
	})

	return kio.Pipeline{
		Inputs:  inputs,
		Filters: []kio.Filter{m, FormatFilter{}}, // format the merged output
		Outputs: []kio.Writer{dest},
	}.Execute()
}

// Filter combines Resources with the same GVK + N + NS into tuples, and then merges them
func (m Merge3) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	// index the nodes by their identity
	tl := tuples{}
	for i := range nodes {
		if err := tl.add(nodes[i]); err != nil {
			return nil, err
		}
	}

	// iterate over the inputs, merging as needed
	var output []*yaml.RNode
	for i := range tl.list {
		t := tl.list[i]
		switch {
		case t.original == nil && t.updated == nil && t.dest != nil:
			// added locally -- keep dest
			output = append(output, t.dest)
		case t.original == nil && t.updated != nil && t.dest == nil:
			// added in the update -- add update
			output = append(output, t.updated)
		case t.original != nil && t.updated == nil:
			// deleted in the update
		// don't include the resource in the output
		case t.original != nil && t.dest == nil:
			// deleted locally
			// don't include the resource in the output
		default:
			// dest and updated are non-nil -- merge them
			node, err := t.merge()
			if err != nil {
				return nil, err
			}
			if node != nil {
				output = append(output, node)
			}
		}
	}
	return output, nil
}

// tuples combines nodes with the same GVK + N + NS
type tuples struct {
	list []*tuple
}

// add adds a node to the list, combining it with an existing matching Resource if found
func (ts *tuples) add(node *yaml.RNode) error {
	nodeMeta, err := node.GetMeta()
	if err != nil {
		return err
	}
	for i := range ts.list {
		t := ts.list[i]
		if t.meta.Name == nodeMeta.Name && t.meta.Namespace == nodeMeta.Namespace &&
			t.meta.ApiVersion == nodeMeta.ApiVersion && t.meta.Kind == nodeMeta.Kind {
			return t.add(node)
		}
	}
	t := &tuple{meta: nodeMeta}
	if err := t.add(node); err != nil {
		return err
	}
	ts.list = append(ts.list, t)
	return nil
}

// tuple wraps an original, updated, and dest tuple for a given Resource
type tuple struct {
	meta     yaml.ResourceMeta
	original *yaml.RNode
	updated  *yaml.RNode
	dest     *yaml.RNode
}

// add sets the corresponding tuple field for the node
func (t *tuple) add(node *yaml.RNode) error {
	meta, err := node.GetMeta()
	if err != nil {
		return err
	}
	switch meta.Annotations[mergeSourceAnnotation] {
	case mergeSourceDest:
		if t.dest != nil {
			return fmt.Errorf("dest source already specified")
		}
		t.dest = node
	case mergeSourceOriginal:
		if t.original != nil {
			return fmt.Errorf("original source already specified")
		}
		t.original = node
	case mergeSourceUpdated:
		if t.updated != nil {
			return fmt.Errorf("updated source already specified")
		}
		t.updated = node
	default:
		return fmt.Errorf("no source annotation for Resource")
	}
	return nil
}

// merge performs a 3-way merge on the tuple
func (t *tuple) merge() (*yaml.RNode, error) {
	return merge3.Merge(t.dest, t.original, t.updated)
}
