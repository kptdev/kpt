// Copyright 2020 Google LLC
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

package merge

import (
	"path/filepath"
	"strings"

	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	mergeSourceAnnotation = "config.kubernetes.io/merge-source"
	mergeSourceOriginal   = "original"
	mergeSourceUpdated    = "updated"
	mergeSourceDest       = "dest"
)

// Merge3 performs a 3-way merge on the original, upstream and
// destination packages. It provides support for doing this only for
// the parent package and ignore any subpackages. Whenever the boundaries
// of a package differs between original, upstream and destination, the
// boundaries in destination will be used.
type Merge3 struct {
	OriginalPath       string
	UpdatedPath        string
	DestPath           string
	MatchFilesGlob     []string
	MergeOnPath        bool
	IncludeSubPackages bool
}

func (m Merge3) Merge() error {
	// If subpackages are not included when doing the merge, first
	// look up the known subpackages in destination so we can make sure
	// those are ignored when reading files from original and updated.
	var relPaths []string
	if !m.IncludeSubPackages {
		var err error
		relPaths, err = m.findExclusions()
		if err != nil {
			return err
		}
	}

	var inputs []kio.Reader
	dest := &kio.LocalPackageReadWriter{
		PackagePath:        m.DestPath,
		MatchFilesGlob:     m.MatchFilesGlob,
		SetAnnotations:     map[string]string{mergeSourceAnnotation: mergeSourceDest},
		IncludeSubpackages: m.IncludeSubPackages,
		PackageFileName:    kptfilev1alpha2.KptFileName,
	}
	inputs = append(inputs, dest)

	// Read the original package
	inputs = append(inputs, PruningLocalPackageReader{
		LocalPackageReader: kio.LocalPackageReader{
			PackagePath:        m.OriginalPath,
			MatchFilesGlob:     m.MatchFilesGlob,
			SetAnnotations:     map[string]string{mergeSourceAnnotation: mergeSourceOriginal},
			IncludeSubpackages: m.IncludeSubPackages,
			PackageFileName:    kptfilev1alpha2.KptFileName,
		},
		Exclusions: relPaths,
	})

	// Read the updated package
	inputs = append(inputs, PruningLocalPackageReader{
		LocalPackageReader: kio.LocalPackageReader{
			PackagePath:        m.UpdatedPath,
			MatchFilesGlob:     m.MatchFilesGlob,
			SetAnnotations:     map[string]string{mergeSourceAnnotation: mergeSourceUpdated},
			IncludeSubpackages: m.IncludeSubPackages,
			PackageFileName:    kptfilev1alpha2.KptFileName,
		},
		Exclusions: relPaths,
	})

	kyamlMerge := filters.Merge3{
		// TODO fix this.
                // MergeOnPath: m.MergeOnPath,
	}

	return kio.Pipeline{
		Inputs:  inputs,
		Filters: []kio.Filter{kyamlMerge},
		Outputs: []kio.Writer{dest},
	}.Execute()
}

func (m Merge3) findExclusions() ([]string, error) {
	var relPaths []string
	paths, err := pathutil.DirsWithFile(m.DestPath, kptfilev1alpha2.KptFileName, true)
	if err != nil {
		return relPaths, err
	}

	for _, p := range paths {
		rel, err := filepath.Rel(m.DestPath, p)
		if err != nil {
			return relPaths, err
		}
		if rel == "." {
			continue
		}
		relPaths = append(relPaths, rel)
	}
	return relPaths, nil
}

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
