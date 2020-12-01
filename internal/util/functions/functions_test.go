// Copyright 2019 Google LLC
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

package functions_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/util/functions"
	"github.com/stretchr/testify/assert"
)

var tests = []testCase{
	// Test 1
	{
		name: "starlarkFunctions",
		inputs: map[string]string{
			"Kptfile": `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: my-pkg
functions:
  starlarkFunctions:
  - name: func
    path: reconcile.star
`,

			"deploy.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			"reconcile.star": `
# set the foo annotation on each resource
def run(r):
  for resource in r:
    resource["metadata"]["annotations"]["foo"] = "bar"

run(ctx.resource_list["items"])
`,
		},
		outputs: map[string]string{
			"deploy.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    foo: bar
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,
		},
	},

	// Test 1
	{
		name:   "starlarkFunctions-nested",
		parent: "a",
		inputs: map[string]string{
			"Kptfile": `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: my-pkg
functions:
  starlarkFunctions:
  - name: func
    path: reconcile.star
`,

			"deploy.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			"reconcile.star": `
# set the foo annotation on each resource
def run(r):
  for resource in r:
    resource["metadata"]["annotations"]["foo"] = "bar"

run(ctx.resource_list["items"])
`,
		},
		outputs: map[string]string{
			"deploy.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    foo: bar
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,
		},
	},

	// Test 2
	{
		name: "autoRunStarlark",
		inputs: map[string]string{
			filepath.Join("Kptfile"): `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: my-pkg
functions:
  autoRunStarlark: true`,

			"deploy1.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    config.kubernetes.io/function: |
      starlark:
        path: "pkg/reconcile.star"
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			"deploy2.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			filepath.Join("pkg", "reconcile.star"): `
# set the foo annotation on each resource
def run(r):
  for resource in r:
    resource["metadata"]["annotations"]["foo"] = "bar"

run(ctx.resource_list["items"])
`,
		},
		outputs: map[string]string{
			"deploy1.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    config.kubernetes.io/function: |
      starlark:
        path: "pkg/reconcile.star"
    foo: bar
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			"deploy2.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
  annotations:
    foo: bar
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,
		},
	},

	// Test 2
	{
		name: "relative",
		inputs: map[string]string{
			filepath.Join("Kptfile"): `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: my-pkg
functions:
  autoRunStarlark: true`,

			"deploy1.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    config.kubernetes.io/function: |
      starlark:
        path: "../reconcile.star"
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			"deploy2.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			"reconcile.star": `
# set the foo annotation on each resource
def run(r):
  for resource in r:
    resource["metadata"]["annotations"]["foo"] = "bar"

run(ctx.resource_list["items"])
`,
		},
		outputs: map[string]string{
			"deploy1.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    config.kubernetes.io/function: |
      starlark:
        path: "../reconcile.star"
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,

			"deploy2.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`,
		},
		err: "function path ../reconcile.star not allowed to start with ../",
	},
}

type testCase struct {
	name    string
	inputs  map[string]string
	outputs map[string]string
	err     string
	parent  string
}

var relativePathValues = []bool{true, false}

func TestReconcileFunctions(t *testing.T) {
	for i := range tests {
		test := tests[i]
		for j := range relativePathValues {
			relativePath := relativePathValues[j]
			name := fmt.Sprintf("%s/relative-%v", test.name, relativePath)
			t.Run(name, func(t *testing.T) {
				d, err := ioutil.TempDir("", "kpt")
				testutil.AssertNoError(t, err)
				defer os.RemoveAll(d)
				testutil.AssertNoError(t, os.Chdir(d))

				for filename, value := range test.inputs {
					abs := filepath.Join(d, test.parent, filename)
					err = os.MkdirAll(filepath.Dir(abs), 0700)
					testutil.AssertNoError(t, err)
					err = ioutil.WriteFile(abs, []byte(value), 0600)
					testutil.AssertNoError(t, err)
				}

				var path string
				if relativePath {
					path = "."
				} else {
					path = d
				}
				path = filepath.Join(path, test.parent)
				err = functions.ReconcileFunctions(path)
				if test.err == "" {
					testutil.AssertNoError(t, err)
				} else if !assert.EqualError(t, err, test.err) {
					t.FailNow()
				}

				for filename, expected := range test.outputs {
					abs := filepath.Join(d, test.parent, filename)
					actual, err := ioutil.ReadFile(abs)
					testutil.AssertNoError(t, err)

					if !assert.Equal(t,
						strings.TrimSpace(expected),
						strings.TrimSpace(string(actual)),
					) {
						t.FailNow()
					}
				}
			})
		}
	}
}
