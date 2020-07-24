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

var cloudBuildTestCases = []testCase{
	{
		description: "generates a Cloud Build pipeline",
		config: &types.PipelineConfig{
			Dir:     ".",
			FnPaths: nil,
		},
		expected: `
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - .
`,
	},
	{
		description: "generates a Cloud Build pipeline with --fn-path",
		config: &types.PipelineConfig{
			Dir:     "resources",
			FnPaths: []string{"config/function.yaml"},
		},
		expected: `
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - resources
      - --fn-path
      - config/function.yaml
`,
	},
	{
		description: "generates a Cloud Build pipeline with multiple --fn-path",
		config: &types.PipelineConfig{
			Dir:     "resources",
			FnPaths: []string{"config/function1.yaml", "config/function2.yaml"},
		},
		expected: `
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - resources
      - --fn-path
      - config/function1.yaml
      - config/function2.yaml
`,
	},
}

var cloudBuildTestSuite = testSuite{
	pipeline:  &CloudBuild{},
	testCases: cloudBuildTestCases,
}
