// Copyright 2022 Google LLC
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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var ResourceMeta = yaml.ResourceMeta{
	TypeMeta: yaml.TypeMeta{
		APIVersion: "config.google.com/v1alpha1",
		Kind:       "Plan",
	},
	ObjectMeta: yaml.ObjectMeta{
		NameMeta: yaml.NameMeta{
			Name: "plan",
		},
		Annotations: map[string]string{
			"config.kubernetes.io/local-config": "true",
		},
	},
}

type Plan struct {
	yaml.ResourceMeta `yaml:",inline" json:",inline"`

	Spec PlanSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type PlanSpec struct {
	Actions []Action `json:"actions,omitempty"`
}

type ActionType string

const (
	Create    ActionType = "Create"
	Unchanged ActionType = "Unchanged"
	Delete    ActionType = "Delete"
	Update    ActionType = "Update"
	Skip      ActionType = "Skip"
	Error     ActionType = "Error"
)

type Action struct {
	Type      ActionType                 `json:"action,omitempty" yaml:"action,omitempty"`
	Group     string                     `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind      string                     `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name      string                     `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string                     `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Before    *unstructured.Unstructured `json:"before,omitempty" yaml:"before,omitempty"`
	After     *unstructured.Unstructured `json:"after,omitempty" yaml:"after,omitempty"`
	Error     string                     `json:"error,omitempty" yaml:"error,omitempty"`
}
