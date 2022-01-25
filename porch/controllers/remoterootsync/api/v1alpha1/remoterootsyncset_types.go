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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RemoteRootSyncSet represents applying a package to multiple target clusters.
// In future, this should use ConfigSync, but while we're iterating on OCI/porch support,
// and making a few similar iterations (e.g. what feedback do we need for rollout),
// we're just applying directly to the target cluster(s).
//
// We follow the "managed remote objects" pattern; we don't want to create a mirror
// object, so we start with the "ReplicaSet" of Pod/ReplicaSet/Deployment.
//
// spec.clusterRefs specifies the target clusters
//
// spec.template maps to the spec of our "Pod", in this case a ConfigSync RootSync/RepoSync.
// Because we're not actually using ConfigSync in this prototype, we are only defining a
// small subset of fields.
type RemoteRootSyncSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemoteRootSyncSetSpec   `json:"spec,omitempty"`
	Status RemoteRootSyncSetStatus `json:"status,omitempty"`
}

// RemoteRootSyncSetSpec defines the desired state of RemoteRootSync
type RemoteRootSyncSetSpec struct {
	ClusterRefs []*ClusterRef     `json:"clusterRefs,omitempty"`
	Template    *RootSyncTemplate `json:"template,omitempty"`
}

type ClusterRef struct {
	ApiVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
}

type RootSyncTemplate struct {
	SourceFormat string `json:"sourceFormat,omitempty"`
	// Git          *GitInfo `json:"git,omitempty"`
	OCI *OCISpec `json:"oci,omitempty"`
}

type OCISpec struct {
	Repository string `json:"repository,omitempty"`
}

// RootSyncSetStatus defines the observed state of RootSyncSet
type RemoteRootSyncSetStatus struct {
	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true

// RemoteRootSyncSetList contains a list of RemoteRootSyncSet
type RemoteRootSyncSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteRootSyncSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemoteRootSyncSet{}, &RemoteRootSyncSetList{})
}
