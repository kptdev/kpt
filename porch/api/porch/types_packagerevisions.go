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

package porch

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PackageRevision
// +k8s:openapi-gen=true
type PackageRevision struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   PackageRevisionSpec
	Status PackageRevisionStatus
}

// PackageRevisionList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PackageRevisionList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []PackageRevision
}

type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft            PackageRevisionLifecycle = "Draft"
	PackageRevisionLifecycleProposed         PackageRevisionLifecycle = "Proposed"
	PackageRevisionLifecyclePublished        PackageRevisionLifecycle = "Published"
	PackageRevisionLifecycleDeletionProposed PackageRevisionLifecycle = "DeletionProposed"
)

type WorkspaceName string

// PackageRevisionSpec defines the desired state of PackageRevision
type PackageRevisionSpec struct {
	// PackageName identifies the package in the repository.
	PackageName string `json:"packageName,omitempty"`

	// RepositoryName is the name of the Repository object containing this package.
	RepositoryName string `json:"repository,omitempty"`

	// WorkspaceName is a short, unique description of the changes contained in this package revision.
	WorkspaceName WorkspaceName `json:"workspaceName,omitempty"`

	// Revision identifies the version of the package.
	Revision string `json:"revision,omitempty"`

	// Parent references a package that provides resources to us
	Parent *ParentReference `json:"parent,omitempty"`

	Lifecycle PackageRevisionLifecycle `json:"lifecycle,omitempty"`

	Tasks []Task `json:"tasks,omitempty"`

	ReadinessGates []ReadinessGate `json:"readinessGates,omitempty"`
}

type ReadinessGate struct {
	ConditionType string `json:"conditionType,omitempty"`
}

// ParentReference is a reference to a parent package
type ParentReference struct {
	// TODO: Should this be a revision or a package?

	// Name is the name of the parent PackageRevision
	Name string `json:"name"`
}

// PackageRevisionStatus defines the observed state of PackageRevision
type PackageRevisionStatus struct {
	// UpstreamLock identifies the upstream data for this package.
	UpstreamLock *UpstreamLock `json:"upstreamLock,omitempty"`

	// PublishedBy is the identity of the user who approved the packagerevision.
	PublishedBy string `json:"publishedBy,omitempty"`

	// PublishedAt is the time when the packagerevision were approved.
	PublishedAt metav1.Time `json:"publishTimestamp,omitempty"`

	// Deployment is true if this is a deployment package (in a deployment repository).
	Deployment bool `json:"deployment,omitempty"`

	Conditions []Condition `json:"conditions,omitempty"`
}

type TaskType string

const (
	TaskTypeInit   TaskType = "init"
	TaskTypeClone  TaskType = "clone"
	TaskTypePatch  TaskType = "patch"
	TaskTypeEdit   TaskType = "edit"
	TaskTypeEval   TaskType = "eval"
	TaskTypeUpdate TaskType = "update"
)

type Task struct {
	Type   TaskType               `json:"type"`
	Init   *PackageInitTaskSpec   `json:"init,omitempty"`
	Clone  *PackageCloneTaskSpec  `json:"clone,omitempty"`
	Patch  *PackagePatchTaskSpec  `json:"patch,omitempty"`
	Edit   *PackageEditTaskSpec   `json:"edit,omitempty"`
	Eval   *FunctionEvalTaskSpec  `json:"eval,omitempty"`
	Update *PackageUpdateTaskSpec `json:"update,omitempty"`
}

type TaskResult struct {
	Task         *Task         `json:"task"`
	RenderStatus *RenderStatus `json:"renderStatus,omitempty"`
}

// RenderStatus represents the result of performing render operation
// on a package resources.
type RenderStatus struct {
	Result ResultList `json:"result,omitempty"`
	Err    string     `json:"error"`
}

