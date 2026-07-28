//go:build docker

// Copyright 2026 The kpt Authors
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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kptdev/kpt/pkg/test/runner"
)

// TestPackageExamples runs e2e tests against the package-examples directory.
// Test fixtures (expected output) are stored in e2e/testdata/package-examples/
// while the actual packages live in the top-level package-examples/ directory.
// This keeps documentation examples clean of test-specific files.
//
// To generate or update expected output:
//
//	KPT_E2E_UPDATE_EXPECTED=true go test --tags=docker --run=TestPackageExamples ./e2e/
func TestPackageExamples(t *testing.T) {
	testdataDir := filepath.Join(".", "testdata", "package-examples")
	pkgExamplesDir := filepath.Join("..", "package-examples")
	updateExpected := strings.ToLower(os.Getenv("KPT_E2E_UPDATE_EXPECTED")) == "true"

	cases, err := runner.ScanTestCases(testdataDir)
	if err != nil {
		t.Fatalf("failed to scan test cases: %s", err)
	}

	for _, c := range *cases {
		c := c
		name := filepath.Base(c.Path)
		t.Run(name, func(t *testing.T) {
			if !c.Config.Sequential {
				t.Parallel()
			}

			pkgSrc := filepath.Join(pkgExamplesDir, name)
			if _, err := os.Stat(filepath.Join(pkgSrc, "Kptfile")); err != nil {
				t.Fatalf("package-example %q does not have a Kptfile: %v", name, err)
			}

			// Create a merged directory: package content + test fixtures
			tmpDir, err := os.MkdirTemp("", "kpt-pkg-examples-e2e-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			mergedPkg := filepath.Join(tmpDir, name)

			// Copy the actual package content
			if out, err := exec.Command("cp", "-r", pkgSrc, mergedPkg).CombinedOutput(); err != nil {
				t.Fatalf("failed to copy package: %v\n%s", err, out)
			}

			// Copy the test fixtures (.expected, .krmignore) on top
			testFixtures := c.Path
			entries, err := os.ReadDir(testFixtures)
			if err != nil {
				t.Fatalf("failed to read test fixtures dir: %v", err)
			}
			for _, entry := range entries {
				src := filepath.Join(testFixtures, entry.Name())
				dst := filepath.Join(mergedPkg, entry.Name())
				if out, err := exec.Command("cp", "-r", src, dst).CombinedOutput(); err != nil {
					t.Fatalf("failed to copy fixture %s: %v\n%s", entry.Name(), err, out)
				}
			}

			// Use the standard runner on the merged package
			mergedCase := runner.TestCase{
				Path:   mergedPkg,
				Config: c.Config,
			}

			r, err := runner.NewRunner(t, mergedCase, c.Config.TestType)
			if err != nil {
				t.Fatalf("failed to create test runner: %v", err)
			}
			if r.Skip() {
				t.Skip()
			}
			if err := r.Run(); err != nil {
				t.Fatalf("failed when running test: %v", err)
			}

			if updateExpected {
				// Copy generated expected output back to testdata
				generatedExpected := filepath.Join(mergedPkg, ".expected")
				targetExpected := filepath.Join(testFixtures, ".expected")
				if out, cpErr := exec.Command("cp", "-r", generatedExpected+"/.", targetExpected).CombinedOutput(); cpErr != nil {
					t.Fatalf("failed to copy updated expected output: %v\n%s", cpErr, out)
				}
				t.Logf("updated expected output for %s", name)
			}
		})
	}
}
