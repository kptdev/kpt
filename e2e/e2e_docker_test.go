// +build docker

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

package e2e_test

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/test/runner"
)

// These e2e tests currently fail. When pipeline is implemented they should pass.
func TestPipeline(t *testing.T) {
	runPipelineTests(t, "../internal/pipeline/testdata/")
}

// runTests will scan test cases in 'path', run the command
// `kpt pipeline run` on all of the packages in path, and test that
// the diff between the results and the original package is as
// expected
func runPipelineTests(t *testing.T, path string) {
	cases, err := runner.ScanTestCases(path)
	if err != nil {
		t.Fatalf("failed to scan test cases: %s", err)
	}
	for _, c := range *cases {
		c := c // capture range variable
		t.Run(c.Path, func(t *testing.T) {
			t.Parallel()
			r, err := runner.NewRunner(c, runner.CommandPipeline)
			if err != nil {
				t.Fatalf("failed to create test runner: %s", err)
			}
			err = r.Run()
			if err != nil {
				t.Fatalf("failed when running test: %s", err)
			}
		})
	}
}
