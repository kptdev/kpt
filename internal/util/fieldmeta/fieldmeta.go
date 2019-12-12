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
	"bytes"
	"encoding/json"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Copied from kpt
type FieldMeta struct {
	Substitutions []Substitution `yaml:"substitutions,omitempty" json:"substitutions,omitempty"`
	SetBy         string         `yaml:"ownedBy,omitempty" json:"ownedBy,omitempty"`
	DefaultedBy   string         `yaml:"defaultedBy,omitempty" json:"defaultedBy,omitempty"`
	Description   string         `yaml:"description,omitempty" json:"description,omitempty"`
}

type Substitution struct {
	Name   string `yaml:"name,omitempty" json:"name,omitempty"`
	Marker string `yaml:"marker,omitempty" json:"marker,omitempty"`
	Value  string `yaml:"value,omitempty" json:"value,omitempty"`
}

func (fm *FieldMeta) Read(n *yaml.RNode) error {
	if n.YNode().LineComment != "" {
		v := strings.TrimLeft(n.YNode().LineComment, "#")
		// if it doesn't Unmarshal that is fine
		d := yaml.NewDecoder(bytes.NewBuffer([]byte(v)))
		d.KnownFields(false)
		_ = d.Decode(fm)
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
