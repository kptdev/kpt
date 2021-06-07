// Copyright 2021 Google LLC
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
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Result contains the structured result from an individual function
type Result struct {
	// Image is the full name of the image that generates this result
	// Image and Exec are mutually exclusive
	Image string `yaml:"image,omitempty"`
	// ExecPath is the the absolute os-specific path to the executable file
	ExecPath string `yaml:"exec,omitempty"`
	// TODO(droot): This is required for making structured results subpackage aware.
	// Enable this once test harness supports filepath based assertions.
	// Pkg is OS specific Absolute path to the package.
	// Pkg string `yaml:"pkg,omitempty"`
	// Stderr is the content in function stderr
	Stderr string `yaml:"stderr,omitempty"`
	// ExitCode is the exit code from running the function
	ExitCode int `yaml:"exitCode"`
	// Results is the list of results for the function
	Results []ResultItem `yaml:"results,omitempty"`
}

const (
	ResultListKind       = "FunctionResultList"
	ResultListGroup      = kptfilev1alpha2.KptFileGroup
	ResultListVersion    = kptfilev1alpha2.KptFileVersion
	ResultListAPIVersion = ResultListGroup + "/" + ResultListVersion
)

// ResultList contains aggregated results from multiple functions
type ResultList struct {
	yaml.ResourceMeta `yaml:",inline"`
	// ExitCode is the exit code of kpt command
	ExitCode int `yaml:"exitCode"`
	// Items contain a list of function result
	Items []Result `yaml:"items,omitempty"`
}

// NewResultList returns an instance of ResultList with metadata
// field populated.
func NewResultList() *ResultList {
	return &ResultList{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: ResultListAPIVersion,
				Kind:       ResultListKind,
			},
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: "fnresults",
				},
			},
		},
		Items: []Result{},
	}
}

// ResultItem is a duplicate of framework.ResultItem, except that
// ResourceRef uses yaml.ResourceIdentifier here, whereas framework.ResultItem
// uses yaml.ResourceMeta. Eventually, we will need to fix it upstream.
// TODO: https://github.com/GoogleContainerTools/kpt/issues/2091
type ResultItem struct {
	// Message is a human readable message
	Message string `yaml:"message,omitempty"`

	// Severity is the severity of this result
	Severity framework.Severity `yaml:"severity,omitempty"`

	// ResourceRef is a reference to a resource
	ResourceRef yaml.ResourceIdentifier `yaml:"resourceRef,omitempty"`

	// Field is a reference to the field in a resource this result refers to
	Field framework.Field `yaml:"field,omitempty"`

	// File references a file containing the resource this result refers to
	File framework.File `yaml:"file,omitempty"`
}
