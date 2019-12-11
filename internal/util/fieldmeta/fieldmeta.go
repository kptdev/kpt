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

package fieldmeta

import (
	"encoding/json"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type FieldMeta struct {
	Substitutions []Substitution `yaml:"substitutions,omitempty" json:"substitutions,omitempty"`
	SetBy         *SetBy         `yaml:"setBy,omitempty" json:"setBy,omitempty"`
	DefaultedBy   *DefaultedBy   `yaml:"defaultedBy,omitempty" json:"defaultedBy,omitempty"`
}

type SetBy struct {
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`
}

type DefaultedBy struct {
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`
}

type Substitution struct {
	Name   string `yaml:"name,omitempty" json:"name,omitempty"`
	Marker string `yaml:"marker,omitempty" json:"marker,omitempty"`
	Value  string `yaml:"value,omitempty" json:"value,omitempty"`
}

func (fm *FieldMeta) Read(n *yaml.RNode) error {
	if n.YNode().LineComment != "" {
		v := strings.TrimLeft(n.YNode().LineComment, "#")
		err := yaml.Unmarshal([]byte(v), fm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fm *FieldMeta) Write(n *yaml.RNode) error {
	b, err := json.Marshal(fm)
	if err != nil {
		return err
	}
	n.YNode().LineComment = string(b)
	return nil
}
