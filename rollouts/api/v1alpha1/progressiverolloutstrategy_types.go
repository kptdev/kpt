/*
Copyright 2023.

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

// ProgressiveRolloutStrategySpec defines the desired state of ProgressiveRolloutStrategy
type ProgressiveRolloutStrategySpec struct {
	// Description is a user friendly description of this rollout strategy.
	Description string `json:"description,omitempty"`

	// Waves defines an order set of waves of rolling updates.
	Waves []Wave `json:"waves"`
}

// Wave represents a group of rolling updates in a progressive rollout. It is also referred as steps, stages or phases
// of a progressive rollout.
type Wave struct {
	// Name identifies the wave.
	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	// MaxConcurrent specifies maximum number of concurrent updates to be performed in this wave.
	MaxConcurrent int64 `json:"maxConcurrent"`

	// Targets specifies the clusters that are part of this wave.
	Targets ClusterTargetSelector `json:"targets,omitempty"`
}

// ProgressiveRolloutStrategyStatus defines the observed state of ProgressiveRolloutStrategy
type ProgressiveRolloutStrategyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProgressiveRolloutStrategy is the Schema for the progressiverolloutstrategies API
type ProgressiveRolloutStrategy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProgressiveRolloutStrategySpec   `json:"spec,omitempty"`
	Status ProgressiveRolloutStrategyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProgressiveRolloutStrategyList contains a list of ProgressiveRolloutStrategy
type ProgressiveRolloutStrategyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProgressiveRolloutStrategy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProgressiveRolloutStrategy{}, &ProgressiveRolloutStrategyList{})
}
