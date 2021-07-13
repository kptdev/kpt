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
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/test/runner"
)

func TestFnRender(t *testing.T) {
	runAllTests(t, filepath.Join(".", "testdata", "fn-render"))
}

func TestFnEval(t *testing.T) {
	runAllTests(t, filepath.Join(".", "testdata", "fn-eval"))
}

func TestFnSink(t *testing.T) {
	runAllTests(t, filepath.Join(".", "testdata", "fn-sink"))
}

// runTests will scan test cases in 'path', run the command
// on all of the packages in path, and test that
// the diff between the results and the original package is as
// expected
func runAllTests(t *testing.T, path string) {
	cases, err := runner.ScanTestCases(path)
	if err != nil {
		t.Fatalf("failed to scan test cases: %s", err)
	}
	// Run all the sequential tests first then run the parallel tests.
	runTests(t, cases, true)
	runTests(t, cases, false)
}

func runTests(t *testing.T, cases *runner.TestCases, sequential bool) {
	for _, c := range *cases {
		c := c // capture range variable
		if c.Config.Sequential != sequential {
			continue
		}
		t.Run(c.Path, func(t *testing.T) {
			if !c.Config.Sequential {
				t.Parallel()
			}
			r, err := runner.NewRunner(t, c, c.Config.TestType)
			if err != nil {
				t.Fatalf("failed to create test runner: %s", err)
			}
			if r.Skip() {
				t.Skip()
			}
			err = r.Run()
			if err != nil {
				t.Fatalf("failed when running test: %s", err)
			}
		})
	}
}
