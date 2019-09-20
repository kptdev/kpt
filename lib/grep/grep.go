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

package grep

import (
	"lib.kpt.dev/yaml"

	"lib.kpt.dev/kio"
)

func init() {
	kio.Filters["GrepFilter"] = func() kio.Filter { return Filter{} }
}

// Filter filters RNodes with a matching field
type Filter struct {
	Path  []string `yaml:"path,omitempty"`
	Value string   `yaml:"value,omitempty"`
}

var _ kio.Filter = Filter{}

func (f Filter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	var output kio.ResourceNodeSlice
	for i := range input {
		node := input[i]
		val, err := node.Pipe(yaml.Lookup(f.Path...), yaml.Match(f.Value))
		if err != nil {
			return nil, err
		}
		if val != nil {
			output = append(output, input[i])
		}
	}
	return output, nil
}
