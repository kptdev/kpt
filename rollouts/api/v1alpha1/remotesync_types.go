/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RemoteSyncSpec defines the desired state of RemoteSync
type RemoteSyncSpec struct {
	// ClusterReference contains the identify information need to refer a cluster.
	ClusterRef ClusterRef       `json:"clusterRef,omitempty"`
	Template   *Template        `json:"template,omitempty"`
	Type       SyncTemplateType `json:"type,omitempty"`
}

type Template struct {
	Spec     *SyncSpec `json:"spec,omitempty"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

type SyncSpec struct {
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

// Metadata specifies labels and annotations to add to the RSync object.
type Metadata struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RemoteSyncStatus defines the observed state of RemoteSync
type RemoteSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// SyncStatus describes the observed state of external sync.
	SyncStatus string `json:"syncStatus,omitempty"`

	// Internal only. SyncCreated describes if the external sync has been created.
	SyncCreated bool `json:"syncCreated"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RemoteSync is the Schema for the remotesyncs API
type RemoteSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemoteSyncSpec   `json:"spec,omitempty"`
	Status RemoteSyncStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RemoteSyncList contains a list of RemoteSync
type RemoteSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemoteSync{}, &RemoteSyncList{})
}
