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

// Package merge contains libraries for merging fields from one RNode to another
// RNode
package merge3

import (
	"lib.kpt.dev/yaml"
	"lib.kpt.dev/yaml/walk"
)

const Help = `
Description:

  merge3 identifies changes between an original source + updated source and merges the result
  into a destination, overriding the destination fields where they have changed between
  original and updated.

  ### Merge Rules

  Fields are recursively merged using the following rules:

  - scalars
    - if present in either dest or updated and 'null', clear the value
    - if unchanged between original and updated, keep dest value
    - if changed between original and updated (added, deleted, changed), take the updated value

  - non-associative lists -- lists without a merge key
    - if present in either dest or updated and 'null', clear the value
    - if unchanged between original and updated, keep dest value
    - if changed between original and updated (added, deleted, changed), take the updated value

  - map keys and fields -- paired by the map-key / field-name
    - if present in either dest or updated and 'null', clear the value
    - if present only in the dest, it keeps its value
    - if not-present in the dest, add the delta between original-updated as a field
    - otherwise recursively merge the value between original, updated, dest

  - associative list elements -- paired by the associative key
    - if present only in the dest, it keeps its value
    - if not-present in the dest, add the delta between original-updated as a field
    - otherwise recursively merge the value between original, updated, dest

  ### Associative Keys

  Associative keys are used to identify "same" elements within 2 different lists, and merge them.
  The following fields are recognized as associative keys:

` + "[`mountPath`, `devicePath`, `ip`, `type`, `topologyKey`, `name`, `containerPort`]" + `

  Any lists where all of the elements contain associative keys will be merged as associative lists.
`

func Merge(srcOriginal, srcUpdated, dest *yaml.RNode) (*yaml.RNode, error) {
	return walk.Walker{Visitor: Visitor{},
		Sources: []*yaml.RNode{srcOriginal, srcUpdated, dest}}.Walk()
}

func MergeStrings(srcOriginalStr, srcUpdatedStr, destStr string) (string, error) {
	srcOriginal, err := yaml.Parse(srcOriginalStr)
	if err != nil {
		return "", err
	}
	srcUpdated, err := yaml.Parse(srcUpdatedStr)
	if err != nil {
		return "", err
	}
	dest, err := yaml.Parse(destStr)
	if err != nil {
		return "", err
	}

	result, err := Merge(srcOriginal, srcUpdated, dest)
	if err != nil {
		return "", err
	}
	return result.String()
}
