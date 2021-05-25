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

// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package fnruntime

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1alpha2"

	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestFunctionConfig(t *testing.T) {
	type input struct {
		name              string
		fn                kptfilev1alpha2.Function
		configFileContent string
		expected          string
	}

	cases := []input{
		{
			name:     "no config",
			fn:       kptfilev1alpha2.Function{},
			expected: "",
		},
		{
			name: "inline config",
			fn: kptfilev1alpha2.Function{
				Config: *yaml.MustParse(`apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy`).YNode(),
			},
			expected: `apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy
`,
		},
		{
			name: "file config",
			fn:   kptfilev1alpha2.Function{},
			configFileContent: `apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy`,
			expected: `apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy
`,
		},
		{
			name: "map config",
			fn: kptfilev1alpha2.Function{
				ConfigMap: map[string]string{
					"foo": "bar",
				},
			},
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {foo: bar}
`,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			if c.configFileContent != "" {
				tmp, err := ioutil.TempFile("", "kpt-pipeline-*")
				assert.NoError(t, err, "unexpected error")
				_, err = tmp.WriteString(c.configFileContent)
				assert.NoError(t, err, "unexpected error")
				c.fn.ConfigPath = path.Base(tmp.Name())
			}
			cn, err := newFnConfig(&c.fn, types.UniquePath(os.TempDir()))
			assert.NoError(t, err, "unexpected error")
			actual, err := cn.String()
			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, c.expected, actual, "unexpected result")
		})
	}
}

func TestMultilineFormatter(t *testing.T) {

	type testcase struct {
		ml       *multiLineFormatter
		expected string
	}

	testcases := map[string]testcase{
		"multiline should format lines and truncate": {
			ml: &multiLineFormatter{
				Title: "Results",
				Lines: []string{
					"line-1",
					"line-2",
					"line-3",
					"line-4",
					"line-5",
				},
				MaxLines:       3,
				TruncateOutput: true,
			},
			expected: `  Results:
    line-1
    line-2
    line-3
    ...(2 line(s) truncated, use '--truncate-output=false' to disable)
`,
		},
		"multiline should format without truncate": {
			ml: &multiLineFormatter{
				Title: "Results",
				Lines: []string{
					"line-1",
					"line-2",
					"line-3",
					"line-4",
					"line-5",
				},
			},
			expected: `  Results:
    line-1
    line-2
    line-3
    line-4
    line-5
`,
		},
	}
	for name, c := range testcases {
		c := c
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, c.ml.String())
		})
	}
}

func TestEnforcePathInvariants(t *testing.T) {
	tests := map[string]struct {
		input       string // input
		expectedErr string // expected result
	}{
		"duplicate": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
`,
			expectedErr: `resource at path "my/path/custom.yaml" and index "0" already exists`,
		},
		"duplicate with `./` prefix": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: './my/path/custom.yaml'
    config.kubernetes.io/index: '0'
`,
			expectedErr: `resource at path "my/path/custom.yaml" and index "0" already exists`,
		},
		"duplicate path, not index": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/custom.yaml'
    config.kubernetes.io/index: '1'
`,
		},
		"duplicate index, not path": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '0'
`,
		},
		"larger number of resources with duplicate": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '2'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '1'
`,
			expectedErr: `resource at path "my/path/b.yaml" and index "1" already exists`,
		},
		"larger number of resources without duplicates": {
			input: `apiVersion: v1
kind: Custom
metadata:
  name: a
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/a.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '2'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '0'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/c.yaml'
    config.kubernetes.io/index: '1'
---
apiVersion: v1
kind: Custom
metadata:
  name: b
  annotations:
    config.kubernetes.io/path: 'my/path/b.yaml'
    config.kubernetes.io/index: '3'
`,
		},

		"no error": {
			input: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-stateful-set
  annotations:
    config.kubernetes.io/path: my-stateful-set.yaml
spec:
  replicas: 3
`,
		},
		"with ../ prefix": {
			input: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-stateful-set
  annotations:
    config.kubernetes.io/path: ../my-stateful-set.yaml
spec:
  replicas: 3

`,
			expectedErr: "function must not modify resources outside of package: resource has path ../my-stateful-set.yaml",
		},
		"with nested ../ in path": {
			input: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-stateful-set
  annotations:
    config.kubernetes.io/path: a/b/../../../my-stateful-set.yaml
spec:
  replicas: 3
`,
			expectedErr: "function must not modify resources outside of package: resource has path a/b/../../../my-stateful-set.yaml",
		},
	}
	for _, tc := range tests {
		out := &bytes.Buffer{}
		r := kio.ByteReadWriter{
			Reader:                bytes.NewBufferString(tc.input),
			Writer:                out,
			KeepReaderAnnotations: true,
			OmitReaderAnnotations: true,
		}
		n, err := r.Read()
		if err != nil {
			t.FailNow()
		}
		err = enforcePathInvariants(n)
		if err != nil && tc.expectedErr == "" {
			t.Errorf("unexpected error %s", err.Error())
			t.FailNow()
		}
		if tc.expectedErr != "" && err == nil {
			t.Errorf("expected error %s", tc.expectedErr)
			t.FailNow()
		}
		if tc.expectedErr != "" && !strings.Contains(err.Error(), tc.expectedErr) {
			t.Errorf("wanted error %s, got %s", tc.expectedErr, err.Error())
			t.FailNow()
		}
	}
}

func TestGetResourceRefMetadata(t *testing.T) {
	tests := map[string]struct {
		input    string // input
		expected string // expected result
	}{
		"new format with name": {
			input: `
message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
field:
    path: selector
file:
    path: resources.yaml
`,
			expected: `message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
field:
    path: selector
file:
    path: resources.yaml
`,
		},
		"new format with namespace": {
			input: `
message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
    namespace: my-namespace
field:
    path: selector
file:
    path: resources.yaml
`,
			expected: `message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
    namespace: my-namespace
field:
    path: selector
file:
    path: resources.yaml
`,
		},
		"old format with name": {
			input: `
message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
field:
    path: selector
file:
    path: resources.yaml
`,
			expected: `message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
field:
    path: selector
file:
    path: resources.yaml
`,
		},
		"old format with namespace": {
			input: `
message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      namespace: my-namespace
field:
    path: selector
file:
    path: resources.yaml
`,
			expected: `message: selector is required
severity: error
resourceRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-deployment
    namespace: my-namespace
field:
    path: selector
file:
    path: resources.yaml
`,
		},
		"no resourceRef": {
			input: `
message: selector is required
severity: error
field:
    path: selector
file:
    path: resources.yaml
`,
			expected: `message: selector is required
severity: error
field:
    path: selector
file:
    path: resources.yaml
`,
		},
	}
	for _, tc := range tests {
		yml, err := yaml.Parse(tc.input)
		assert.NoError(t, err)

		result := &fnresult.ResultItem{}
		err = yaml.Unmarshal([]byte(tc.input), result)
		assert.NoError(t, err)
		assert.NoError(t, populateResourceRef(yml, result))

		out, err := yaml.Marshal(result)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, string(out))
	}
}
