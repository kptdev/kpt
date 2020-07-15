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
	"strings"
	"testing"

	"gotest.tools/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
)

type circleCIOrbTestCase struct {
	description string
	config      *CircleCIOrbConfig
	expected    string
}

var circleCIOrbTestCases = []circleCIOrbTestCase{
	{
		description: "generate a CircleCI Orb",
		config: &CircleCIOrbConfig{
			PipelineConfig: &types.PipelineConfig{
				Dir:     "resources",
				FnPaths: nil,
			},
			ExecutorName: "kpt-container",
			CommandName:  "run-functions",
			JobName:      "run-kpt-functions",
		},
		expected: `
executors:
    kpt-container:
        docker:
          - image: gongpu/kpt:latest
commands:
    run-functions:
        steps:
          - run: kpt fn run resources
jobs:
    run-kpt-functions:
        executor: kpt-container
        steps:
          - setup_remote_docker
          - run-functions
`,
	},
	{
		description: "generate a CircleCI Orb with multiple fn-path",
		config: &CircleCIOrbConfig{
			PipelineConfig: &types.PipelineConfig{
				Dir: "resources",
				FnPaths: []string{
					"config/gate-keeper.yaml",
					"config/label-namespace.yaml",
				},
			},
			ExecutorName: "kpt-container",
			CommandName:  "run-functions",
			JobName:      "run-kpt-functions",
		},
		expected: `
executors:
    kpt-container:
        docker:
          - image: gongpu/kpt:latest
commands:
    run-functions:
        steps:
          - run: kpt fn run resources --fn-path config/gate-keeper.yaml --fn-path
                config/label-namespace.yaml
jobs:
    run-kpt-functions:
        executor: kpt-container
        steps:
          - setup_remote_docker
          - run-functions
`,
	},
}

func TestCircleCIOrb(t *testing.T) {
	for i := range circleCIOrbTestCases {
		testCase := circleCIOrbTestCases[i]

		t.Run(testCase.description, func(t *testing.T) {
			orb := new(CircleCIOrb).Init(testCase.config)

			marshalledOrb, err := yaml.Marshal(orb)
			testutil.AssertNoError(t, err)
			actual := string(marshalledOrb)
			expected := strings.TrimLeft(testCase.expected, "\n")

			assert.Equal(t, expected, actual)
		})
	}
}

var circleCITestCases = []testCase{
	{
		description: "generate a CircleCI workflow",
		config: &types.PipelineConfig{
			Dir: "local-resources",
		},
		expected: `
version: "2.1"
orbs:
    kpt:
        executors:
            kpt-container:
                docker:
                  - image: gongpu/kpt:latest
        commands:
            kpt-fn-run:
                steps:
                  - run: kpt fn run local-resources
        jobs:
            run-functions:
                executor: kpt-container
                steps:
                  - setup_remote_docker
                  - kpt-fn-run
workflows:
    main:
        jobs:
          - kpt/run-functions
`,
	},
	{
		description: "generate a CircleCI workflow with fn-path",
		config: &types.PipelineConfig{
			Dir:     "local-resources",
			FnPaths: []string{"functions.yaml"},
		},
		expected: `
version: "2.1"
orbs:
    kpt:
        executors:
            kpt-container:
                docker:
                  - image: gongpu/kpt:latest
        commands:
            kpt-fn-run:
                steps:
                  - run: kpt fn run local-resources --fn-path functions.yaml
        jobs:
            run-functions:
                executor: kpt-container
                steps:
                  - setup_remote_docker
                  - kpt-fn-run
workflows:
    main:
        jobs:
          - kpt/run-functions
`,
	},
}

var circleCITestSuite = testSuite{
	pipeline:  new(CircleCI),
	testCases: circleCITestCases,
}
