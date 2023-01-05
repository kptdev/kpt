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

// RolloutSpec defines the desired state of Rollout
type RolloutSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Description is a user friendly description of this Rollout.
	Description string `json:"description,omitempty"`

	// Packages source for this Rollout.
	Packages PackagesConfig `json:"packages"`

	// Targets specifies the clusters that will receive the KRM config packages.
	Targets ClusterTargetSelector `json:"targets,omitempty"`

	// PackageToTargetMatcher specifies the clusters that will receive a specific package.
	PackageToTargetMatcher PackageToClusterMatcher `json:"packageToTargetMatcher"`
	// Strategy specifies the rollout strategy to use for this rollout.
	Strategy RolloutStrategy `json:"strategy"`
}

type ClusterTargetSelector struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// ClusterReference contains the identify information
// need to refer a cluster.
type ClusterRef struct {
	Name string `json:"name"`
}

const (
	GitHub PackageSourceType = "GitHub"
)

// +kubebuilder:validation:Enum=GitHub
type PackageSourceType string

// PackagesConfig defines the packages the Rollout should deploy.
type PackagesConfig struct {
	SourceType PackageSourceType `json:"sourceType"`

	GitHub GitHubSource `json:"github"`
}

// GitHubSource defines the packages source in Git.
type GitHubSource struct {
	Selector GitHubSelector `json:"selector"`
}

// GitHubSelector defines the selector to apply to Git.
type GitHubSelector struct {
	Org       string          `json:"org"`
	Repo      string          `json:"repo"`
	Directory string          `json:"directory,omitempty"`
	Revision  string          `json:"revision"`
	SecretRef SecretReference `json:"secretRef,omitempty"`
}

// SecretReference contains the reference to the secret
type SecretReference struct {
	// Name represents the secret name
	Name string `json:"name,omitempty"`
}

// +kubebuilder:validation:Enum=CEL
type MatcherType string

type PackageToClusterMatcher struct {
	Type            MatcherType `json:"type"`
	MatchExpression string      `json:"matchExpression"`
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

// StrategyProgressive allows staged rollouts
// where the entire rollout will progress through different stages (aka steps, phases or waves).
type StrategyProgressive struct{}

type RolloutStrategy struct {
	Type          StrategyType           `json:"type"`
	AllAtOnce     *StrategyAllAtOnce     `json:"allAtOnce,omitempty"`
	RollingUpdate *StrategyRollingUpdate `json:"rollingUpdate,omitempty"`
	Progressive   *StrategyProgressive   `json:"progressive,omitempty"`
}

// RolloutStatus defines the observed state of Rollout
type RolloutStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions describes the reconciliation state of the object.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

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
