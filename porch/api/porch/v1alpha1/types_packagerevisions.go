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

// +genclient
// +genclient:method=UpdateApproval,verb=update,subresource=approval,input=github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1.PackageRevision,result=github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1.PackageRevision
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PackageRevision
// +k8s:openapi-gen=true
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageRevisionSpec   `json:"spec,omitempty"`
	Status PackageRevisionStatus `json:"status,omitempty"`
}

// PackageRevisionList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PackageRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PackageRevision `json:"items"`
}

// PackageRevisionSpec defines the desired state of PackageRevision
type PackageRevisionSpec struct {
	PackageName string `json:"packageName,omitempty"`

	Revision string `json:"revision,omitempty"`

	RepositoryName string `json:"repository,omitempty"`

	Tasks []Task `json:"tasks,omitempty"`
}

// PackageRevisionStatus defines the observed state of PackageRevision
type PackageRevisionStatus struct {
}

type TaskType string

const (
	TaskTypeClone TaskType = "clone"
	TaskTypePatch TaskType = "patch"
	TaskTypeEval  TaskType = "eval"
)

type Task struct {
	Type  TaskType              `json:"type"`
	Clone *PackageCloneTaskSpec `json:"clone,omitempty"`
	Patch *PackagePatchTaskSpec `json:"patch,omitempty"`
	Eval  *FunctionEvalTaskSpec `json:"eval,omitempty"`
}

type PackageCloneTaskSpec struct {
	// // `Subpackage` is a path to a directory where to clone the upstream package.
	// Subpackage string `json:"subpackage,omitempty"`

	// `Upstream` is the reference to the upstream package to clone.
	Upstream UpstreamPackage `json:"upstreamRef,omitempty"`

	// // 	Defines which strategy should be used to update the package. It defaults to 'resource-merge'.
	// 	//     * resource-merge: Perform a structural comparison of the original /
	// 	//       updated resources, and merge the changes into the local package.
	// 	//     * fast-forward: Fail without updating if the local package was modified
	// 	//       since it was fetched.
	// 	//     * force-delete-replace: Wipe all the local changes to the package and replace
	// 	//       it with the remote version.
	// 	Strategy PackageMergeStrategy `json:"strategy,omitempty"`
}

type PackagePatchTaskSpec struct {
	// TODO: We're going to need something better here to actually represent or reference the patch
	Patches []string `json:"patches,omitempty"`
}

type RepositoryType string

const (
	RepositoryTypeGit RepositoryType = "git"
	RepositoryTypeOCI RepositoryType = "oci"
)

// UpstreamRepository repository may be specified directly or by referencing another Repository resource.
type UpstreamPackage struct {
	// Type of the repository (i.e. git, OCI). If empty, `upstreamRef` will be used.
	Type RepositoryType `json:"type,omitempty"`

	// Git upstream package specification. Required if `type` is `git`. Must be unspecified if `type` is not `git`.
	Git *GitPackage `json:"git,omitempty"`

	// OCI upstream package specification. Required if `type` is `oci`. Must be unspecified if `type` is not `oci`.
	Oci *OciPackage `json:"oci,omitempty"`

	// UpstreamRef is the reference to the package from a registered repository rather than external package.
	UpstreamRef PackageRevisionRef `json:"upstreamRef,omitempty"`
}

type GitPackage struct {
	// Address of the Git repository, for example:
	//   `https://github.com/GoogleCloudPlatform/blueprints.git`
	Repo string `json:"repo"`

	// `Ref` is the git ref containing the package. Ref can be a branch, tag, or commit SHA.
	Ref string `json:"ref"`

	// Directory within the Git repository where the packages are stored. A subdirectory of this directory containing a Kptfile is considered a package.
	Directory string `json:"directory"`

	// Reference to secret containing authentication credentials. Optional.
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

type SecretRef struct {
	// Name of the secret. The secret is expected to be located in the same namespace as the resource containing the reference.
	Name string `json:"name"`
}

// OciPackage describes a repository compatible with the Open Coutainer Registry standard.
type OciPackage struct {
	// Image is the address of an OCI image.
	Image string `json:"image"`
}

// PackageRevisionRef is a reference to a package revision.
type PackageRevisionRef struct {
	// `Name` is the name of the referenced PackageRevision resource.
	Name string `json:"name"`
}

// RepositoryRef identifies a reference to a Repository resource.
type RepositoryRef struct {
	// Name of the Repository resource referenced.
	Name string `json:"name"`
}

type FunctionEvalTaskSpec struct {
	// `Subpackage` is a directory path to a subpackage in which to evaluate the function.
	Subpackage string `json:"subpackage,omitempty"`
	// `Image` specifies the function image, such as `gcr.io/kpt-fn/gatekeeper:v0.2`. Use of `Image` is mutually exclusive with `FunctionRef`.
	Image string `json:"image,omitempty"`
	// `FunctionRef` specifies the function by reference to a Function resource. Mutually exclusive with `Image`.
	FunctionRef *FunctionRef `json:"functionRef,omitempty"`
	// `ConfigMap` specifies the function config (https://kpt.dev/reference/cli/fn/eval/). Mutually exclusive with Config.
	ConfigMap map[string]string `json:"configMap,omitempty"`

	// TODO: openapi generation doesn't work for Unstructured. Use runtime.RawExtension ???
	// // `Config` specifies the function config, arbitrary KRM resource. Mutually exclusive with ConfigMap.
	// Config unstructured.Unstructured `json:"config,omitempty"`

	// If enabled, meta resources (i.e. `Kptfile` and `functionConfig`) are included
	// in the input to the function. By default it is disabled.
	IncludeMetaResources bool `json:"includeMetaResources,omitempty"`
	// `EnableNetwork` controls whether the function has access to network. Defaults to `false`.
	EnableNetwork bool `json:"enableNetwork,omitempty"`
	// Match specifies the selection criteria for the function evaluation.
	// Corresponds to `kpt fn eval --match-???` flgs (https://kpt.dev/reference/cli/fn/eval/).
	Match Selector `json:"match,omitempty"`
}

// Selector corresponds to the `--match-???` set of flags of the `kpt fn eval` command:
// See https://kpt.dev/reference/cli/fn/eval/ for additional information.
type Selector struct {
	// APIVersion of the target resources
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind of the target resources
	Kind string `json:"kind,omitempty"`
	// Name of the target resources
	Name string `json:"name,omitempty"`
	// Namespace of the target resources
	Namespace string `json:"namespace,omitempty"`
}
