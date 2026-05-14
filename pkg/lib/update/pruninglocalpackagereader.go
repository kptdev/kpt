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

package update

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// PruningLocalPackageReader implements the Reader interface. It is similar
// to the LocalPackageReader but allows for exclusion of subdirectories.
type PruningLocalPackageReader struct {
	LocalPackageReader kio.LocalPackageReader
	Exclusions         []string
}

func (p PruningLocalPackageReader) Read() ([]*yaml.RNode, error) {
	// Delegate reading the resources to the LocalPackageReader.
	nodes, err := p.LocalPackageReader.Read()
	if err != nil {
		return nil, err
	}

	// Exclude any resources that exist underneath an excluded path.
	var filteredNodes []*yaml.RNode
	for _, node := range nodes {
		if err := kioutil.CopyLegacyAnnotations(node); err != nil {
			return nil, err
		}
		n, err := node.Pipe(yaml.GetAnnotation(kioutil.PathAnnotation))
		if err != nil {
			return nil, err
		}
		path := n.YNode().Value
		if p.isExcluded(path) {
			continue
		}
		filteredNodes = append(filteredNodes, node)
	}
	return filteredNodes, nil
}

func (p PruningLocalPackageReader) isExcluded(path string) bool {
	for _, e := range p.Exclusions {
		if strings.HasPrefix(path, e) {
			return true
		}
	}
	return false
}
