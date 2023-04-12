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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PackageRevisionResources
// +k8s:openapi-gen=true
type PackageRevisionResources struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageRevisionResourcesSpec   `json:"spec,omitempty"`
	Status PackageRevisionResourcesStatus `json:"status,omitempty"`
}

// PackageRevisionResourcesList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PackageRevisionResourcesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PackageRevisionResources `json:"items"`
}

// PackageRevisionResourcesSpec represents resources (as ResourceList serialized as yaml string) of the PackageRevision.
type PackageRevisionResourcesSpec struct {
	// PackageName identifies the package in the repository.
	PackageName string `json:"packageName,omitempty"`

	// WorkspaceName identifies the workspace of the package.
	WorkspaceName WorkspaceName `json:"workspaceName,omitempty"`

	// Revision identifies the version of the package.
	Revision string `json:"revision,omitempty"`

	// RepositoryName is the name of the Repository object containing this package.
	RepositoryName string `json:"repository,omitempty"`

	// Resources are the content of the package.
	Resources map[string]string `json:"resources,omitempty"`
}

// PackageRevisionResourcesStatus represents state of the rendered package resources.
type PackageRevisionResourcesStatus struct {
	// RenderStatus contains the result of rendering the package resources.
	RenderStatus RenderStatus `json:"renderStatus,omitempty"`
}
