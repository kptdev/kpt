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

package kioutil

import (
	"fmt"
	"sort"
	"strconv"

	"lib.kpt.dev/yaml"
)

type AnnotationKey = string

const (
	// IndexAnnotation records the index of a specific resource in a file or input stream.
	IndexAnnotation AnnotationKey = "kpt.dev/kio/index"

	// PathAnnotation records the path to the file the Resource was read from
	PathAnnotation AnnotationKey = "kpt.dev/kio/path"

	// PackageAnnotation records the name of the package the Resource was read from
	PackageAnnotation AnnotationKey = "kpt.dev/kio/package"
)

// ErrorIfMissingAnnotation validates the provided annotations are present on the given resources
func ErrorIfMissingAnnotation(nodes []*yaml.RNode, keys ...AnnotationKey) error {
	for _, key := range keys {
		for _, node := range nodes {
			val, err := node.Pipe(yaml.GetAnnotation(key))
			if err != nil {
				return err
			}
			if val == nil {
				return fmt.Errorf("missing package annotation %s", key)
			}
		}
	}
	return nil
}

// SortNodes sorts nodes in place:
// - by PathAnnotation annotation
// - by IndexAnnotation annotation
func SortNodes(nodes []*yaml.RNode) error {
	var err error
	// use stable sort to keep ordering of equal elements
	sort.SliceStable(nodes, func(i, j int) bool {
		if err != nil {
			return false
		}
		var iMeta, jMeta yaml.ResourceMeta
		if iMeta, _ = nodes[i].GetMeta(); err != nil {
			return false
		}
		if jMeta, _ = nodes[j].GetMeta(); err != nil {
			return false
		}

		iValue := iMeta.Annotations[PathAnnotation]
		jValue := jMeta.Annotations[PathAnnotation]
		if iValue != jValue {
			return iValue < jValue
		}

		iValue = iMeta.Annotations[IndexAnnotation]
		jValue = jMeta.Annotations[IndexAnnotation]

		// put resource config without an index first
		if iValue == jValue {
			return false
		}
		if iValue == "" {
			return true
		}
		if jValue == "" {
			return false
		}

		// sort by index
		var iIndex, jIndex int
		iIndex, err = strconv.Atoi(iValue)
		if err != nil {
			err = fmt.Errorf("unable to parse kpt.dev/kio/index %s :%v", iValue, err)
			return false
		}
		jIndex, err = strconv.Atoi(jValue)
		if err != nil {
			err = fmt.Errorf("unable to parse kpt.dev/kio/index %s :%v", jValue, err)
			return false
		}
		if iIndex != jIndex {
			return iValue < jValue
		}

		// elements are equal
		return false
	})
	return err
}
