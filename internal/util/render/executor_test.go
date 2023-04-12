// Copyright 2022 The kpt Authors
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

package render

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestPathRelToRoot(t *testing.T) {
	tests := []struct {
		name         string
		rootPath     string
		subPkgPath   string
		resourcePath string
		expected     string
		errString    string
	}{
		{
			name:         "root package with non absolute path",
			rootPath:     "tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("root package path %q must be absolute", "tmp"),
		},
		{
			name:         "subpackage with non absolute path",
			rootPath:     "/tmp",
			subPkgPath:   "tmp/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("subpackage path %q must be absolute", "tmp/a"),
		},
		{
			name:         "resource in a subpackage",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "c.yaml",
			expected:     "a/c.yaml",
		},
		{
			name:         "resource exists in a deeply nested subpackage",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a/b/c",
			resourcePath: "c.yaml",
			expected:     "a/b/c/c.yaml",
		},
		{
			name:         "resource exists in a sub dir with same name as sub package",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "a/c.yaml",
			expected:     "a/a/c.yaml",
		},
		{
			name:         "subpackage is not a descendant of root package",
			rootPath:     "/tmp",
			subPkgPath:   "/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("subpackage %q is not a descendant of %q", "/a", "/tmp"),
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			newPath, err := pathRelToRoot(tc.rootPath,
				tc.subPkgPath, tc.resourcePath)
			assert.Equal(t, newPath, tc.expected)
			if tc.errString != "" {
				assert.Contains(t, err.Error(), tc.errString)
			}
		})
	}
}

func TestMergeWithInput(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		selectedInput string
		output        string
		expected      string
	}{
		{
			name: "simple input",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selectedInput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			output: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3
`,
		},
		{
			name: "complex example with generation, transformation and deletion of resource",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    internal.config.k8s.io/kpt-resource-id: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
  annotations:
    internal.config.k8s.io/kpt-resource-id: "2"
`,
			selectedInput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    internal.config.k8s.io/kpt-resource-id: "1"
`,
			output: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  namespace: staging # transformed
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1 # generated resource
kind: Deployment
metadata:
  name: nginx-deployment-3
`,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  namespace: staging # transformed
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
  annotations:
    internal.config.k8s.io/kpt-resource-id: "2"
---
apiVersion: apps/v1 # generated resource
kind: Deployment
metadata:
  name: nginx-deployment-3
`,
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			output, err := kio.ParseAll(tc.output)
			assert.NoError(t, err)
			selectedInput, err := kio.ParseAll(tc.selectedInput)
			assert.NoError(t, err)
			input, err := kio.ParseAll(tc.input)
			assert.NoError(t, err)
			result := fnruntime.MergeWithInput(output, selectedInput, input)
			actual, err := kio.StringAll(result)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
