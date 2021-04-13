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

import "github.com/GoogleContainerTools/kpt/internal/cmdexport/types"

//nolint:lll
var gitlabCITestCases = []testCase{
	{
		description: "generate a GitLab CI pipeline against the current directory",
		config: &types.PipelineConfig{
			Dir: ".",
		},
		expected: `
stages:
    - run-kpt-functions
kpt:
    stage: run-kpt-functions
    image: docker
    services:
        - docker:dind
    script: docker run -v $PWD:/app -v /var/run/docker.sock:/var/run/docker.sock gcr.io/kpt-dev/kpt:latest fn run /app
`,
	},
	{
		description: "generate a GitLab CI pipeline with --fn-path",
		config: &types.PipelineConfig{
			Dir: "resources",
			FnPaths: []string{
				"config/label-namespace.yaml",
				"config/application-cr.yaml",
			},
		},
		expected: `
stages:
    - run-kpt-functions
kpt:
    stage: run-kpt-functions
    image: docker
    services:
        - docker:dind
    script: docker run -v $PWD:/app -v /var/run/docker.sock:/var/run/docker.sock gcr.io/kpt-dev/kpt:latest fn run /app/resources --fn-path /app/config/label-namespace.yaml /app/config/application-cr.yaml
`,
	},
}

var gitlabCITestSuite = testSuite{
	pipeline:  new(GitLabCI),
	testCases: gitlabCITestCases,
}