// PackageInitTaskSpec defines the package initialization task.
type PackageInitTaskSpec struct {
	// `Subpackage` is a directory path to a subpackage to initialize. If unspecified, the main package will be initialized.
	Subpackage string `json:"subpackage,omitempty"`
	// `Description` is a short description of the package.
	Description string `json:"description,omitempty"`
	// `Keywords` is a list of keywords describing the package.
	Keywords []string `json:"keywords,omitempty"`
	// `Site is a link to page with information about the package.
	Site string `json:"site,omitempty"`
}

type PackageCloneTaskSpec struct {
	// // `Subpackage` is a path to a directory where to clone the upstream package.
	// Subpackage string `json:"subpackage,omitempty"`

	// `Upstream` is the reference to the upstream package to clone.
	Upstream UpstreamPackage `json:"upstreamRef,omitempty"`

	// 	Defines which strategy should be used to update the package. It defaults to 'resource-merge'.
	//  * resource-merge: Perform a structural comparison of the original /
	//    updated resources, and merge the changes into the local package.
	//  * fast-forward: Fail without updating if the local package was modified
	//    since it was fetched.
	//  * force-delete-replace: Wipe all the local changes to the package and replace
	//    it with the remote version.
	Strategy PackageMergeStrategy `json:"strategy,omitempty"`
}

type PackageMergeStrategy string

type PackageUpdateTaskSpec struct {
	// `Upstream` is the reference to the upstream package.
	Upstream UpstreamPackage `json:"upstreamRef,omitempty"`
}

const (
	ResourceMerge      PackageMergeStrategy = "resource-merge"
	FastForward        PackageMergeStrategy = "fast-forward"
	ForceDeleteReplace PackageMergeStrategy = "force-delete-replace"
)

type PackagePatchTaskSpec struct {
	// Patches is a list of individual patch operations.
	Patches []PatchSpec `json:"patches,omitempty"`
}

type PatchType string

const (
	PatchTypeCreateFile PatchType = "CreateFile"
	PatchTypeDeleteFile PatchType = "DeleteFile"
	PatchTypePatchFile  PatchType = "PatchFile"
)

type PatchSpec struct {
	File      string    `json:"file,omitempty"`
	Contents  string    `json:"contents,omitempty"`
	PatchType PatchType `json:"patchType,omitempty"`
}

type PackageEditTaskSpec struct {
	Source *PackageRevisionRef `json:"sourceRef,omitempty"`
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
	UpstreamRef *PackageRevisionRef `json:"upstreamRef,omitempty"`
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

// OciPackage describes a repository compatible with the Open Container Registry standard.
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

	// `Config` specifies the function config, arbitrary KRM resource. Mutually exclusive with ConfigMap.
	Config runtime.RawExtension `json:"config,omitempty"`

	// If enabled, meta resources (i.e. `Kptfile` and `functionConfig`) are included
	// in the input to the function. By default it is disabled.
	IncludeMetaResources bool `json:"includeMetaResources,omitempty"`
	// `EnableNetwork` controls whether the function has access to network. Defaults to `false`.
	EnableNetwork bool `json:"enableNetwork,omitempty"`
	// Match specifies the selection criteria for the function evaluation.
	Match Selector `json:"match,omitempty"`
}

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

// The following types (UpstreamLock, OriginType, and GitLock) are duplicates from the kpt library.
// We are repeating them here to avoid cyclic dependencies, but these duplicate type should be removed when
// https://github.com/GoogleContainerTools/kpt/issues/3297 is resolved.

type OriginType string

// UpstreamLock is a resolved locator for the last fetch of the package.
type UpstreamLock struct {
	// Type is the type of origin.
	Type OriginType `json:"type,omitempty"`

	// Git is the resolved locator for a package on Git.
	Git *GitLock `json:"git,omitempty"`
}

