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

package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNewPkg(t *testing.T) {
	var tests = []struct {
		name        string
		inputPath   string
		uniquePath  string
		displayPath string
	}{
		{
			name:        "test1",
			inputPath:   ".",
			displayPath: ".",
		},
		{
			name:        "test2",
			inputPath:   "../",
			displayPath: "..",
		},
		{
			name:        "test3",
			inputPath:   "./foo/bar/",
			displayPath: "foo/bar",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			p, err := New(test.inputPath)
			assert.NoError(t, err)
			assert.Equal(t, test.displayPath, string(p.DisplayPath))
		})
	}
}

func TestFilterMetaResources(t *testing.T) {
	tests := map[string]struct {
		resources []string
		expected  []string
	}{
		"no resources": {
			resources: nil,
			expected:  nil,
		},

		"nothing to filter": {
			resources: []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}`,
				`
apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3`,
			},
			expected: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
`,
				`apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3
`,
			},
		},

		"filter out metadata": {
			resources: []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}`,
				`
apiVersion: config.kpt.dev/v1
Kind: FunctionPermission
Metadata:
  Name: functionPermission
Spec:
  Allow:
  - imageName: gcr.io/my-project/*…..
  Permissions:
  - network
  - mount
  Disallow:
  - Name: gcr.io/my-project/*`,
				`
apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3`,
				`
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: mysql
setterDefinitions:
  replicas:
    description: "replica setter"
    type: integer
setterValues:
  replicas: 5`,
				`
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
sources:
  - "."`,
			},
			expected: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
`,
				`apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3
`,
			},
		},
	}

	for name := range tests {
		test := tests[name]
		t.Run(name, func(t *testing.T) {
			var nodes []*yaml.RNode

			for _, r := range test.resources {
				res, err := yaml.Parse(r)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				nodes = append(nodes, res)
			}

			filteredRes, err := filterMetaResources(nodes, nil)
			if err != nil {
				t.Errorf("unexpected error in filtering meta resources: %v", err)
			}
			if len(filteredRes) != len(test.expected) {
				t.Fatal("length of filtered resources not equal to expected")
			}

			for i, r := range filteredRes {
				res, err := r.String()
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, test.expected[i], res)
			}
		})
	}
}
