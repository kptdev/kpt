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

package cmdsearch

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/cmd/config/ext"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

func TestSearchCommand(t *testing.T) {
	var tests = []struct {
		name              string
		input             string
		args              []string
		out               string
		expectedResources string
		errMsg            string
	}{
		{
			name: "search by value",
			args: []string{"--by-value", "3"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
			out: `${baseDir}/
matched 2 field(s)
${filePath}:  spec.replicas: 3
${filePath}:  spec.foo: 3
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
		},
		{
			name: "search replace by value",
			args: []string{"--by-value", "3", "--put-literal", "4"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
foo:
  bar: 3
 `,
			out: `${baseDir}/
matched 2 field(s)
${filePath}:  spec.replicas: 4
${filePath}:  foo.bar: 4
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
foo:
  bar: 4
 `,
		},
		{
			name: "search replace multiple deployments",
			args: []string{"--by-value", "3", "--put-literal", "4"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 3
 `,
			out: `${baseDir}/
matched 2 field(s)
${filePath}:  spec.replicas: 4
${filePath}:  spec.replicas: 4
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 4
 `,
		},
		{
			name: "search replace multiple deployments different value",
			args: []string{"--by-value", "3", "--put-literal", "4"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 5
 `,
			out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.replicas: 4
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 5
 `,
		},
		{
			name: "search by regex",
			args: []string{"--by-value-regex", "nginx-*"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			out: `${baseDir}/
matched 1 field(s)
${filePath}:  metadata.name: nginx-deployment
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
		},
		{
			name: "search replace by regex",
			args: []string{"--by-value-regex", "nginx-*", "--put-literal", "ubuntu-deployment"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			out: `${baseDir}/
matched 1 field(s)
${filePath}:  metadata.name: ubuntu-deployment
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ubuntu-deployment
spec:
  replicas: 3
 `,
		},
		{
			name: "search by path",
			args: []string{"--by-path", "spec.replicas"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
			out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.replicas: 3
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
		},
		{
			name: "replace by path and value",
			args: []string{"--by-path", "spec.replicas", "--by-value", "3", "--put-literal", "4"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
			out: `${baseDir}/
matched 1 field(s)
${filePath}:  spec.replicas: 4
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4
  foo: 3
---
apiVersion: apps/v1
kind: Service
metadata:
  name: nginx-service
 `,
		},
		{
			name: "add non-existing field",
			args: []string{"--by-path", "metadata.namespace", "--put-literal", "myspace"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			out: `${baseDir}/
matched 1 field(s)
${filePath}:  metadata.namespace: myspace
`,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: myspace
spec:
  replicas: 3
 `,
		},
		{
			name: "put literal error",
			args: []string{"--put-literal", "something"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			errMsg: `at least one of ["by-value", "by-value-regex", "by-path"] must be provided`,
		},
		{
			name: "error when both by-value and by-regex provided",
			args: []string{"--by-value", "nginx-deployment", "--by-value-regex", "nginx-*"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			errMsg: `only one of ["by-value", "by-value-regex"] can be provided`,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			// reset the openAPI afterward

			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)

			r, err := ioutil.TempFile(baseDir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.Remove(r.Name())
			err = ioutil.WriteFile(r.Name(), []byte(test.input), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			runner := NewSearchRunner("")
			out := &bytes.Buffer{}
			runner.Command.SetOut(out)
			runner.Command.SetArgs(append([]string{baseDir}, test.args...))
			err = runner.Command.Execute()
			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}

			// normalize path format for windows
			actualNormalized := strings.ReplaceAll(
				strings.ReplaceAll(out.String(), "\\", "/"),
				"//", "/")

			expected := strings.ReplaceAll(test.out, "${baseDir}", baseDir)
			expected = strings.ReplaceAll(expected, "${filePath}", filepath.Base(r.Name()))
			expectedNormalized := strings.ReplaceAll(
				strings.ReplaceAll(expected, "\\", "/"),
				"//", "/")

			if test.errMsg == "" && !assert.Equal(t, expectedNormalized, actualNormalized) {
				t.FailNow()
			}

			actualResources, err := ioutil.ReadFile(r.Name())
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				strings.TrimSpace(test.expectedResources),
				strings.TrimSpace(string(actualResources))) {
				t.FailNow()
			}
		})
	}
}

func TestSearchSubPackages(t *testing.T) {
	var tests = []struct {
		name    string
		dataset string
		args    []string
		out     string
		errMsg  string
	}{
		{
			name:    "search-replace-recurse-subpackages",
			dataset: "dataset-with-autosetters",
			args:    []string{"--by-value", "myspace", "--put-literal", "otherspace"},
			out: `${baseDir}/
matched 0 field(s)

${baseDir}/mysql/
matched 1 field(s)
deployment.yaml:  metadata.namespace: otherspace

${baseDir}/mysql/nosetters/
matched 1 field(s)
deployment.yaml:  metadata.namespace: otherspace

${baseDir}/mysql/storage/
matched 1 field(s)
deployment.yaml:  metadata.namespace: otherspace
`,
		},
		{
			name:    "search-recurse-subpackages",
			dataset: "dataset-with-autosetters",
			args:    []string{"--by-value", "mysql"},
			out: `${baseDir}/
matched 0 field(s)

${baseDir}/mysql/
matched 1 field(s)
deployment.yaml:  spec.template.spec.containers.name: mysql

${baseDir}/mysql/nosetters/
matched 0 field(s)

${baseDir}/mysql/storage/
matched 0 field(s)
`,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			testDataDir := filepath.Join("../", "testutil", "testdata")
			sourceDir := filepath.Join(testDataDir, test.dataset)
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = copyutil.CopyDir(sourceDir, baseDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			ext.KRMFileName = func() string {
				return "Kptfile"
			}
			defer os.RemoveAll(baseDir)
			runner := NewSearchRunner("")
			out := &bytes.Buffer{}
			runner.Command.SetOut(out)
			runner.Command.SetArgs(append([]string{baseDir}, test.args...))
			err = runner.Command.Execute()
			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}

			// normalize path format for windows
			actualNormalized := strings.ReplaceAll(
				strings.ReplaceAll(out.String(), "\\", "/"),
				"//", "/")

			expected := strings.ReplaceAll(test.out, "${baseDir}", baseDir)
			expectedNormalized := strings.ReplaceAll(
				strings.ReplaceAll(expected, "\\", "/"),
				"//", "/")

			if test.errMsg == "" && !assert.Equal(t, expectedNormalized, actualNormalized) {
				t.FailNow()
			}
		})
	}
}
