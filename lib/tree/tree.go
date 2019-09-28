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

package tree

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xlab/treeprint"
	"lib.kpt.dev/kio/kioutil"
	"lib.kpt.dev/yaml"
)

type Printer struct {
	Writer io.Writer
	Root   string
}

func (p Printer) Write(nodes []*yaml.RNode) error {
	nodeIndex := map[string][]*yaml.RNode{}

	// index the ResourceNodes by package
	for i := range nodes {
		meta, err := nodes[i].GetMeta()
		if err != nil || meta.Kind == "" {
			// not a resource
			continue
		}
		pkg := meta.Annotations[kioutil.PackageAnnotation]
		nodeIndex[pkg] = append(nodeIndex[pkg], nodes[i])
	}

	// sort the ResourceNodes, and get the sorted package names
	var keys []string
	for k := range nodeIndex {
		pkgNodes := nodeIndex[k]
		sort.Slice(pkgNodes, func(i, j int) bool {
			// should have already fetched meta for each node
			metai, _ := pkgNodes[i].GetMeta()
			metaj, _ := pkgNodes[j].GetMeta()
			pi := metai.Annotations[kioutil.PathAnnotation]
			pj := metaj.Annotations[kioutil.PathAnnotation]
			if filepath.Base(pi) != filepath.Base(pj) {
				return filepath.Base(pi) < filepath.Base(pj)
			}

			if metai.Namespace != metaj.Namespace {
				return metai.Namespace < metaj.Namespace
			}
			if metai.Name != metaj.Name {
				return metai.Name < metaj.Name
			}
			if metai.Kind != metaj.Kind {
				return metai.Kind < metaj.Kind
			}
			if metai.ApiVersion != metaj.ApiVersion {
				return metai.ApiVersion < metaj.ApiVersion
			}
			return true
		})
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// construct the tree
	tree := treeprint.New()
	tree.SetValue(p.Root)
	treeIndex := map[string]treeprint.Tree{}
	for _, pkg := range keys {
		match := tree
		for parent, tree := range treeIndex {
			if strings.HasPrefix(pkg, parent) {
				// put the package under this one
				match = tree
				// don't break, continue searching for more closely related ancestors
			}
		}

		// Add the branch and its leaves
		var branch treeprint.Tree
		if pkg != "." {
			branch = match.AddBranch(pkg)
		} else {
			branch = match
		}
		treeIndex[pkg] = branch
		leaves := nodeIndex[pkg]
		for i := range leaves {
			leaf := leaves[i]
			meta, _ := leaf.GetMeta()
			path := meta.Annotations[kioutil.PathAnnotation]
			path = filepath.Base(path)
			value := fmt.Sprintf("%s.%s %s", meta.ApiVersion, meta.Kind, meta.Name)
			if len(meta.Namespace) > 0 {
				value = fmt.Sprintf("%s.%s %s/%s", meta.ApiVersion, meta.Kind, meta.Namespace,
					meta.Name)
			}
			branch.AddMetaNode(path, value)
		}
	}

	_, err := io.WriteString(p.Writer, tree.String())
	return err
}
