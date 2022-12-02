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

// DownstreamPackage represents an upstream and downstream porch package pair.
// The upstream package should already exist. The DownstreamPackage controller is
// responsible for creating the downstream package revisions based on the spec.
type DownstreamPackage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DownstreamPackageSpec   `json:"spec,omitempty"`
	Status DownstreamPackageStatus `json:"status,omitempty"`
}

func (o *DownstreamPackage) GetSpec() *DownstreamPackageSpec {
	if o == nil {
		return nil
	}
	return &o.Spec
}

type AdoptionPolicy string
type DeletionPolicy string

const (
	AdoptionPolicyAdoptExisting AdoptionPolicy = "adoptExisting"
	AdoptionPolicyAdoptNone     AdoptionPolicy = "adoptNone"

	DeletionPolicyDelete = "delete"
	DeletionPolicyDisown = "disown"
)

// DownstreamPackageSpec defines the desired state of DownstreamPackage
type DownstreamPackageSpec struct {
	Upstream   *Upstream   `json:"upstream,omitempty"`
	Downstream *Downstream `json:"downstream,omitempty"`

	AdoptionPolicy AdoptionPolicy `json:"adoptionPolicy,omitempty"`
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

type Upstream struct {
	Repo     string `json:"repo,omitempty"`
	Package  string `json:"package,omitempty"`
	Revision string `json:"revision,omitempty"`
}

type Downstream struct {
	Repo    string `json:"repo,omitempty"`
	Package string `json:"package,omitempty"`
}

// DownstreamPackageStatus defines the observed state of DownstreamPackage
type DownstreamPackageStatus struct{}

//+kubebuilder:object:root=true

// DownstreamPackageList contains a list of DownstreamPackage
type DownstreamPackageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DownstreamPackage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DownstreamPackage{}, &DownstreamPackageList{})
}
