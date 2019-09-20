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

import "gopkg.in/yaml.v3"

// AnnotationClearer removes an annotation at metadata.annotations.
// Returns nil if the annotation or field does not exist.
type AnnotationClearer struct {
	Kind string `yaml:"kind,omitempty"`
	Key  string `yaml:"key,omitempty"`
}

func (c AnnotationClearer) Filter(rn *RNode) (*RNode, error) {
	return rn.Pipe(
		PathGetter{Path: []string{"metadata", "annotations"}},
		FieldClearer{Name: c.Key})
}

func ClearAnnotation(key string) AnnotationClearer {
	return AnnotationClearer{Key: key}
}

// AnnotationSetter sets an annotation at metadata.annotations.
// Creates metadata.annotations if does not exist.
type AnnotationSetter struct {
	Kind  string `yaml:"kind,omitempty"`
	Key   string `yaml:"key,omitempty"`
	Value string `yaml:"value,omitempty"`
}

func (s AnnotationSetter) Filter(rn *RNode) (*RNode, error) {
	return rn.Pipe(
		PathGetter{Path: []string{"metadata", "annotations"}, Create: yaml.MappingNode},
		FieldSetter{Name: s.Key, Value: NewScalarRNode(s.Value)})
}

func SetAnnotation(key, value string) AnnotationSetter {
	return AnnotationSetter{Key: key, Value: value}
}

// AnnotationGetter gets an annotation at metadata.annotations.
// Returns nil if metadata.annotations does not exist.
type AnnotationGetter struct {
	Kind  string `yaml:"kind,omitempty"`
	Key   string `yaml:"key,omitempty"`
	Value string `yaml:"value,omitempty"`
}

// AnnotationGetter returns the annotation value.
// Returns "", nil if the annotation does not exist.
func (g AnnotationGetter) Filter(rn *RNode) (*RNode, error) {
	v, err := rn.Pipe(PathGetter{Path: []string{"metadata", "annotations", g.Key}})
	if v == nil || err != nil {
		return v, err
	}
	if g.Value == "" || v.value.Value == g.Value {
		return v, err
	}
	return nil, err
}

func GetAnnotation(key string) AnnotationGetter {
	return AnnotationGetter{Key: key}
}
