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

package cmdexport

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"
)

var tempDir, _ = ioutil.TempDir("", "kpt-fn-export-test")

type TestCase struct {
	description string
	params      []string
	expected    string
	err         string
}

var testCases = []TestCase{
	{
		description: "fails on an unsupported orchestrator",
		params:      []string{"random-orchestrator", "."},
		err:         "unsupported orchestrator random-orchestrator",
	},
	{
		description: "exports a GitHub Actions pipeline",
		params:      []string{"github-actions", "."},
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
		description: "exports a GitHub Actions pipeline with --output",
		params: []string{
			"github-actions",
			".",
			"--output",
			filepath.Join(tempDir, "main.yaml"),
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
		description: "exports a GitHub Actions pipeline with --fn-path",
		params:      []string{"github-actions", ".", "--fn-path", "functions/"},
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
}

func TestCmdExport(t *testing.T) {
	for i := range testCases {
		testCase := testCases[i]

		t.Run(testCase.description, func(t *testing.T) {
			r := GetExportRunner()
			r.Command.SetArgs(testCase.params)

			b := &bytes.Buffer{}
			// out will be overridden during execution if OutputFilePath is present.
			r.Command.SetOut(b)

			err := r.Command.Execute()

			if testCase.err != "" {
				assert.Error(t, err, testCase.err)
			} else {
				assert.NilError(t, err)

				expected := strings.TrimLeft(testCase.expected, "\n")
				var actual string
				if r.OutputFilePath == "" {
					actual = b.String()
				} else {
					content, _ := ioutil.ReadFile(r.OutputFilePath)

					actual = string(content)
				}

				assert.Equal(t, expected, actual)
			}
		})
	}

	_ = os.RemoveAll(tempDir)
}
