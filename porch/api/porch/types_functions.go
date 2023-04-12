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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Function represents a kpt function discovered in a repository
// Function resources are created automatically by discovery in a registered Repository.
// Function resource names will be computed as <Repository Name>:<Function Name>
// to ensure uniqueness of names, and will follow formatting of
// [DNS Subdomain Names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names).
// +k8s:openapi-gen=true
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

// FunctionList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Function `json:"items"`
}

type FunctionType string

const (
	FunctionTypeValidator FunctionType = "validator"
	FunctionTypeMutator   FunctionType = "mutator"
)

// FunctionSpec defines the desired state of a Function
type FunctionSpec struct {
	// Image specifies the function image, such as 'gcr.io/kpt-fn/gatekeeper:v0.2'.
	Image string `json:"image"`

	// RepositoryRef references the repository in which the function is located.
	RepositoryRef RepositoryRef `json:"repositoryRef"`

	// FunctionType specifies the function types (mutator, validator or/and others).
	FunctionTypes []FunctionType `json:"functionTypes,omitempty"`

	FunctionConfigs []FunctionConfig `json:"functionConfigs,omitempty"`

	// Keywords are used as filters to provide correlation in function discovery.
	Keywords []string `json:"keywords,omitempty"`

	// Description is a short description of the function.
	Description string `json:"description"`

	// `DocumentationUrl specifies the URL of comprehensive function documentation`
	DocumentationUrl string `json:"documentationUrl,omitempty"`

	// InputTypes specifies to which input KRM types the function applies. Specified as Group Version Kind.
	// For example:
	//
	//    inputTypes:
	//    - kind: RoleBinding
	//      # If version is unspecified, applies to all versions
	//      apiVersion: rbac.authorization.k8s.io
	//    - kind: ClusterRoleBinding
	//      apiVersion: rbac.authorization.k8s.io/v1
	// InputTypes []metav1.TypeMeta

	// OutputTypes specifies types of any KRM resources the function creates
	// For example:
	//
	//     outputTypes:
	//     - kind: ConfigMap
	//       apiVersion: v1
	// OutputTypes []metav1.TypeMeta

}

// FunctionConfig specifies all the valid types of the function config for this function.
// If unspecified, defaults to v1/ConfigMap. For example, function `set-namespace` accepts both `ConfigMap` and `SetNamespace`
type FunctionConfig struct {
	metav1.TypeMeta `json:",inline"`
	// Experimental: requiredFields tells necessary fields and is aimed to help users write the FunctionConfig.
	// Otherwise, users can get the required fields info from the function evaluation error message.
	RequiredFields []string `json:"requiredFields,omitempty"`
}

// FunctionRef is a reference to a Function resource.
type FunctionRef struct {
	// Name is the name of the Function resource referenced. The resource is expected to be within the same namespace.
	Name string `json:"name"`
}

// FunctionStatus defines the observed state of Function
type FunctionStatus struct {
}
