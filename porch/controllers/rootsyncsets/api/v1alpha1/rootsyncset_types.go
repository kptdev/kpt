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

// RootSyncSetSpec defines the desired state of RootSyncSet
type RootSyncSetSpec struct {
	ClusterRefs []*ClusterRef `json:"clusterRefs,omitempty"`
	Template    *RootSyncInfo `json:"template,omitempty"`
}

type ClusterRef struct {
	ApiVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
}

func (r *ClusterRef) GetKind() string {
	return r.Kind
}

func (r *ClusterRef) GetName() string {
	return r.Name
}

func (r *ClusterRef) GetNamespace() string {
	return r.Namespace
}

func (r *ClusterRef) GetAPIVersion() string {
	return r.ApiVersion
}

type RootSyncInfo struct {
	Spec *RootSyncSpec `json:"spec,omitempty"`
}

type RootSyncSpec struct {
	SourceFormat string   `json:"sourceFormat,omitempty"`
	Git          *GitInfo `json:"git,omitempty"`
}

type GitInfo struct {
	Repo                   string          `json:"repo"`
	Branch                 string          `json:"branch,omitempty"`
	Revision               string          `json:"revision,omitempty"`
	Dir                    string          `json:"dir,omitempty"`
	Period                 metav1.Duration `json:"period,omitempty"`
	Auth                   string          `json:"auth"`
	GCPServiceAccountEmail string          `json:"gcpServiceAccountEmail,omitempty"`
	Proxy                  string          `json:"proxy,omitempty"`
	SecretRef              SecretReference `json:"secretRef,omitempty"`
	NoSSLVerify            bool            `json:"noSSLVerify,omitempty"`
}

// SecretReference contains the reference to the secret used to connect to
// Git source of truth.
type SecretReference struct {
	// Name represents the secret name.
	// +optional
	Name string `json:"name,omitempty"`
}

// RootSyncSetStatus defines the observed state of RootSyncSet
type RootSyncSetStatus struct {
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
	SyncStatus string `json:"syncStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RootSyncSet is the Schema for the rootsyncsets API
type RootSyncSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RootSyncSetSpec   `json:"spec,omitempty"`
	Status RootSyncSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RootSyncSetList contains a list of RootSyncSet
type RootSyncSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RootSyncSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RootSyncSet{}, &RootSyncSetList{})
}
