// Copyright 2022 The kpt Authors
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

package live

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Flatten returns a list containing 'in' objects with objects of kind List
// replaced by their members.
func Flatten(in []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	var out []*unstructured.Unstructured

	for _, o := range in {
		if o.IsList() {
			err := o.EachListItem(func(item runtime.Object) error {
				item2 := item.(*unstructured.Unstructured)
				out = append(out, item2)
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			out = append(out, o)
		}
	}
	return out, nil
}
