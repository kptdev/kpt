// Copyright 2020 Google LLC
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

package pkgbuilder

import "sigs.k8s.io/kustomize/kyaml/yaml"

// SetFieldPath returns a new FieldPathSetter that updates the property
// given by the path with the given value.
func SetFieldPath(value string, path ...string) FieldPathSetter {
	return FieldPathSetter{
		Value: value,
		Path:  path,
	}
}

// FieldPathSetter updates the value of the field given by the path.
type FieldPathSetter struct {
	Path []string

	Value string
}

func (f FieldPathSetter) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
	n, err := rn.Pipe(yaml.PathGetter{
		Path: f.Path,
	})
	if err != nil {
		return n, err
	}

	n.YNode().Value = f.Value
	return rn, nil
}
