// Copyright 2023 The kpt Authors
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Plan stores the expected changes when applying changes to a cluster.
// For consistency, we represent it as a KRM object, but we don't expect it to be applied to a cluster.
type Plan struct {
	metav1.TypeMeta

	Spec PlanSpec `json:"spec,omitempty"`
}

// PlanSpec is the Spec for a Plan object.
type PlanSpec struct {
	Actions []Action `json:"actions,omitempty"`
}

// ActionType is an enum type for the type of change (no-change/create/update etc).
type ActionType string

// Action represents an individual object change.
type Action struct {
	Type       ActionType                 `json:"action,omitempty"`
	APIVersion string                     `json:"apiVersion,omitempty"`
	Kind       string                     `json:"kind,omitempty"`
	Name       string                     `json:"name,omitempty"`
	Namespace  string                     `json:"namespace,omitempty"`
	Object     *unstructured.Unstructured `json:"object,omitempty"`
}
