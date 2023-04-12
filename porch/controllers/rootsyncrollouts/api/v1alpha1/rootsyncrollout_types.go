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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RootSyncRolloutSpec
type RootSyncRolloutSpec struct {
	Targets ClusterTargetSelector `json:"targets,omitempty"`

	Packages PackageSelector `json:"packages,omitempty"`
}

type ClusterTargetSelector struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type PackageSelector struct {
	Namespace string `json:"namespace,omitempty"`

	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// RootSyncRolloutStatus defines the observed state of RootSyncRollout
type RootSyncRolloutStatus struct {
	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	PackageStatuses []PackageStatus `json:"packageStatus,omitempty"`
}

type PackageStatus struct {
	Package string `json:"package"`

	Revisions []Revision `json:"revision"`
}

type Revision struct {
	Revision string `json:"revision"`

	Count int `json:"count"`

	SyncedCount int `json:"syncedCount"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RootSyncRollout is the Schema for the rootsyncrollouts API
type RootSyncRollout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RootSyncRolloutSpec   `json:"spec,omitempty"`
	Status RootSyncRolloutStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RootSyncRolloutList contains a list of RootSyncRollout
type RootSyncRolloutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootSyncRollout `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RootSyncRollout{}, &RootSyncRolloutList{})
}
