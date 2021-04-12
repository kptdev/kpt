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
		description: "fails on providing too many args",
		params:      []string{"dir", "extra"},
		err:         "accepts 1 arg(s), received 2",
	},
	{
		description: "fails on not providing working orchestrator",
		params:      []string{"dir"},
		err:         "--workflow flag is required. It must be one of circleci, cloud-build, github-actions, gitlab-ci, jenkins, tekton",
	},
	{
		description: "fails on an unsupported workflow orchestrator",
		params:      []string{".", "--workflow", "random-orchestrator"},
		err:         "unsupported orchestrator random-orchestrator. It must be one of circleci, cloud-build, github-actions, gitlab-ci, jenkins, tekton",
	},
	{
		description: "exports a GitHub Actions pipeline",
		params:      []string{".", "--workflow", "github-actions"},
		expected: `
name: kpt
"on":
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - name: Run all kpt functions
            uses: docker://gcr.io/kpt-dev/kpt:latest
            with:
                args: fn run .
`,
	},
	{
		description: "exports a GitHub Actions pipeline with --output",
		params: []string{
			".",
			"-w",
			"github-actions",
			"--output",
			"main.yaml",
		},
		expected: `
name: kpt
"on":
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - name: Run all kpt functions
            uses: docker://gcr.io/kpt-dev/kpt:latest
            with:
                args: fn run .
`,
	},
	{
		description: "exports a GitHub Actions pipeline with --fn-path",
		files: map[string]string{
			"function.yaml": "",
		},
		params: []string{".", "--fn-path", "function.yaml", "-w", "github-actions"},
		expected: `
name: kpt
"on":
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - name: Run all kpt functions
            uses: docker://gcr.io/kpt-dev/kpt:latest
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
			".",
			"--fn-path",
			"functions/",
			"--output",
			"cloudbuild.yaml",
			"-w",
			"cloud-build",
		},
		expected: `
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - .
      - --fn-path
      - functions
`,
	},
	{
		description: "fails to export a Cloud Build pipeline with outside function paths",
		params: []string{
			".",
			"--fn-path",
			"../functions/functions.yaml",
			"--fn-path",
			"../functions/functions2.yaml",
			"-w",
			"cloud-build",
			"--output",
			"cloudbuild.yaml",
		},
		err: `
function paths are not within the current working directory:
../functions/functions.yaml
../functions/functions2.yaml`,
	},
	{
		description: "converts input paths into relative format",
		files: map[string]string{
			"functions/function.yaml": "",
		},
		params: []string{
			// NOTE: `{DIR}` is a macro variable and will be replaced with cwd before test cases are executed.
			"{DIR}",
			"-w",
			"cloud-build",
			"--fn-path",
			"{DIR}/functions/",
			"--output",
			"cloudbuild.yaml",
		},
		expected: `
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - .
      - --fn-path
      - functions
`,
	},
	{
		description: "exports a GitLab CI pipeline with --fn-path",
		files: map[string]string{
			"resources/resource.yaml": "",
			"functions/function.yaml": "",
		},
		params: []string{
			"resources",
			"--fn-path",
			"functions",
			"-w",
			"gitlab-ci",
			"--output",
			".gitlab-ci.yml",
		},
		expected: `
stages:
  - run-kpt-functions
kpt:
    stage: run-kpt-functions
    image: docker
    services:
      - docker:dind
    script: docker run -v $PWD:/app -v /var/run/docker.sock:/var/run/docker.sock gcr.io/kpt-dev/kpt:latest
        fn run /app/resources --fn-path /app/functions
`,
	},
	{
		description: "exports a Jenkinsfile",
		params: []string{
			"resources",
			"-w",
			"jenkins",
		},
		expected: `
pipeline {
    agent any

    stages {
        stage('Run kpt functions') {
            steps {
                // This requires that docker is installed on the agent.
                // And your user, which is usually "jenkins", should be added to the "docker" group to access "docker.sock".
                sh '''
                    docker run \
                    -v $PWD:/app \
                    -v /var/run/docker.sock:/var/run/docker.sock \
                    gcr.io/kpt-dev/kpt:latest \
                    fn run /app/resources
                '''
            }
        }
    }
}
`,
	},
	{
		description: "exports a Jenkinsfile with --fn-path",
		params: []string{
			"resources",
			"-w",
			"jenkins",
			"--fn-path",
			"functions/label-namespace.yaml",
			"--fn-path",
			"functions/gate-keeper.yaml",
		},
		expected: `
pipeline {
    agent any

    stages {
        stage('Run kpt functions') {
            steps {
                // This requires that docker is installed on the agent.
                // And your user, which is usually "jenkins", should be added to the "docker" group to access "docker.sock".
                sh '''
                    docker run \
                    -v $PWD:/app \
                    -v /var/run/docker.sock:/var/run/docker.sock \
                    gcr.io/kpt-dev/kpt:latest \
                    fn run /app/resources \
                    --fn-path /app/functions/label-namespace.yaml \
                    --fn-path /app/functions/gate-keeper.yaml
                '''
            }
        }
    }
}
`,
	},
	{
		description: "exports a CircleCI workflow",
		params: []string{
			"resources",
			"-w",
			"circleci",
		},
		expected: `
version: "2.1"
orbs:
    kpt:
        executors:
            kpt-container:
                docker:
                  - image: gcr.io/kpt-dev/kpt:latest
        commands:
            kpt-fn-run:
                steps:
                  - run: kpt fn run resources
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
		description: "exports a CircleCI workflow with fn-path",
		params: []string{
			"resources",
			"--fn-path",
			"config/functions.yaml",
			"-w",
			"circleci",
		},
		expected: `
version: "2.1"
orbs:
    kpt:
        executors:
            kpt-container:
                docker:
                  - image: gcr.io/kpt-dev/kpt:latest
        commands:
            kpt-fn-run:
                steps:
                  - run: kpt fn run resources --fn-path config/functions.yaml
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

// ReplaceDIRMacro replaces all `{DIR}` macros in params with cwd.
func (t *TestCase) ReplaceDIRMacro() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var params []string
	for _, param := range t.params {
		param = strings.ReplaceAll(param, "{DIR}", cwd)

		params = append(params, param)
	}

	t.params = params

	return nil
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
	t.Skip()
	for i := range testCases {
		testCase := testCases[i]
		tempDir, err := setupTempDir(testCase.files)
		assert.NilError(t, err)
		err = os.Chdir(tempDir)
		assert.NilError(t, err)

		err = testCase.ReplaceDIRMacro()
		assert.NilError(t, err)

		t.Run(testCase.description, func(t *testing.T) {

			r := GetExportRunner()
			r.Command.SetArgs(testCase.params)

			b := &bytes.Buffer{}
			// out will be overridden during execution if OutputFilePath is present.
			r.Command.SetOut(b)

			err := r.Command.Execute()

			if testCase.err != "" {
				expectedError := strings.TrimLeft(testCase.err, "\n")
				assert.Error(t, err, expectedError)
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
