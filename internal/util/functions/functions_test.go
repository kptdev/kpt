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

	"github.com/GoogleContainerTools/kpt/internal/util/functions"
	"github.com/stretchr/testify/assert"
)

var tests = []testCase{
	// Test 1
	{
		name: "starlarkFunctions",
		inputs: func(s string) map[string]string {
			return map[string]string{
				"Kptfile": fmt.Sprintf(`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: my-pkg
functions:
  starlarkFunctions:
  - name: func
    path: %s
`, filepath.Join(s, "reconcile.star")),

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

run(resourceList["items"])
`,
			}
		},
		outputs: func(s string) map[string]string {
			return map[string]string{
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
			}
		},
	},

	// Test 2
	{
		name: "autoRunStarlark",
		inputs: func(s string) map[string]string {
			return map[string]string{
				"Kptfile": `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: my-pkg
functions:
  autoRunStarlark: true`,

				"deploy1.yaml": fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    config.kubernetes.io/function: |
      starlark:
        path: %s
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`, filepath.Join(s, "reconcile.star")),

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

run(resourceList["items"])
`,
			}
		},
		outputs: func(s string) map[string]string {
			return map[string]string{
				"deploy1.yaml": fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    config.kubernetes.io/function: |
      starlark:
        path: %s
    foo: bar
spec:
  template:
    spec:
      containers:
      - name: nginx
        # head comment
        image: nginx:1.8.1 # {"$ref": "#/definitions/io.k8s.cli.substitutions.image"}
`, filepath.Join(s, "reconcile.star")),

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
			}
		},
	},
}

type testCase struct {
	name    string
	inputs  func(string) map[string]string
	outputs func(string) map[string]string
}

func TestReconcileFunctions(t *testing.T) {
	t.SkipNow() //TODO: this test is failing due to changes in downstream, identify and fix it
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			d, err := ioutil.TempDir("", "kpt")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(d)

			for filename, value := range test.inputs(d) {
				err = ioutil.WriteFile(
					filepath.Join(d, filename), []byte(value), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

			err = functions.ReconcileFunctions(d)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			for filename, expected := range test.outputs(d) {
				actual, err := ioutil.ReadFile(filepath.Join(d, filename))
				if !assert.NoError(t, err) {
					t.FailNow()
				}

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
