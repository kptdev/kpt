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
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
	"gopkg.in/yaml.v3"
)

// GitLabCi is a simplified representation of GitLab CI/CD Configuration.
// @see https://docs.gitlab.com/ee/ci/yaml/
type GitLabCI struct {
	Stages []string `yaml:",omitempty"`
	// @see https://github.com/go-yaml/yaml/issues/63
	GitLabCIStages `yaml:",inline"`
}

type GitLabCIStages map[string]GitLabCIStage

type GitLabCIStage struct {
	Stage    string   `yaml:",omitempty"`
	Image    string   `yaml:",omitempty"`
	Services []string `yaml:",omitempty"`
	Script   string   `yaml:",omitempty"`
}

func (p *GitLabCI) Init(config *types.PipelineConfig) Pipeline {
	stage := "run-kpt-functions"
	p.Stages = []string{stage}

	p.GitLabCIStages = map[string]GitLabCIStage{
		"kpt": {
			Stage:    stage,
			Image:    "docker",
			Services: []string{"docker:dind"},
			Script:   p.generateScript(config),
		},
	}

	return p
}

func (p *GitLabCI) Generate() (out []byte, err error) {
	return yaml.Marshal(p)
}

func (p *GitLabCI) generateScript(config *types.PipelineConfig) (script string) {
	mountDir := "/app"
	fnPaths := []string{"--fn-path"}

	for _, p := range config.FnPaths {
		// p is preprocessed in ExportRunner and guaranteed to be a relative path.
		fnPaths = append(fnPaths, path.Join(mountDir, p))
	}

	parts := []string{
		"docker run",
		fmt.Sprintf("-v $PWD:%s", mountDir),
		"-v /var/run/docker.sock:/var/run/docker.sock",
		KptImage,
		"fn run",
		path.Join(mountDir, config.Dir),
	}

	if len(config.FnPaths) > 0 {
		parts = append(parts, fnPaths...)
	}

	script = strings.Join(parts, " ")

	return
}
