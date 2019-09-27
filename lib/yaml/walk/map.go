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

// walkMap descends into each map key-value pair, recursively invoking Filter on each match --
// calling SetMapField on each pair where either the src or dest has an empty or missing value.
func (l Filter) walkMap(dest *yaml.RNode) error {
	return l.Source.VisitFields(func(srcField *yaml.MapNode) error {
		fieldName := srcField.Key.YNode().Value
		destField := dest.Field(fieldName)

		// field is missing from either the src or dest -- invoke the Visitor
		if yaml.IsFieldEmpty(destField) || yaml.IsFieldEmpty(srcField) {
			r, err := l.SetMapField(srcField, destField)
			if err != nil || r == nil {
				if yaml.IsEmpty(destField.Value) {
					// remove the field if it has been cleared
					_, err = dest.Pipe(yaml.Clear(fieldName))
				}
				return err
			}
			if yaml.IsEmpty(r) {
				_, err = dest.Pipe(yaml.Clear(fieldName))
			} else {
				_, err = dest.Pipe(yaml.SetField(fieldName, r))
			}
			return err
		}

		// field is present in both src and dest -- recurse on the values
		_, err := destField.Value.Pipe(
			Filter{Visitor: l, Source: srcField.Value, Path: append(l.Path, fieldName)})
		if err != nil {
			return err
		}

		// clear the field if the value is empty
		if yaml.IsEmpty(srcField.Value) {
			_, err = dest.Pipe(yaml.Clear(fieldName))
		} else {
			if err = l.SetComments(srcField.Key, destField.Key); err != nil {
				return err
			}
			if err = l.SetComments(srcField.Value, destField.Value); err != nil {
				return err
			}
		}
		return err
	})
}
