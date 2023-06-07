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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RolloutSpec defines the desired state of Rollout
type RolloutSpec struct {
	// Description is a user friendly description of this Rollout.
	Description string `json:"description,omitempty"`

	// Clusters specifies the source for discovering the clusters.
	Clusters ClusterDiscovery `json:"clusters"`

	// Packages source for this Rollout.
	Packages PackagesConfig `json:"packages"`

	// Targets specifies the clusters that will receive the KRM config packages.
	Targets ClusterTargetSelector `json:"targets,omitempty"`

	// PackageToTargetMatcher specifies the clusters that will receive a specific package.
	PackageToTargetMatcher PackageToClusterMatcher `json:"packageToTargetMatcher"`

	// SyncTemplate defines the type and attributes for the RSync object used to syncing the packages.
	SyncTemplate *SyncTemplate `json:"syncTemplate,omitempty"`

	// Strategy specifies the rollout strategy to use for this rollout.
	Strategy RolloutStrategy `json:"strategy"`
}

type ClusterTargetSelector struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// ClusterReference contains the identify information
// need to refer a cluster.
type ClusterRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
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
	return r.APIVersion
}

// different types of cluster sources
const (
	KCC         ClusterSourceType = "KCC"
	GCPFleet    ClusterSourceType = "GCPFleet"
	KindCluster ClusterSourceType = "Kind"
)

// +kubebuilder:validation:Enum=KCC;GCPFleet;Kind
type ClusterSourceType string

// ClusterDiscovery represents configuration needed to discover clusters.
type ClusterDiscovery struct {
	SourceType ClusterSourceType      `json:"sourceType"`
	GCPFleet   *ClusterSourceGCPFleet `json:"gcpFleet,omitempty"`
	Kind       *ClusterSourceKind     `json:"kind,omitempty"`
}

// ClusterSourceGCPFleet represents configuration needed to discover gcp fleet clusters.
type ClusterSourceGCPFleet struct {
	ProjectIds []string `json:"projectIds"`
}

// ClusterSourceKind contains configuration needed to discover kind clusters.
type ClusterSourceKind struct {
	// Namespace where configmaps corresponding to kind clusters live
	// defaults to `kind-clusters`
	Namespace string `json:"namespace,omitempty"`
}

const (
	GitHub PackageSourceType = "GitHub"
	GitLab PackageSourceType = "GitLab"
	OCI    PackageSourceType = "OCI"
)

// +kubebuilder:validation:Enum=GitHub;GitLab;OCI
type PackageSourceType string

// PackagesConfig defines the packages the Rollout should deploy.
type PackagesConfig struct {
	SourceType PackageSourceType `json:"sourceType"`

	// TODO(droot): Change Github and Gitlab to pointers because
	// One of the the following will be non-nil to follow OneOf semantics.
	GitHub    GitHubSource `json:"github,omitempty"`
	GitLab    GitLabSource `json:"gitlab,omitempty"`
	OciSource *OCISource   `json:"oci,omitempty"`
}

// GitHubSource defines the packages source in GitHub.
type GitHubSource struct {
	Selector GitHubSelector `json:"selector"`
}

// GitHubSelector defines the selector to apply to packages in GitHub.
type GitHubSelector struct {
	Org       string          `json:"org"`
	Repo      string          `json:"repo"`
	Directory string          `json:"directory,omitempty"`
	Revision  string          `json:"revision,omitempty"`
	Branch    string          `json:"branch,omitempty"`
	SecretRef SecretReference `json:"secretRef,omitempty"`
}

// GitLabSource defines the packages source in GitLab.
type GitLabSource struct {
	// SecretReference is the reference to a kubernetes secret
	// that contains GitLab access token
	SecretRef SecretReference `json:"secretRef,omitempty"`
	// Selector defines the package selector in GitLab.
	Selector GitLabSelector `json:"selector"`
}

// GitLabSelector defines how to select packages in GitLab.
type GitLabSelector struct {
	// ProjectID is the numerical identifier of the GitLab project
	// It will not be specified if selection involves multiple projects
	ProjectID string `json:"projectID,omitempty"`
	// Directory refers to the subdirectory path in the project
	Directory string `json:"directory,omitempty"`
	// Revision refers to the branch, tag of the GitLab repo
	Revision string `json:"revision,omitempty"`
	// Branch refers to the branch
	Branch string `json:"branch,omitempty"`
}

// OCISource defines configuration to discover OCI packages.
type OCISource struct {
	// Selector contains config to select OCI packages
	Selector OCISelector `json:"selector"`
}

// Selector represent info on how to select OCI packages.
type OCISelector struct {
	// image is the OCI image repository URL for the package to sync from.
	// e.g. `LOCATION-docker.pkg.dev/PROJECT_ID/REPOSITORY_NAME/PACKAGE_NAME`.
	// The image can be pulled by TAG or by DIGEST if it is specified in PACKAGE_NAME.
	// - Pull by tag: `LOCATION-docker.pkg.dev/PROJECT_ID/REPOSITORY_NAME/PACKAGE_NAME:TAG`.
	// - Pull by digest: `LOCATION-docker.pkg.dev/PROJECT_ID/REPOSITORY_NAME/PACKAGE_NAME@sha256:DIGEST`.
	// If neither TAG nor DIGEST is specified, it pulls with the `latest` tag by default.
	// Required
	Image string `json:"image"`

	// dir is the absolute path of the directory that contains
	// the local resources.  Default: the root directory of the image.
	// Note (droot): We will extend `Dir` to express variants of a package at some point.
	// +optional
	Dir string `json:"dir,omitempty"`
}

