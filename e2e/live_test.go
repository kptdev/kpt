// +build kind

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

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	livetest "github.com/GoogleContainerTools/kpt/pkg/test/live"
)

func TestLiveApply(t *testing.T) {
	runTests(t, filepath.Join(".", "testdata", "live-apply"))
}

func runTests(t *testing.T, path string) {
	testCases := scanTestCases(t, path)

	for p := range testCases {
		c := testCases[p]
		t.Run(p, func(t *testing.T) {
			(&livetest.Runner{
				Config: c,
				Path: p,
			}).Run(t)
		})
	}
}

func scanTestCases(t *testing.T, path string) map[string]livetest.TestCaseConfig {
	testCases := make(map[string]livetest.TestCaseConfig)
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}
		if path == p {
			return nil
		}

		testCases[p] = livetest.ReadTestCaseConfig(t, p)
		return filepath.SkipDir
	})
	if err != nil {
		t.Fatalf("failed to scan for test cases in %s", path)
	}
	return testCases
}