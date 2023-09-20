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

type FleetMembershipBindingData struct {
	FullName   string `json:"name,omitempty"`
	Project    string `json:"project,omitempty"`
	Location   string `json:"location,omitempty"`
	Membership string `json:"membership",omitempty"`
	Binding    string `json:"binding,omitempty"`

	ScopeFullName string `json:"scopeFullName,omitempty"`
	ScopeProject  string `json:"scopeProject,omitempty"`
	ScopeLocation string `json:"scopeLocation,omitempty"`
	Scope         string `json:"scope,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	State MembershipBindingState `json:"state,omitempty"`
}

type MembershipBindingState struct {
	Code MembershipBindingStateCode `json:"code,omitempty"`
}

type MembershipBindingStateCode string

const (
	MBSCodeUnspecified MembershipBindingStateCode = "unspecified"
	MBSCodeCreating    MembershipBindingStateCode = "creating"
	MBSCodeReady       MembershipBindingStateCode = "ready"
	MBSCodeDeleting    MembershipBindingStateCode = "deleting"
	MBSCodeUpdating    MembershipBindingStateCode = "updating"
)

type FleetMembershipBindingStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type FleetMembershipBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Data contains the discovered (synced) information
	Data   FleetMembershipBindingData   `json:"data,omitempty"`
	Status FleetMembershipBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type FleetMembershipBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FleetMembershipBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FleetMembershipBinding{}, &FleetMembershipBindingList{})
}
