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

// RootSyncDeploymentSpec defines the desired state of RootSyncDeployment
type RootSyncDeploymentSpec struct {
	Targets         ClusterTargetSelector `json:"targets,omitempty"`
	PackageRevision PackageRevisionRef    `json:"packageRevision,omitempty"`
}

type ClusterTargetSelector struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type PackageRevisionRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// RootSyncDeploymentStatus defines the observed state of RootSyncDeployment
type RootSyncDeploymentStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ClusterRefStatuses []ClusterRefStatus `json:"clusterRefStatuses,omitempty"`
}

type ClusterRefStatus struct {
	ApiVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Synced     bool   `json:"synced"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RootSyncDeployment is the Schema for the rootsyncdeployments API
type RootSyncDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RootSyncDeploymentSpec   `json:"spec,omitempty"`
	Status RootSyncDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RootSyncDeploymentList contains a list of RootSyncDeployment
type RootSyncDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootSyncDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RootSyncDeployment{}, &RootSyncDeploymentList{})
}
