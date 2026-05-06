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
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	MergeCommentPrefix = "kpt-merge:"
)

var _ filters.ResourceMatcher = &resourceMergeMatcher{}

// resourceMergeMatcher differs from the default matcher in that Namespace and Name are derived
// from merge comment, which is of format "kpt-merge: namespace/name",
// if the merge comment is not present, then it falls back to Namespace and Name in the resource meta.
type resourceMergeMatcher struct{}

// IsSameResource determines if 2 resources are same to be merged by matching GKNN+filepath
// Group, Kind are derived from resource metadata directly, Namespace and Name are derived
// from merge comment which is of format "kpt-merge: namespace/name", if the merge comment
// is not present, then it falls back to Namespace and Name on the resource meta
func (rm *resourceMergeMatcher) IsSameResource(node1, node2 *yaml.RNode) bool {
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

	// directories may contain multiple copies of a resource with the same
	// name, namespace, apiVersion and kind -- e.g. kustomize patches, or
	// multiple environments
	// mergeOnPath configures the merge logic to use the path as part of the
	// resource key
	if meta1.Annotations[kioutil.PathAnnotation] != meta2.Annotations[kioutil.PathAnnotation] {
		return false
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
