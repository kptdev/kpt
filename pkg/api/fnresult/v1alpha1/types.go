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

package v1alpha1

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Result contains the structured result from an individual function
type Result struct {
	// Image is the full name of the image that generates this result
	// Image and Exec are mutually exclusive
	Image string `yaml:"image,omitempty"`
	// ExecPath is the the absolute path to the executable file
	ExecPath string `yaml:"exec,omitempty"`
	// Stderr is the content in function stderr
	Stderr string `yaml:"stderr,omitempty"`
	// ExitCode is the exit code from running the function
	ExitCode int `yaml:"exitCode,omitempty"`
	// Results is the list of results for the function
	Results []framework.Item `yaml:"results,omitempty"`
}

const (
	ResultListKind       = "FunctionResultList"
	ResultListGroup      = "config.kubernetes.io"
	ResultListVersion    = "v1alpha1"
	ResultListAPIVersion = ResultListGroup + "/" + ResultListVersion
)

// ResultList contains aggregated results from multiple functions
type ResultList struct {
	yaml.ResourceMeta `yaml:",inline"`
	// ExitCode is the exit code of kpt command
	ExitCode int `yaml:"exitCode,omitempty"`
	// Items contain a list of function result
	Items []Result `yaml:"items,omitempty"`
}
