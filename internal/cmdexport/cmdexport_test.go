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

// Use file path as key, and content as value.
type files map[string]string

type TestCase struct {
	description string
	params      []string
	expected    string
	err         string
	files       files
}

var testCases = []TestCase{
	{
		description: "fails on not providing enough args",
		params:      []string{"github-actions"},
		err:         "accepts 2 args, received 1",
	},
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
			"main.yaml",
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
		files: map[string]string{
			"function.yaml": "",
		},
		params: []string{"github-actions", ".", "--fn-path", "function.yaml"},
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
                args: fn run . --fn-path function.yaml
`,
	},
	{
		description: "exports a Cloud Build pipeline with --fn-path",
		files: map[string]string{
			"functions/function.yaml": "",
		},
		params: []string{
			"cloud-build",
			".",
			"--fn-path",
			"functions/",
			"--output",
			"cloudbuild.yaml",
		},
		expected: `
steps:
  - name: gongpu/kpt:latest
    args:
      - fn
      - run
      - .
      - --fn-path
      - functions/
`,
	},
	{
		description: "fails to export a Cloud Build pipeline with non-exist function path",
		params: []string{
			"cloud-build",
			".",
			"--fn-path",
			"functions.yaml",
			"--output",
			"cloudbuild.yaml",
		},
		err: "function path (functions.yaml) does not exist",
	},
	{
		description: "fails to export a Cloud Build pipeline with outside function path",
		params: []string{
			"cloud-build",
			".",
			"--fn-path",
			"../functions/functions.yaml",
			"--output",
			"cloudbuild.yaml",
		},
		err: "function path (../functions/functions.yaml) is not within the current working directory",
	},
}

func setupTempDir(files files) (dir string, err error) {
	tempDir, err := ioutil.TempDir("", "kpt-fn-export-test")
	if err != nil {
		return "", err
	}

	for p, content := range files {
		p = filepath.Join(tempDir, p)

		err = os.MkdirAll(
			filepath.Dir(p),
			0755, // drwxr-xr-x
		)
		if err != nil {
			return "", nil
		}

		err = ioutil.WriteFile(
			p,
			[]byte(content),
			0644, // -rw-r--r--
		)
		if err != nil {
			return "", err
		}
	}

	return tempDir, nil
}

func TestCmdExport(t *testing.T) {
	for i := range testCases {
		testCase := testCases[i]
		tempDir, err := setupTempDir(testCase.files)
		assert.NilError(t, err)
		err = os.Chdir(tempDir)
		assert.NilError(t, err)

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

		_ = os.RemoveAll(tempDir)
	}
}
