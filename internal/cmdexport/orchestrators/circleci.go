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

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
)

// CircleCI represents a config file for CircleCI pipelines.
type CircleCI struct {
	Version   string                       `yaml:",omitempty"`
	Orbs      map[string]*CircleCIOrb      `yaml:",omitempty"`
	Workflows map[string]*CircleCIWorkflow `yaml:",omitempty"`
}

func (p *CircleCI) Init(config *types.PipelineConfig) Pipeline {
	p.Version = "2.1"

	orbName := "kpt"
	commandName := "kpt-fn-run"
	jobName := "run-functions"
	orbConfig := &CircleCIOrbConfig{
		PipelineConfig: config,
		ExecutorName:   "kpt-container",
		CommandName:    commandName,
		JobName:        jobName,
	}
	orb := new(CircleCIOrb).Init(orbConfig)

	p.Orbs = map[string]*CircleCIOrb{
		orbName: orb,
	}

	p.Workflows = map[string]*CircleCIWorkflow{
		"main": {
			Jobs: []string{
				fmt.Sprintf("%s/%s", orbName, jobName),
			},
		},
	}

	return p
}

func (p *CircleCI) Generate() (out []byte, err error) {
	return yaml.Marshal(p)
}

// CircleCIOrb represents a reusable orb object that is a collection of executors, commands, and jobs.
type CircleCIOrb struct {
	Executors map[string]*CircleCIExecutor `yaml:",omitempty"`
	Commands  map[string]*CircleCICommand  `yaml:",omitempty"`
	Jobs      map[string]*CircleCIJob      `yaml:",omitempty"`
}

// CircleCIOrbConfig allows to customize a CircleCI Orb object.
type CircleCIOrbConfig struct {
	*types.PipelineConfig
	ExecutorName string
	CommandName  string
	JobName      string
}

func (orb *CircleCIOrb) Init(config *CircleCIOrbConfig) *CircleCIOrb {
	orb.Executors = map[string]*CircleCIExecutor{
		config.ExecutorName: {
			"docker": []*CircleCIDockerExecutor{
				{Image: KptImage},
			},
		},
	}

	command := fmt.Sprintf("kpt fn run %s", config.Dir)
	for _, fnPath := range config.FnPaths {
		command = fmt.Sprintf("%s --fn-path %s", command, fnPath)
	}

	orb.Commands = map[string]*CircleCICommand{
		config.CommandName: {
			Steps: []*CircleCICommandStep{
				{Run: command},
			},
		},
	}

	orb.Jobs = map[string]*CircleCIJob{
		config.JobName: {
			Executor: config.ExecutorName,
			Steps: []string{
				"setup_remote_docker",
				config.CommandName,
			},
		},
	}

	return orb
}

// CircleCIExecutor represents an executor which only has one key as its type in the map.
type CircleCIExecutor = map[string][]*CircleCIDockerExecutor

// CircleCIDockerExecutor represents a dicker executor.
type CircleCIDockerExecutor struct {
	Image string `yaml:",omitempty"`
}

// CircleCICommand represents a multi-step command.
type CircleCICommand struct {
	Steps []*CircleCICommandStep `yaml:",omitempty"`
}

// CircleCICommandStep represents a step in the command steps.
type CircleCICommandStep struct {
	Run string `yaml:",omitempty"`
}

// CircleCIJob wraps a sequence of commands to run and their executor.
type CircleCIJob struct {
	Executor string   `yaml:",omitempty"`
	Steps    []string `yaml:",omitempty"`
}

// CircleCIWorkflow defines a sequence of job to execute.
type CircleCIWorkflow struct {
	Jobs []string `yaml:",omitempty"`
}
