// Copyright 2019 Google LLC
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

package fn

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// FunctionResult contains the structured result from an individual function
type FunctionResult struct {
	// Image is the full name of the image that generates this result
	// TODO: Image and Exec are mutually exclusive
	Image string `yaml:"image,omitempty"`
	// Exec is the the absolute path to the executable file
	Exec string `yaml:"exec,omitempty"`
	// ExitCode is the exit code from running the function
	ExitCode int `yaml:"exitCode,omitempty"`
	// Results is the list of results for the function
	// TODO: The type has been changed to ResultItem in newer version of kyaml
	Results []framework.Item `yaml:"results,omitempty"`
}

// FunctionResultList contains aggregated results from multiple functions
type FunctionResultList struct {
	// The output GVK must be:
	//  apiVersion: config.kubernetes.io/v1beta1
	//  kind: FunctionResultList
	yaml.ResourceMeta `yaml:",inline"`
	// Items contain a list of function result
	Items []FunctionResult `yaml:"items,omitempty"`
}
