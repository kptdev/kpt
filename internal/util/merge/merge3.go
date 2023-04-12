// Copyright 2020 The kpt Authors
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

	"github.com/GoogleContainerTools/kpt/internal/util/attribution"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	mergeSourceAnnotation = "config.kubernetes.io/merge-source"
	mergeSourceOriginal   = "original"
	mergeSourceUpdated    = "updated"
	mergeSourceDest       = "dest"
	MergeCommentPrefix    = "kpt-merge:"
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
		PackageFileName:    kptfilev1.KptFileName,
		PreserveSeqIndent:  true,
		WrapBareSeqNode:    true,
	}
	inputs = append(inputs, dest)

	// Read the original package
	inputs = append(inputs, PruningLocalPackageReader{
		LocalPackageReader: kio.LocalPackageReader{
			PackagePath:        m.OriginalPath,
			MatchFilesGlob:     m.MatchFilesGlob,
			SetAnnotations:     map[string]string{mergeSourceAnnotation: mergeSourceOriginal},
			IncludeSubpackages: m.IncludeSubPackages,
			PackageFileName:    kptfilev1.KptFileName,
			PreserveSeqIndent:  true,
			WrapBareSeqNode:    true,
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
			PackageFileName:    kptfilev1.KptFileName,
			PreserveSeqIndent:  true,
			WrapBareSeqNode:    true,
		},
		Exclusions: relPaths,
	})

	rmMatcher := ResourceMergeMatcher{MergeOnPath: m.MergeOnPath}
	resourceHandler := resourceHandler{}
	kyamlMerge := filters.Merge3{
		Matcher: &rmMatcher,
		Handler: &resourceHandler,
	}

	return kio.Pipeline{
		Inputs:  inputs,
		Filters: []kio.Filter{kyamlMerge},
		Outputs: []kio.Writer{dest},
	}.Execute()
}

