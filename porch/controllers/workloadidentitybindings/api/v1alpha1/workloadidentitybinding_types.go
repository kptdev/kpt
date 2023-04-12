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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].reason`

// WorkloadIdentityBinding
type WorkloadIdentityBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadIdentityBindingSpec   `json:"spec,omitempty"`
	Status WorkloadIdentityBindingStatus `json:"status,omitempty"`
}

// WorkloadIdentityBindingSpec defines the desired state of WorkloadIdentityBinding
type WorkloadIdentityBindingSpec struct {
	KubernetesServiceAccountRef KubernetesServiceAccountRef `json:"kubernetesServiceAccountRef,omitempty"`
	GcpServiceAccountRef        GcpServiceAccountRef        `json:"gcpServiceAccountRef,omitempty"`
}

type KubernetesServiceAccountRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type GcpServiceAccountRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	External  string `json:"external,omitempty"`
}

// WorkloadIdentityBindingStatus defines the observed state of WorkloadIdentityBinding
type WorkloadIdentityBindingStatus struct {
	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true

// WorkloadIdentityBindingList contains a list of WorkloadIdentityBinding
type WorkloadIdentityBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadIdentityBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkloadIdentityBinding{}, &WorkloadIdentityBindingList{})
}
