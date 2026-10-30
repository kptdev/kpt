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

// Package krmvalidate is the shared implementation of "is this a valid
// KRM resource" and the tree-wide non-KRM pre-scan built on it, used by
// every kpt command that needs to detect non-KRM files consistently.
package krmvalidate

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	kustomizationAPIGroup = "kustomize.config.k8s.io"
	kustomizationKind     = "Kustomization"
)

// RecognizedKustomizationFileNames are the filenames kustomize itself
// recognizes as a kustomization root, regardless of content.
func RecognizedKustomizationFileNames() []string {
	return []string{
		"kustomization.yaml",
		"kustomization.yml",
		"Kustomization",
	}
}

// IsKRM reports whether n is a valid KRM resource: apiVersion and kind
// are always required. metadata.name is required too, except for
// kustomization files (any shape), "*List" kinds, resources using
// metadata.generateName instead, and resources annotated local-config.
func IsKRM(n *yaml.RNode) error {
	if isKustomization(n) {
		return nil
	}
	meta, err := n.GetMeta()
	if err != nil {
		return fmt.Errorf("resource must have `apiVersion`, `kind`, and `name`")
	}
	if meta.APIVersion == "" {
		return fmt.Errorf("resource must have `apiVersion`")
	}
	if meta.Kind == "" {
		return fmt.Errorf("resource must have `kind`")
	}
	if meta.Name == "" && !strings.HasSuffix(meta.Kind, "List") &&
		!hasGenerateName(n) && !isLocalConfig(meta) {
		return fmt.Errorf("resource must have `metadata.name`")
	}
	return nil
}

// hasGenerateName reports whether n sets metadata.generateName, a
// legitimate alternative to metadata.name that kyaml's typed
// ResourceMeta doesn't expose, so it needs raw node access.
func hasGenerateName(n *yaml.RNode) bool {
	v, _ := n.Pipe(yaml.Lookup("metadata", "generateName"))
	return v != nil && v.YNode().Value != ""
}

// isLocalConfig reports whether meta carries the local-config
// annotation with a non-"false" value, mirroring pkg/live's own
// filterLocalConfig so both agree on what's exempt from needing a name.
func isLocalConfig(meta yaml.ResourceMeta) bool {
	v, ok := meta.Annotations[filters.LocalConfigAnnotation]
	return ok && v != "false"
}

// AreKRM checks every node in nodes via IsKRM, stopping at the first
// failure. Prefer ScanTree for anything with tree-wide context; use
// AreKRM only where there's no tree to batch across.
func AreKRM(nodes []*yaml.RNode) error {
	for i := range nodes {
		if err := IsKRM(nodes[i]); err != nil {
			path, _, _ := kioutil.GetFileAnnotations(nodes[i])
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return nil
}

// isKustomization reports whether n is a kustomization file or resource.
func isKustomization(n *yaml.RNode) bool {
	resourcePath, _, err := kioutil.GetFileAnnotations(n)
	if err == nil {
		resourceFile := filepath.Base(resourcePath)
		if slices.Contains(RecognizedKustomizationFileNames(), resourceFile) {
			return true
		}
	}
	meta, err := n.GetMeta()
	if err != nil {
		return false
	}

	if strings.HasPrefix(meta.APIVersion, kustomizationAPIGroup) {
		return true
	}

	if meta.APIVersion == "" && meta.Kind == kustomizationKind {
		return true
	}

	return false
}
