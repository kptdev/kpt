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

package walk

import (
	"lib.kpt.dev/yaml"
)

// walkAssociativeSequence descends into each element and pairs src and dest by an associative
// key, recursively invoking Filter on each match.
func (l Filter) walkAssociativeSequence(dest *yaml.RNode) error {
	key := dest.GetAssociativeKey()
	return l.Source.VisitElements(func(srcElem *yaml.RNode) error {
		value := srcElem.Field(key)
		destElem := dest.Element(key, value.Value.YNode().Value)

		// element is missing or empty in either the src or dest -- invoke the Visitor
		if yaml.IsEmpty(destElem) || yaml.IsEmpty(srcElem) {
			r, err := l.SetElement(srcElem, destElem)
			if err != nil || r == nil {
				return err
			}
			_, err = dest.Pipe(yaml.Append(r.YNode()))
			return err
		}

		// TODO: Support deletion of elements from the destination

		// element is present in both src and dest -- recurse on the element
		_, err := destElem.Pipe(Filter{Source: srcElem, Path: l.Path, Visitor: l})
		return err
	})
}
