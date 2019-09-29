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

package filters

import (
	"regexp"
	"strings"

	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

type GrepType int

const (
	Regexp GrepType = 1 << iota
	GreaterThanEq
	GreaterThan
	LessThan
	LessThanEq
)

// GrepFilter filters RNodes with a matching field
type GrepFilter struct {
	Path        []string `yaml:"path,omitempty"`
	Value       string   `yaml:"value,omitempty"`
	MatchType   GrepType `yaml:"matchType,omitempty"`
	InvertMatch bool     `yaml:"invertMatch,omitempty"`
	Compare     func(a, b string) (int, error)
}

var _ kio.Filter = GrepFilter{}

func (f GrepFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	// compile the regular expression 1 time if we are matching using regex
	var reg *regexp.Regexp
	var err error
	if f.MatchType == Regexp || f.MatchType == 0 {
		reg, err = regexp.Compile(f.Value)
		if err != nil {
			return nil, err
		}
	}

	var output kio.ResourceNodeSlice
	for i := range input {
		node := input[i]
		val, err := node.Pipe(&yaml.PathMatcher{Path: f.Path})
		if err != nil {
			return nil, err
		}
		if val == nil || len(val.Content()) == 0 {
			if f.InvertMatch {
				output = append(output, input[i])
			}
			continue
		}
		found := false
		err = val.VisitElements(func(elem *yaml.RNode) error {
			// get the value
			var str string
			if f.MatchType == Regexp {
				style := elem.YNode().Style
				defer func() { elem.YNode().Style = style }()
				elem.YNode().Style = yaml.FlowStyle
				str, err = elem.String()
				if err != nil {
					return err
				}
				str = strings.TrimSpace(strings.Replace(str, `"`, "", -1))
			} else {
				// if not regexp, then it needs to parse into a quantity and comments will
				// break that
				str = elem.YNode().Value
				if str == "" {
					return nil
				}
			}

			if f.MatchType == Regexp || f.MatchType == 0 {
				if reg.MatchString(str) {
					found = true
				}
				return nil
			}

			comp, err := f.Compare(str, f.Value)
			if err != nil {
				return err
			}

			if f.MatchType == GreaterThan && comp > 0 {
				found = true
			}
			if f.MatchType == GreaterThanEq && comp >= 0 {
				found = true
			}
			if f.MatchType == LessThan && comp < 0 {
				found = true
			}
			if f.MatchType == LessThanEq && comp <= 0 {
				found = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if found == f.InvertMatch {
			continue
		}

		output = append(output, input[i])
	}
	return output, nil
}
