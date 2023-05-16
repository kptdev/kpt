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

package v1alpha2

import (
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
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
	Upstream *pkgvarapi.Upstream `json:"upstream,omitempty"`
	Targets  []Target            `json:"targets,omitempty"`
}

type Target struct {
	// Exactly one of Repositories, RepositorySeletor, and ObjectSelector must be
	// populated
	// option 1: an explicit repositories and package names
	Repositories []RepositoryTarget `json:"repositories,omitempty"`

	// option 2: a label selector against a set of repositories
	RepositorySelector *metav1.LabelSelector `json:"repositorySelector,omitempty"`

	// option 3: a selector against a set of arbitrary objects
	ObjectSelector *ObjectSelector `json:"objectSelector,omitempty"`

	// Template specifies how to generate a PackageVariant from a target
	Template *PackageVariantTemplate `json:"template,omitempty"`
}

type RepositoryTarget struct {
	// Name contains the name of the Repository resource, which must be in
	// the same namespace as the PackageVariantSet resource.
	// +required
	Name string `json:"name"`

	// PackageNames contains names to use for package instances in this repository;
	// that is, the same upstream will be instantiated multiple times using these names.
	// +optional
	PackageNames []string `json:"packageNames,omitempty"`
}

type ObjectSelector struct {
	metav1.LabelSelector `json:",inline"`

	// APIVersion of the target resources
	APIVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`

	// Kind of the target resources
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`

	// Name of the target resource
	// +optional
	Name *string `yaml:"name,omitempty" json:"name,omitempty"`

	// Note: while v1alpha1 had Namespace, that is not allowed; the namespace
	// must match the namespace of the PackageVariantSet resource
}

type PackageVariantTemplate struct {
	// Downstream allows overriding the default downstream package and repository name
	// +optional
	Downstream *DownstreamTemplate `json:"downstream,omitempty"`

	// AdoptionPolicy allows overriding the PackageVariant adoption policy
	// +optional
	AdoptionPolicy *pkgvarapi.AdoptionPolicy `json:"adoptionPolicy,omitempty"`

	// DeletionPolicy allows overriding the PackageVariant deletion policy
	// +optional
	DeletionPolicy *pkgvarapi.DeletionPolicy `json:"deletionPolicy,omitempty"`

	// Labels allows specifying the spec.Labels field of the generated PackageVariant
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// LabelsExprs allows specifying the spec.Labels field of the generated PackageVariant
	// using CEL to dynamically create the keys and values. Entries in this field take precedent over
	// those with the same keys that are present in Labels.
	// +optional
	LabelExprs []MapExpr `json:"labelExprs,omitempty"`

	// Annotations allows specifying the spec.Annotations field of the generated PackageVariant
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// AnnotationsExprs allows specifying the spec.Annotations field of the generated PackageVariant
	// using CEL to dynamically create the keys and values. Entries in this field take precedent over
	// those with the same keys that are present in Annotations.
	// +optional
	AnnotationExprs []MapExpr `json:"annotationExprs,omitempty"`

	// PackageContext allows specifying the spec.PackageContext field of the generated PackageVariant
	// +optional
	PackageContext *PackageContextTemplate `json:"packageContext,omitempty"`

	// Pipeline allows specifying the spec.Pipeline field of the generated PackageVariant
	// +optional
	Pipeline *PipelineTemplate `json:"pipeline,omitempty"`

	// Injectors allows specifying the spec.Injectors field of the generated PackageVariant
	// +optional
	Injectors []InjectionSelectorTemplate `json:"injectors,omitempty"`
}

// DownstreamTemplate is used to calculate the downstream field of the resulting
// package variants. Only one of Repo and RepoExpr may be specified;
// similarly only one of Package and PackageExpr may be specified.
type DownstreamTemplate struct {
	Repo        *string `json:"repo,omitempty"`
	Package     *string `json:"package,omitempty"`
	RepoExpr    *string `json:"repoExpr,omitempty"`
	PackageExpr *string `json:"packageExpr,omitempty"`
}

// PackageContextTemplate is used to calculate the packageContext field of the
// resulting package variants. The plain fields and Exprs fields will be
// merged, with the Exprs fields taking precedence.
type PackageContextTemplate struct {
	Data           map[string]string `json:"data,omitempty"`
	RemoveKeys     []string          `json:"removeKeys,omitempty"`
	DataExprs      []MapExpr         `json:"dataExprs,omitempty"`
	RemoveKeyExprs []string          `json:"removeKeyExprs,omitempty"`
}

// InjectionSelectorTemplate is used to calculate the injectors field of the
// resulting package variants. Exactly one of the Name and NameExpr fields must
// be specified. The other fields are optional.
type InjectionSelectorTemplate struct {
	Group   *string `json:"group,omitempty"`
	Version *string `json:"version,omitempty"`
	Kind    *string `json:"kind,omitempty"`
	Name    *string `json:"name,omitempty"`

	NameExpr *string `json:"nameExpr,omitempty"`
}

// MapExpr is used for various fields to calculate map entries. Only one of
// Key and KeyExpr may be specified; similarly only on of Value and ValueExpr
// may be specified.
type MapExpr struct {
	Key       *string `json:"key,omitempty"`
	Value     *string `json:"value,omitempty"`
	KeyExpr   *string `json:"keyExpr,omitempty"`
	ValueExpr *string `json:"valueExpr,omitempty"`
}

// PipelineTemplate is used to calculate the pipeline field of the resulting
// package variants.
type PipelineTemplate struct {
	// Validators is used to caculate the pipeline.validators field of the
	// resulting package variants.
	// +optional
	Validators []FunctionTemplate `json:"validators,omitempty"`

	// Mutators is used to caculate the pipeline.mutators field of the
	// resulting package variants.
	// +optional
	Mutators []FunctionTemplate `json:"mutators,omitempty"`
}

// FunctionTemplate is used in generating KRM function pipeline entries; that
// is, it is used to generate Kptfile Function objects.
type FunctionTemplate struct {
	kptfilev1.Function `json:",inline"`

	// ConfigMapExprs allows use of CEL to dynamically create the keys and values in the
	// function config ConfigMap. Entries in this field take precedent over those with
	// the same keys that are present in ConfigMap.
	// +optional
	ConfigMapExprs []MapExpr `json:"configMapExprs,omitempty"`
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
