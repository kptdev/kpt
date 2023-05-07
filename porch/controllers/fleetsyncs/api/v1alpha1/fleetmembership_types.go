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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FleetMembershipSpec struct {
	Description string `json:"description,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	State MembershipState `json:"state,omitempty"`
}

type MembershipState struct {
	Code MembershipStateCode `json:"code,omitempty"`
}

type MembershipStateCode string

const (
	CodeUnspecified     MembershipStateCode = "unspecified"
	CodeCreating        MembershipStateCode = "creating"
	CodeReady           MembershipStateCode = "ready"
	CodeDeleting        MembershipStateCode = "deleting"
	CodeUpdating        MembershipStateCode = "updating"
	CodeServiceUpdating MembershipStateCode = "serviceupdating"
)

type FleetMembershipStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type FleetMembership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetMembershipSpec   `json:"spec,omitempty"`
	Status FleetMembershipStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type FleetMembershipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FleetMembership `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FleetMembership{}, &FleetMembershipList{})
}
