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

package plan

import (
	"reflect"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Diff struct {
	Type  string
	Left  interface{}
	Right interface{}
	Path  string
}

func diffObjects(b, a *unstructured.Unstructured) ([]Diff, error) {
	diffs, err := diffMaps("", b.Object, a.Object)
	if err != nil {
		return nil, err
	}
	return diffs, nil
}

func diffMaps(prefix string, l, r map[string]interface{}) ([]Diff, error) {
	var diffs []Diff
	for k, lv := range l {
		childPrefix := prefix + "." + k

		rv, ok := r[k]
		if !ok {
			diffs = append(diffs, Diff{Type: "LeftAdd", Path: childPrefix, Left: lv, Right: rv})
			continue
		}
		childDiffs, err := diffValue(childPrefix, lv, rv)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, childDiffs...)
	}

	for k, rv := range r {
		childPrefix := prefix + "." + k

		lv, ok := l[k]
		if !ok {
			diffs = append(diffs, Diff{Type: "RightAdd", Path: childPrefix, Left: lv, Right: rv})
			continue
		}
	}

	return diffs, nil
}

func diffSlices(prefix string, l, r []interface{}) ([]Diff, error) {
	var diffs []Diff
	for i, lv := range l {
		childPrefix := prefix + "." + strconv.Itoa(i)

		if len(r) <= i {
			diffs = append(diffs, Diff{Type: "LeftAdd", Path: childPrefix, Left: l[i], Right: nil})
			continue
		}
		rv := r[i]
		childDiffs, err := diffValue(childPrefix, lv, rv)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, childDiffs...)
	}

	for i, rv := range r {
		childPrefix := prefix + "." + strconv.Itoa(i)

		if len(l) <= i {
			diffs = append(diffs, Diff{Type: "RightAdd", Path: childPrefix, Left: nil, Right: rv})
			continue
		}
	}

	return diffs, nil
}

func diffValue(path string, lv, rv interface{}) ([]Diff, error) {
	switch lv := lv.(type) {
	// case string:
	// 	rvString, ok := rv.(string)
	// 	if !ok || lv != rvString {
	// 		diffs = append(diffs, Diff{Type: "Change", Path: childPrefix, Left: lv, Right: rv})
	// 	}
	// case int64:
	// 	rvInt64, ok := rv.(int64)
	// 	if !ok || lv != rvInt64 {
	// 		diffs = append(diffs, Diff{Type: "Change", Path: childPrefix, Left: lv, Right: rv})
	// 	}
	case map[string]interface{}:
		rvMap, ok := rv.(map[string]interface{})
		if !ok {
			return []Diff{{Type: "Change", Path: path, Left: lv, Right: rv}}, nil
		}
		return diffMaps(path, lv, rvMap)

	case []interface{}:
		rvSlice, ok := rv.([]interface{})
		if !ok {
			return []Diff{{Type: "Change", Path: path, Left: lv, Right: rv}}, nil
		}
		return diffSlices(path, lv, rvSlice)

	default:
		if !reflect.DeepEqual(lv, rv) {
			return []Diff{{Type: "Change", Path: path, Left: lv, Right: rv}}, nil
		}
		return nil, nil
	}
}
