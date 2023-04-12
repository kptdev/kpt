// Copyright 2021 The kpt Authors
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

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Result contains the structured result from an individual function
type Result struct {
	// Image is the full name of the image that generates this result
	// Image and Exec are mutually exclusive
	Image string `yaml:"image,omitempty"`
	// ExecPath is the the absolute os-specific path to the executable file
	// If user provides an executable file with commands, ExecPath should
	// contain the entire input string.
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
	Results framework.Results `yaml:"results,omitempty"`
}

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

// KptFileGVK is the GroupVersionKind of FunctionResultList objects
func ResultListGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "kpt.dev",
		Version: "v1",
		Kind:    "FunctionResultList",
	}
}

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