func (m Merge3) findExclusions() ([]string, error) {
	var relPaths []string
	paths, err := pathutil.DirsWithFile(m.DestPath, kptfilev1.KptFileName, true)
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

type ResourceMergeMatcher struct {
	MergeOnPath bool
}

// IsSameResource determines if 2 resources are same to be merged by matching GKNN+filepath
// Group, Kind are derived from resource metadata directly, Namespace and Name are derived
// from merge comment which is of format "kpt-merge: namespace/name", if the merge comment
// is not present, then it falls back to Namespace and Name on the resource meta
func (rm *ResourceMergeMatcher) IsSameResource(node1, node2 *yaml.RNode) bool {
	if node1 == nil || node2 == nil {
		return false
	}

	if err := kioutil.CopyLegacyAnnotations(node1); err != nil {
		return false
	}
	if err := kioutil.CopyLegacyAnnotations(node2); err != nil {
		return false
	}

	meta1, err := node1.GetMeta()
	if err != nil {
		return false
	}

	meta2, err := node2.GetMeta()
	if err != nil {
		return false
	}

	if resolveGroup(meta1) != resolveGroup(meta2) {
		return false
	}

	if meta1.Kind != meta2.Kind {
		return false
	}

	if resolveName(meta1, metadataComment(node1)) != resolveName(meta2, metadataComment(node2)) {
		return false
	}

	if resolveNamespace(meta1, metadataComment(node1)) != resolveNamespace(meta2, metadataComment(node2)) {
		return false
	}

	if rm.MergeOnPath {
		// directories may contain multiple copies of a resource with the same
		// name, namespace, apiVersion and kind -- e.g. kustomize patches, or
		// multiple environments
		// mergeOnPath configures the merge logic to use the path as part of the
		// resource key
		if meta1.Annotations[kioutil.PathAnnotation] != meta2.Annotations[kioutil.PathAnnotation] {
			return false
		}
	}
	return true
}

// resolveGroup resolves the group of a resource from ResourceMeta
func resolveGroup(meta yaml.ResourceMeta) string {
	group, _ := resid.ParseGroupVersion(meta.APIVersion)
	return group
}

// resolveNamespace resolves the namespace which should be used for merging resources
// uses namespace from comment on metadata field if present, falls back to resource namespace
func resolveNamespace(meta yaml.ResourceMeta, metadataComment string) string {
	nsName := NsAndNameForMerge(metadataComment)
	if nsName == nil {
		return meta.Namespace
	}
	return nsName[0]
}

// resolveName resolves the name which should be used for merging resources
// uses name from comment on metadata field if present, falls back to resource name
func resolveName(meta yaml.ResourceMeta, metadataComment string) string {
	nsName := NsAndNameForMerge(metadataComment)
	if nsName == nil {
		return meta.Name
	}
	return nsName[1]
}

// NsAndNameForMerge returns the namespace and name for merge
// from the line comment on the metadata field
// e.g. metadata: # kpt-merge: default/foo returns [default, foo]
func NsAndNameForMerge(metadataComment string) []string {
	comment := strings.TrimPrefix(metadataComment, "#")
	comment = strings.TrimSpace(comment)
	if !strings.HasPrefix(comment, MergeCommentPrefix) {
		return nil
	}
	comment = strings.TrimPrefix(comment, MergeCommentPrefix)
	nsAndName := strings.SplitN(strings.TrimSpace(comment), "/", 2)
	if len(nsAndName) != 2 {
		return nil
	}
	return nsAndName
}

// metadataComment returns the line comment on the metadata field of input RNode
func metadataComment(node *yaml.RNode) string {
	mf := node.Field(yaml.MetadataField)
	if mf.IsNilOrEmpty() {
		return ""
	}
	return mf.Key.YNode().LineComment
}

// resourceHandler is an implementation of the ResourceHandler interface from
// kyaml. It is used to decide how a resource should be handled during the
// 3-way merge. This differs from the default implementation in that if a
// resource is deleted from upstream, it will only be deleted from local if
// there is no diff between origin and local.
type resourceHandler struct {
	keptResources []*yaml.RNode
}

func (r *resourceHandler) Handle(origin, upstream, local *yaml.RNode) (filters.ResourceMergeStrategy, error) {
	var strategy filters.ResourceMergeStrategy
	switch {
	// Keep the resource if added locally.
	case origin == nil && upstream == nil && local != nil:
		strategy = filters.KeepDest
	// Add the resource if added in upstream.
	case origin == nil && upstream != nil && local == nil:
		strategy = filters.KeepUpdated
	// Do not re-add the resource if deleted from both upstream and local
	case upstream == nil && local == nil:
		strategy = filters.Skip
	// If deleted from upstream, only delete if local fork does not have changes.
	case origin != nil && upstream == nil:
		equal, err := r.equals(origin, local)
		if err != nil {
			return strategy, err
		}
		if equal {
			strategy = filters.Skip
		} else {
			r.keptResources = append(r.keptResources, local)
			strategy = filters.KeepDest
		}
	// Do not re-add if deleted from local.
	case origin != nil && local == nil:
		strategy = filters.Skip
	default:
		strategy = filters.Merge
	}
	return strategy, nil
}

func (*resourceHandler) equals(r1, r2 *yaml.RNode) (bool, error) {
	// We need to create new copies of the resources since we need to
	// mutate them before comparing them.
	r1Clone, err := yaml.Parse(r1.MustString())
	if err != nil {
		return false, err
	}
	r2Clone, err := yaml.Parse(r2.MustString())
	if err != nil {
		return false, err
	}

	// The resources include annotations with information used during the merge
	// process. We need to remove those before comparing the resources.
	if err := stripKyamlAnnos(r1Clone); err != nil {
		return false, err
	}
	if err := stripKyamlAnnos(r2Clone); err != nil {
		return false, err
	}

	return r1Clone.MustString() == r2Clone.MustString(), nil
}

func stripKyamlAnnos(n *yaml.RNode) error {
	for _, a := range []string{mergeSourceAnnotation, kioutil.PathAnnotation, kioutil.IndexAnnotation,
		kioutil.LegacyPathAnnotation, kioutil.LegacyIndexAnnotation, // nolint:staticcheck
		kioutil.InternalAnnotationsMigrationResourceIDAnnotation, attribution.CNRMMetricsAnnotation} {
		err := n.PipeE(yaml.ClearAnnotation(a))
		if err != nil {
			return err
		}
	}
	return nil
}
