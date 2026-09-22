// Copyright 2025-2026 The kpt Authors
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
	"log"
	"slices"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func asString(node *yaml.Node) (string, bool) {
	if node.Kind == yaml.ScalarNode && (node.Tag == tagString || node.Tag == "") {
		return node.Value, true
	}
	return "", false
}

func getValueNode(m *yaml.Node, key string) (*yaml.Node, bool) {
	children := m.Content
	if len(children)%2 != 0 {
		log.Fatalf("unexpected number of children for map %d", len(children))
	}

	for i := 0; i < len(children); i += 2 {
		keyNode := children[i]

		k, ok := asString(keyNode)
		if ok && k == key {
			valueNode := children[i+1]
			return valueNode, true
		}
	}
	return nil, false
}

// findMapKey finds `key` in the Content list of a mapping node `node`.
// Returns with the index of the key node if was found, and -1 if not
func findMapKey(node *yaml.Node, key string) int {
	return findMapKeyAfter(node, key, 0)
}

// findMapKey finds `key` in the Content list of a mapping node `node`.
// It starts searching from the `startIdx` position of the nodes' Content.
// Returns with the index of the key node if was found, and -1 if not
func findMapKeyAfter(node *yaml.Node, key string, startIdx int) int {
	children := node.Content
	if len(children)%2 != 0 {
		log.Fatalf("couldn't find field %q in map node: unexpected number of children %d", key, len(children))
	}

	for i := startIdx; i < len(children); i += 2 {
		keyNode := children[i]
		k, ok := asString(keyNode)
		if ok && k == key {
			return i
		}
	}
	return -1
}

// shallowCopyFormatting copies comments from `src` to `dst` non-recursively
func shallowCopyFormatting(src, dst *yaml.Node) {
	dst.Style = src.Style
	dst.HeadComment = src.HeadComment
	dst.LineComment = src.LineComment
	dst.FootComment = src.FootComment
	dst.Line = src.Line
	dst.Column = src.Column
}

// deepCopyFormatting copies formatting (comments and order of fields) from `src` to `dst` recursively
func deepCopyFormatting(src, dst *yaml.Node) {
	if src.Kind != dst.Kind {
		return
	}

	switch dst.Kind {
	case yaml.MappingNode:
		copyMapFormatting(src, dst)
	case yaml.SequenceNode, yaml.DocumentNode:
		copyListFormatting(src, dst)
	case yaml.DocumentNode:
		if len(src.Content) == 1 && len(dst.Content) == 1 {
			// for single-item document lists always copy formatting
			// (not only if the nodes are equal)
			shallowCopyFormatting(src, dst)
			deepCopyFormatting(src.Content[0], dst.Content[0])
		} else {
			// this shouldn't really happen with YAML nodes in KubeObjects
			copyListFormatting(src, dst)
		}
	default:
		shallowCopyFormatting(src, dst)
	}
}

// copyMapFormatting copies formatting between MappingNodes recursively
func copyMapFormatting(src, dst *yaml.Node) {
	if (len(src.Content)%2 != 0) || (len(dst.Content)%2 != 0) {
		log.Fatalf("copy formatting: unexpected number of children of mapping node (%d or %d)", len(src.Content), len(dst.Content))
	}

	// keep comments
	shallowCopyFormatting(src, dst)

	// keep ordering: swap dst fields that match src fields to the beginning of dst, in the original order of src fields
	newDstIdx := 0
	for srcIdx := 0; srcIdx < len(src.Content); srcIdx += 2 {
		key, ok := asString(src.Content[srcIdx])
		if !ok {
			continue
		}
		origDstIdx := findMapKeyAfter(dst, key, newDstIdx)
		if origDstIdx < 0 {
			continue
		}
		// swap field to match order in src
		if origDstIdx != newDstIdx {
			dst.Content[newDstIdx], dst.Content[origDstIdx] = dst.Content[origDstIdx], dst.Content[newDstIdx]
			dst.Content[newDstIdx+1], dst.Content[origDstIdx+1] = dst.Content[origDstIdx+1], dst.Content[newDstIdx+1]
		}
		// copy formatting from the matching src field
		shallowCopyFormatting(src.Content[srcIdx], dst.Content[newDstIdx])
		deepCopyFormatting(src.Content[srcIdx+1], dst.Content[newDstIdx+1])
		newDstIdx += 2
	}
}

// copyListFormatting copies formatting between SequenceNodes recursively.
// List items are handled as atomic types: If an item of `dst` also found
// in `src` (with an exactly matching value), the formatting is copied
// from `src` to `dst`.
func copyListFormatting(src, dst *yaml.Node) {
	// keep comments
	shallowCopyFormatting(src, dst)

	// shallow copy source list items to a new slice
	srcItems := slices.Clone(src.Content)

	for _, dstItem := range dst.Content {
		for srcIdx, srcItem := range srcItems {
			if nodeDeepEqualValue(srcItem, dstItem) {
				// copy formatting from srcItem to dstItem
				deepCopyFormatting(srcItem, dstItem)
				// remove srcItem from the list
				srcItems = slices.Delete(srcItems, srcIdx, srcIdx+1)
				break
			}
		}
	}
}

// nodeDeepEqualValue returns whether `a` and `b` encode the exact same values (ignores formatting)
func nodeDeepEqualValue(a, b *yaml.Node) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case yaml.ScalarNode:
		return a.Value == b.Value

	case yaml.MappingNode:
		if len(a.Content) != len(b.Content) {
			return false
		}
		if len(a.Content)%2 != 0 {
			log.Fatalf("unexpected number of children for YAML map")
		}
		for aIdx := 0; aIdx < len(a.Content); aIdx += 2 {
			key, ok := asString(a.Content[aIdx])
			if !ok {
				return false
			}
			bIdx := findMapKey(b, key)
			if bIdx < 0 {
				return false
			}
			if !nodeDeepEqualValue(a.Content[aIdx+1], b.Content[bIdx+1]) {
				return false
			}
		}
		return true

	case yaml.SequenceNode, yaml.DocumentNode:
		if len(a.Content) != len(b.Content) {
			return false
		}
		for i := range a.Content {
			if !nodeDeepEqualValue(a.Content[i], b.Content[i]) {
				return false
			}
		}
		return true

	case yaml.AliasNode:
		// skip Alias nodes
		// TODO: check AliasNode properly?
		return true
	default:
		log.Fatalf("unexpected YAML node type: %v", a.Kind)
		return false
	}
}