// SecretReference contains the reference to the secret
type SecretReference struct {
	// Name represents the secret name
	Name string `json:"name,omitempty"`
}

// different types of sync templates.
const (
	TemplateTypeRootSync SyncTemplateType = "RootSync"
	TemplateTypeRepoSync SyncTemplateType = "RepoSync"
)

// +kubebuilder:validation:Enum=RootSync;RepoSync
type SyncTemplateType string

// SyncTemplate defines the configuration for RSync templates.
type SyncTemplate struct {
	Type     SyncTemplateType  `json:"type"`
	RootSync *RootSyncTemplate `json:"rootSync,omitempty"`
	RepoSync *RepoSyncTemplate `json:"repoSync,omitempty"`
}

// RootSyncTemplate represent the sync template for RootSync.
type RootSyncTemplate struct {
	SourceFormat string    `json:"sourceFormat,omitempty"`
	Git          *GitInfo  `json:"git,omitempty"`
	Oci          *OciInfo  `json:"oci,omitempty"`
	Metadata     *Metadata `json:"metadata,omitempty"`
}

// RepoSyncTemplate represent the sync template for RepoSync.
type RepoSyncTemplate struct {
	SourceFormat string    `json:"sourceFormat,omitempty"`
	Git          *GitInfo  `json:"git,omitempty"`
	Oci          *OciInfo  `json:"oci,omitempty"`
	Metadata     *Metadata `json:"metadata,omitempty"`
}

// +kubebuilder:validation:Enum=AllClusters;Custom
type MatcherType string

const (
	MatchAllClusters MatcherType = "AllClusters"
	CustomMatcher    MatcherType = "Custom"
)

type PackageToClusterMatcher struct {
	Type            MatcherType `json:"type"`
	MatchExpression string      `json:"matchExpression,omitempty"`
}

// +kubebuilder:validation:Enum=AllAtOnce;RollingUpdate;Progressive
type StrategyType string

const (
	AllAtOnce     StrategyType = "AllAtOnce"
	RollingUpdate StrategyType = "RollingUpdate"
	Progressive   StrategyType = "Progressive"
)

type StrategyAllAtOnce struct{}

type StrategyRollingUpdate struct {
	MaxConcurrent int64 `json:"maxConcurrent"`
}

// StrategyProgressive defines the progressive rollout strategy to use.
type StrategyProgressive struct {
	// Name of the ProgressiveRolloutStrategy to use.
	Name string `json:"name"`

	// Namespace of the ProgressiveRolloutStrategy to use.
	Namespace string `json:"namespace"`

	// PauseAfterWave represents the highest wave the strategy will deploy.
	PauseAfterWave PauseAfterWave `json:"pauseAfterWave,omitempty"`
}

type PauseAfterWave struct {
	// WaveName represents name of the wave defined in the ProgressiveRolloutStrategy.
	WaveName string `json:"waveName"`
}

type RolloutStrategy struct {
	Type          StrategyType           `json:"type"`
	AllAtOnce     *StrategyAllAtOnce     `json:"allAtOnce,omitempty"`
	RollingUpdate *StrategyRollingUpdate `json:"rollingUpdate,omitempty"`
	Progressive   *StrategyProgressive   `json:"progressive,omitempty"`
}

// RolloutStatus defines the observed state of Rollout
type RolloutStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	Overall      string       `json:"overall,omitempty"`
	WaveStatuses []WaveStatus `json:"waveStatuses,omitempty"`

	ClusterStatuses []ClusterStatus `json:"clusterStatuses,omitempty"`
}

type WaveStatus struct {
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	Paused          bool            `json:"paused,omitempty"`
	ClusterStatuses []ClusterStatus `json:"clusterStatuses,omitempty"`
}

type ClusterStatus struct {
	Name          string        `json:"name"`
	PackageStatus PackageStatus `json:"packageStatus"`
}

type PackageStatus struct {
	PackageID  string `json:"packageId"`
	SyncStatus string `json:"syncStatus"`
	Status     string `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Rollout is the Schema for the rollouts API
type Rollout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RolloutSpec   `json:"spec,omitempty"`
	Status RolloutStatus `json:"status,omitempty"`
}

func (rollout *Rollout) GetSyncTemplateType() SyncTemplateType {
	if rollout.Spec.SyncTemplate == nil {
		return TemplateTypeRootSync
	}
	return rollout.Spec.SyncTemplate.Type
}

//+kubebuilder:object:root=true

// RolloutList contains a list of Rollout
type RolloutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rollout `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Rollout{}, &RolloutList{})
}
