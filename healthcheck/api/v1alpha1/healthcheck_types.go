// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

// NOTE: json tags are required.  Any new fields you add must have
// json tags for the fields to be serialized.
// HealthCheckSpec defines the metadata of a single health check.
type HealthCheckSpec struct {
}

// HealthCheckStatus defines the status of a single health check.
type HealthCheckStatus struct {
	// Conditions represents the status of health check.
	// +kubebuilder:validation:MaxItems=1
	Conditions []HealthCheckCondition `json:"conditions,omitempty"`
}

// HealthCheckConditionType defines the type of health check conditions.
type HealthCheckConditionType string

// The valid conditions of health check.
const (
	FatalError    HealthCheckConditionType = "FatalError"
	NonFatalError HealthCheckConditionType = "NonFatalError"
)

// HealthCheckCondition represents the status of health check.
type HealthCheckCondition struct {
	// +kubebuilder:validation:Enum=FatalError;NonFatalError
	Type HealthCheckConditionType `json:"type,omitempty"`
	// +kubebuilder:validation:Enum=Unknown;Healthy;Unhealthy
	Status metav1.ConditionStatus `json:"status,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The unique error ID.
	CanonicalID string `json:"canonicalID,omitempty"`
	// The unique error name.
	CanonicalName string `json:"canonicalName,omitempty"`
	// The last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

const (
	// LabelComponent indicates the component to which the health check belongs
	LabelComponent = "healthcheck.krmapihosting.cloud.google.com/component"
	// LabelServiceError is set to true if the health check is service level
	LabelServiceError = "healthcheck.krmapihosting.cloud.google.com/serviceError"
	// LabelUnknown means users haven't set the label value.
	LabelUnknown = "Unknown"
)

// +kubebuilder:object:root=true
// HealthCheck is the Schema for a single health check.
type HealthCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HealthCheckSpec   `json:"spec,omitempty"`
	Status            HealthCheckStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// HealthCheckList contains a list of HealthCheck.
type HealthCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealthCheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HealthCheck{}, &HealthCheckList{})
}
