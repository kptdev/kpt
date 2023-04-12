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
//+kubebuilder:resource:path=repositories,singular=repository
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Content",type=string,JSONPath=`.spec.content`
//+kubebuilder:printcolumn:name="Deployment",type=boolean,JSONPath=`.spec.deployment`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
//+kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.spec['git','oci']['repo','registry']`

// Repository
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

type RepositoryType string

const (
	RepositoryTypeGit RepositoryType = "git"
	RepositoryTypeOCI RepositoryType = "oci"
)

type RepositoryContent string

const (
	RepositoryContentFunction RepositoryContent = "Function"
	RepositoryContentPackage  RepositoryContent = "Package"
)

// RepositorySpec defines the desired state of Repository
//
// Notes:
//   - deployment repository - in KRM API ConfigSync would be configured directly? (or via this API)
type RepositorySpec struct {
	// User-friendly description of the repository
	Description string `json:"description,omitempty"`
	// The repository is a deployment repository; final packages in this repository are deployment ready.
	Deployment bool `json:"deployment,omitempty"`
	// Type of the repository (i.e. git, OCI)
	Type RepositoryType `json:"type,omitempty"`
	// Content stored in the repository (i.e. Function, Package - the literal values correspond to the API resource names).
	// TODO: support repository with mixed content?
	Content RepositoryContent `json:"content,omitempty"`
	// Git repository details. Required if `type` is `git`. Ignored if `type` is not `git`.
	Git *GitRepository `json:"git,omitempty"`
	// OCI repository details. Required if `type` is `oci`. Ignored if `type` is not `oci`.
	Oci *OciRepository `json:"oci,omitempty"`
	// Upstream is the default upstream repository for packages in this
	// repository. Specifying it per repository allows simpler UX when
	// creating packages.
	Upstream *UpstreamRepository `json:"upstream,omitempty"`

	// `Mutators` specifies list of functions to be added to the list of package's mutators on changes to the packages in the repository to ensure the packages meet constraints
	// enforced by the mutators associated with the repository.
	// Based on the Kubernetest Admission Controllers (https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/). The functions will be evaluated
	// in the order specified in the list.
	Mutators []FunctionEval `json:"mutators,omitempty"`

	// `Validators` specifies list of functions to be added to the list of package's validators on changes to the packages in the repository to ensure the packages meet constraints
	// enforced by the validators associated with the repository.
	// Based on the Kubernetest Admission Controllers (https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/). The functions will be evaluated
	// in the order specified in the list.
	Validators []FunctionEval `json:"validators,omitempty"`
}

// GitRepository describes a Git repository.
// TODO: authentication methods
type GitRepository struct {
	// Address of the Git repository, for example:
	//   `https://github.com/GoogleCloudPlatform/blueprints.git`
	Repo string `json:"repo"`
	// Name of the branch containing the packages. Finalized packages will be committed to this branch (if the repository allows write access). If unspecified, defaults to "main".
	Branch string `json:"branch,omitempty"`
	// CreateBranch specifies if Porch should create the package branch if it doesn't exist.
	CreateBranch bool `json:"createBranch,omitempty"`
	// Directory within the Git repository where the packages are stored. A subdirectory of this directory containing a Kptfile is considered a package. If unspecified, defaults to root directory.
	Directory string `json:"directory,omitempty"`
	// Reference to secret containing authentication credentials.
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

// OciRepository describes a repository compatible with the Open Container Registry standard.
// TODO: allow sub-selection of the registry, i.e. filter by tags, ...?
// TODO: authentication types?
type OciRepository struct {
	// Registry is the address of the OCI registry
	Registry string `json:"registry"`
	// Reference to secret containing authentication credentials.
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

// UpstreamRepository repository may be specified directly or by referencing another Repository resource.
type UpstreamRepository struct {
	// Type of the repository (i.e. git, OCI). If empty, repositoryRef will be used.
	Type RepositoryType `json:"type,omitempty"`
	// Git repository details. Required if `type` is `git`. Must be unspecified if `type` is not `git`.
	Git *GitRepository `json:"git,omitempty"`
	// OCI repository details. Required if `type` is `oci`. Must be unspecified if `type` is not `oci`.
	Oci *OciRepository `json:"oci,omitempty"`
	// RepositoryRef contains a reference to an existing Repository resource to be used as the default upstream repository.
	RepositoryRef *RepositoryRef `json:"repositoryRef,omitempty"`
}

// RepositoryRef identifies a reference to a Repository resource.
type RepositoryRef struct {
	// Name of the Repository resource referenced.
	Name string `json:"name"`
}

type SecretRef struct {
	// Name of the secret. The secret is expected to be located in the same namespace as the resource containing the reference.
	Name string `json:"name"`
}

type FunctionEval struct {
	// `Image` specifies the function image, such as `gcr.io/kpt-fn/gatekeeper:v0.2`. Use of `Image` is mutually exclusive with `FunctionRef`.
	Image string `json:"image,omitempty"`
	// `FunctionRef` specifies the function by reference to a Function resource. Mutually exclusive with `Image`.
	FunctionRef *FunctionRef `json:"functionRef,omitempty"`
	// `ConfigMap` specifies the function config (https://kpt.dev/reference/cli/fn/eval/).
	ConfigMap map[string]string `json:"configMap,omitempty"`
}

// `FunctionRef` is a reference to a `Function` resource.
type FunctionRef struct {
	// `Name` is the name of the `Function` resource referenced. The resource is expected to be within the same namespace.
	Name string `json:"name"`
}

const (
	// Type of the Repository condition.
	RepositoryReady = "Ready"

	// Reason for the condition is error.
	ReasonError = "Error"
	// Reason for the condition is the repository is ready.
	ReasonReady = "Ready"
)

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true

// RepositoryList contains a list of Repo
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}
