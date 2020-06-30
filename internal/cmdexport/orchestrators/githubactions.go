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
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Represent a GitHub Actions workflow.
// @see https://help.github.com/en/actions/reference/workflow-syntax-for-github-actions
type GitHubActions struct {
	Name string                          `yaml:",omitempty"`
	On   map[string]GitHubActionsTrigger `yaml:",omitempty"`
	Jobs map[string]GitHubActionsJob     `yaml:",omitempty"`
}

type GitHubActionsTrigger struct {
	Branches []string `yaml:",omitempty"`
}

type GitHubActionsJob struct {
	RunsOn string              `yaml:"runs-on,omitempty"`
	Steps  []GitHubActionsStep `yaml:",omitempty"`
}

type GitHubActionsStep struct {
	Name string               `yaml:",omitempty"`
	Uses string               `yaml:",omitempty"`
	With GitHubActionStepArgs `yaml:",omitempty"`
}

type GitHubActionStepArgs struct {
	Args string `yaml:",omitempty"`
}

func (p *GitHubActions) Init(config *types.PipelineConfig) Pipeline {
	var runFnCommand = fmt.Sprintf("fn run %s", config.Dir)

	if fnPaths := config.FnPaths; len(fnPaths) > 0 {
		runFnCommand = fmt.Sprintf(
			"%s --fn-path %s",
			runFnCommand,
			strings.Join(fnPaths, " "),
		)
	}

	p.Name = "kpt"
	p.On = map[string]GitHubActionsTrigger{
		"push": {Branches: []string{"master"}},
	}
	p.Jobs = map[string]GitHubActionsJob{
		"Kpt": {
			RunsOn: "ubuntu-latest",
			Steps: []GitHubActionsStep{
				{
					Name: "Run all kpt functions",
					Uses: "docker://" + KptImage,
					With: GitHubActionStepArgs{
						Args: runFnCommand,
					},
				},
			},
		},
	}

	return p
}

func (p *GitHubActions) Generate() []byte {
	data, _ := yaml.Marshal(p)

	return data
}