// GitLock is the resolved locator for a package on Git.
type GitLock struct {
	// Repo is the git repository that was fetched.
	// e.g. 'https://github.com/kubernetes/examples.git'
	Repo string `json:"repo,omitempty"`

	// Directory is the sub directory of the git repository that was fetched.
	// e.g. 'staging/cockroachdb'
	Directory string `json:"directory,omitempty"`

	// Ref can be a Git branch, tag, or a commit SHA-1 that was fetched.
	// e.g. 'master'
	Ref string `json:"ref,omitempty"`

	// Commit is the SHA-1 for the last fetch of the package.
	// This is set by kpt for bookkeeping purposes.
	Commit string `json:"commit,omitempty"`
}

type Condition struct {
	Type string `json:"type"`

	Status ConditionStatus `json:"status"`

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`
}

type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

const (
	// Deprecated: prefer ResultListGVK
	ResultListKind = "FunctionResultList"
	// Deprecated: prefer ResultListGVK
	ResultListGroup = "kpt.dev"
	// Deprecated: prefer ResultListGVK
	ResultListVersion = "v1"
	// Deprecated: prefer ResultListGVK
	ResultListAPIVersion = ResultListGroup + "/" + ResultListVersion
)

// ResultList contains aggregated results from multiple functions
type ResultList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// ExitCode is the exit code of kpt command
	ExitCode int `json:"exitCode"`
	// Items contain a list of function result
	Items []*Result `json:"items,omitempty"`
}

// Result contains the structured result from an individual function
type Result struct {
	// Image is the full name of the image that generates this result
	// Image and Exec are mutually exclusive
	Image string `json:"image,omitempty"`
	// ExecPath is the the absolute os-specific path to the executable file
	// If user provides an executable file with commands, ExecPath should
	// contain the entire input string.
	ExecPath string `json:"exec,omitempty"`
	// TODO(droot): This is required for making structured results subpackage aware.
	// Enable this once test harness supports filepath based assertions.
	// Pkg is OS specific Absolute path to the package.
	// Pkg string `yaml:"pkg,omitempty"`
	// Stderr is the content in function stderr
	Stderr string `json:"stderr,omitempty"`
	// ExitCode is the exit code from running the function
	ExitCode int `json:"exitCode"`
	// Results is the list of results for the function
	Results []ResultItem `json:"results,omitempty"`
}

// ResultItem defines a validation result
type ResultItem struct {
	// Message is a human readable message. This field is required.
	Message string `json:"message,omitempty"`

	// Severity is the severity of this result
	Severity string `json:"severity,omitempty"`

	// ResourceRef is a reference to a resource.
	// Required fields: apiVersion, kind, name.
	ResourceRef *ResourceIdentifier `json:"resourceRef,omitempty"`

	// Field is a reference to the field in a resource this result refers to
	Field *Field `json:"field,omitempty"`

	// File references a file containing the resource this result refers to
	File *File `json:"file,omitempty"`

	// Tags is an unstructured key value map stored with a result that may be set
	// by external tools to store and retrieve arbitrary metadata
	Tags map[string]string `json:"tags,omitempty"`
}

// File references a file containing a resource
type File struct {
	// Path is relative path to the file containing the resource.
	// This field is required.
	Path string `json:"path,omitempty"`

	// Index is the index into the file containing the resource
	// (i.e. if there are multiple resources in a single file)
	Index int `json:"index,omitempty"`
}

// Field references a field in a resource
type Field struct {
	// Path is the field path. This field is required.
	Path string `json:"path,omitempty"`

	// CurrentValue is the current field value
	CurrentValue string `json:"currentValue,omitempty"`

	// ProposedValue is the proposed value of the field to fix an issue.
	ProposedValue string `json:"proposedValue,omitempty"`
}

// ResourceIdentifier contains the information needed to uniquely
// identify a resource in a cluster.
type ResourceIdentifier struct {
	metav1.TypeMeta `json:",inline"`
	NameMeta        `json:",inline"`
}

// NameMeta contains name information.
type NameMeta struct {
	// Name is the metadata.name field of a Resource
	Name string `json:"name,omitempty"`
	// Namespace is the metadata.namespace field of a Resource
	Namespace string `json:"namespace,omitempty"`
}
