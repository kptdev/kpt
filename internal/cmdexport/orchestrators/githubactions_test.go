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
)

var githubActionsTestCases = []testCase{
	{
		description: "generate GitHub Actions pipeline against the current directory",
		config: &types.PipelineConfig{
			Dir: ".",
		},
		expected: `
name: kpt
on:
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - name: Run all kpt functions
            uses: docker://gongpu/kpt:latest
            with:
                args: fn run .
`,
	},
	{
		description: "generates a GitHub Actions pipeline with --fn-path",
		config: &types.PipelineConfig{
			Dir:     ".",
			FnPaths: []string{"functions/"},
		},
		expected: `
name: kpt
on:
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - name: Run all kpt functions
            uses: docker://gongpu/kpt:latest
            with:
                args: fn run . --fn-path functions/
`,
	},
	{
		description: "generates a GitHub Actions pipeline with multiple function paths",
		config: &types.PipelineConfig{
			Dir:     ".",
			FnPaths: []string{"functions1/", "functions2/"},
		},
		expected: `
name: kpt
on:
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - name: Run all kpt functions
            uses: docker://gongpu/kpt:latest
            with:
                args: fn run . --fn-path functions1/ functions2/
`,
	},
}

var githubActionsTestSuite = testSuite{
	pipeline:  new(GitHubActions),
	testCases: githubActionsTestCases,
}
