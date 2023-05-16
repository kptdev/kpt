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
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:unservedversion
//
// PackageVariantSet represents an upstream package revision and a way to
// target specific downstream repositories where a variant of the upstream
// package should be created.
type PackageVariantSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageVariantSetSpec   `json:"spec,omitempty"`
	Status PackageVariantSetStatus `json:"status,omitempty"`
}

func (o *PackageVariantSet) GetSpec() *PackageVariantSetSpec {
	if o == nil {
		return nil
	}
	return &o.Spec
}

// PackageVariantSetSpec defines the desired state of PackageVariantSet
type PackageVariantSetSpec struct {
	Upstream *Upstream `json:"upstream,omitempty"`
	Targets  []Target  `json:"targets,omitempty"`

	AdoptionPolicy pkgvarapi.AdoptionPolicy `json:"adoptionPolicy,omitempty"`
	DeletionPolicy pkgvarapi.DeletionPolicy `json:"deletionPolicy,omitempty"`

	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Upstream struct {
	Package *Package `json:"package,omitempty"`

	Revision string `json:"revision,omitempty"`

	Tag string `json:"ref,omitempty"`
}

type Target struct {
	// option 1: an explicit repo/package name pair
	Package *Package `json:"package,omitempty"`

	// option 2: a label selector against a set of repositories
	Repositories *metav1.LabelSelector `json:"repositories,omitempty"`

	// option 3: a selector against a set of arbitrary objects
	Objects *ObjectSelector `json:"objects,omitempty"`

	// For options 2 and 3, PackageName specifies how to create the name of the
	// package variant
	PackageName *PackageName `json:"packageName,omitempty"`
}

type Package struct {
	Repo string `json:"repo,omitempty"`
	Name string `json:"name,omitempty"`
}

type ObjectSelector struct {
	Selectors []Selector `json:"selectors,omitempty"`

	RepoName *ValueOrFromField `json:"repoName,omitempty"`
}

type Selector struct {
	// APIVersion of the target resources
	APIVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	// Kind of the target resources
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`
	// Name of the target resources
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// Namespace of the target resources
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	// Labels on the target resources
	Labels *metav1.LabelSelector `yaml:"labelSelector,omitempty" json:"labelSelector,omitempty"`
	// Annotations on the target resources
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

func (s *Selector) ToKptfileSelector() kptfilev1.Selector {
	var labels map[string]string
	if s.Labels != nil {
		labels = s.Labels.MatchLabels
	}
	return kptfilev1.Selector{
		APIVersion:  s.APIVersion,
		Kind:        s.Kind,
		Name:        s.Name,
		Namespace:   s.Namespace,
		Labels:      labels,
		Annotations: s.Annotations,
	}
}

type PackageName struct {
	Name *ValueOrFromField `json:"baseName,omitempty"`

	NameSuffix *ValueOrFromField `json:"nameSuffix,omitempty"`

	NamePrefix *ValueOrFromField `json:"namePrefix,omitempty"`
}

type ValueOrFromField struct {
	Value     string `json:"value,omitempty"`
	FromField string `json:"fromField,omitempty"`
}

// PackageVariantSetStatus defines the observed state of PackageVariantSet
type PackageVariantSetStatus struct {
	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true

// PackageVariantSetList contains a list of PackageVariantSet
type PackageVariantSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PackageVariantSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PackageVariantSet{}, &PackageVariantSetList{})
}
