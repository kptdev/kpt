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

package yaml

import (
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PathGetter returns the RNode under Path.
type PathMatcher struct {
	child bool

	Kind string `yaml:"kind,omitempty"`

	// Path is a slice of parts leading to the RNode to lookup.
	// Each path part may be one of:
	// * FieldMatcher -- e.g. "spec"
	// * Map Key -- e.g. "app.k8s.io/version"
	// * List Entry -- e.g. "[name=nginx]" or "[=-jar]"
	//
	// Map Keys and Fields are equivalent.
	// See FieldMatcher for more on Fields and Map Keys.
	//
	// List Entries are specified as map entry to match [fieldName=fieldValue].
	// See Elem for more on List Entries.
	//
	// Examples:
	// * spec.template.spec.container with matching name: [name=nginx]
	// * spec.template.spec.container.argument matching a value: [=-jar]
	Path []string `yaml:"path,omitempty"`

	Match []string
}

func (l *PathMatcher) Filter(rn *RNode) (*RNode, error) {
	if len(l.Path) == 0 {
		// found the node
		return rn, nil
	}
	if !l.child {
		l.Path = cleanPath(l.Path)
	}
	if IsListIndex(l.Path[0]) {
		return l.doElem(rn)
	} else {
		return l.doField(rn)
	}
}

func (l *PathMatcher) doElem(rn *RNode) (*RNode, error) {
	name, match, err := SplitIndexNameValue(l.Path[0])
	if err != nil {
		return nil, err
	}
	node := NewRNode(&yaml.Node{Kind: yaml.SequenceNode})

	// add each of the matching elements to a sequence return value
	err = rn.VisitElements(func(elem *RNode) error {
		if name == "" {
			// primitive type
			s, err := elem.String()
			if err != nil {
				return err
			}
			match, err := filepath.Match(match, strings.TrimSpace(s))
			if err != nil || !match {
				return err
			}
			node.YNode().Content = append(node.YNode().Content, elem.YNode())
		}

		// find each of the matching elements
		val := elem.Field(name)
		if val == nil || val.Value == nil {
			return nil
		}
		s, err := val.Value.String()
		if err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		match, err := filepath.Match(match, s)
		if err != nil || !match {
			return err
		}

		l.Match = append(l.Match, s)

		add, err := elem.Pipe(&PathMatcher{Path: l.Path[1:], child: true})
		if err != nil || add == nil {
			return err
		}
		if add.YNode().Kind == yaml.SequenceNode {
			// returns a sequence, add all of the elements
			node.YNode().Content = append(node.YNode().Content, add.YNode().Content...)
		} else {
			// add the matching value
			node.YNode().Content = append(node.YNode().Content, add.YNode())
		}
		return nil
	})
	if err != nil || len(node.Content()) == 0 {
		// error or no results
		return nil, err
	}
	return node, nil
}

func (l *PathMatcher) doField(rn *RNode) (*RNode, error) {
	field, err := rn.Pipe(Get(l.Path[0]))
	if err != nil || field == nil {
		return nil, err
	}
	return field.Pipe(&PathMatcher{Path: l.Path[1:], child: true})
}

func cleanPath(path []string) []string {
	var p []string
	for _, elem := range path {
		elem = strings.TrimSpace(elem)
		if len(elem) == 0 {
			continue
		}
		p = append(p, elem)
	}
	return p
}
