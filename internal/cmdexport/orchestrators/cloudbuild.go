// Copyright 2020 Google LLC
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

package orchestrators

import (
	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// CloudBuild is a simplified representation of Cloud Build config.
// @see https://cloud.google.com/cloud-build/docs/build-config
type CloudBuild struct {
	Steps []CloudBuildStep `yaml:",omitempty"`
}

type CloudBuildStep struct {
	Name string   `yaml:",omitempty"`
	Args []string `yaml:",omitempty"`
}

func (p *CloudBuild) Init(config *types.PipelineConfig) Pipeline {
	step := CloudBuildStep{}
	step.Name = KptImage
	step.Args = []string{
		"fn",
		"run",
		config.Dir,
	}

	if fnPaths := config.FnPaths; len(fnPaths) > 0 {
		step.Args = append(
			step.Args,
			append(
				[]string{"--fn-path"},
				fnPaths...
			)...
		)
	}

	p.Steps = []CloudBuildStep{step}

	return p
}

func (p *CloudBuild) Generate() []byte {
	data, _ := yaml.Marshal(p)

	return data
}
