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

package cmdutil

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestWriteFnOutput(t *testing.T) {
	var tests = []struct {
		name           string
		dest           string
		content        string
		fromStdin      bool
		writer         bytes.Buffer
		expectedStdout string
		expectedPkg    string
	}{
		{
			name:   "wrapped output to stdout",
			dest:   "stdout",
			writer: bytes.Buffer{},
			content: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
			expectedStdout: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
		},
		{
			name:   "unwrapped output to stdout",
			dest:   "unwrap",
			writer: bytes.Buffer{},
			content: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
			expectedStdout: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
`,
		},
		{
			name: "output to another directory",
			dest: "foo/bar",
			content: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
			expectedPkg: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    config.kubernetes.io/path: 'foo/bar/deployment.yaml'
    internal.config.kubernetes.io/path: 'foo/bar/deployment.yaml'
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
  annotations:
    config.kubernetes.io/path: 'foo/bar/svc.yaml'
    internal.config.kubernetes.io/path: 'foo/bar/svc.yaml'
`,
		},
		{
			name:      "wrapped output to stdout by default if input is from stdin",
			fromStdin: true,
			writer:    bytes.Buffer{},
			content: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
			expectedStdout: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
		},
		{
			name:      "unwrapped output to stdout for input from stdin",
			fromStdin: true,
			dest:      "unwrap",
			writer:    bytes.Buffer{},
			content: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
			expectedStdout: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
`,
		},
		{
			name:      "output to directory for input from stdin",
			fromStdin: true,
			dest:      "foo/bar",
			content: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'svc.yaml'
`,
			expectedPkg: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    config.kubernetes.io/path: 'foo/bar/deployment.yaml'
    internal.config.kubernetes.io/path: 'foo/bar/deployment.yaml'
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
  annotations:
    config.kubernetes.io/path: 'foo/bar/svc.yaml'
    internal.config.kubernetes.io/path: 'foo/bar/svc.yaml'
`,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)

			if test.dest != "" && test.dest != Stdout && test.dest != Unwrap {
				test.dest = filepath.Join(baseDir, test.dest)
			}

			// this method should create a directory and write the output if the dest is a directory path
			err = WriteFnOutput(test.dest, test.content, test.fromStdin, &test.writer)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			actualStdout := test.writer.String()
			if !assert.Equal(t, test.expectedStdout, actualStdout) {
				t.FailNow()
			}

			// read the resources from output dir
			in := &kio.LocalPackageReader{
				PackagePath:       baseDir,
				PreserveSeqIndent: true,
				WrapBareSeqNode:   true,
			}
			out := &bytes.Buffer{}

			err = kio.Pipeline{
				Inputs:  []kio.Reader{in},
				Outputs: []kio.Writer{&kio.ByteWriter{Writer: out}},
			}.Execute()

			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// verify that the resources in the output dir are as expected
			if !assert.Equal(t, test.expectedPkg, out.String()) {
				t.FailNow()
			}
		})
	}
}

func TestListImages(t *testing.T) {
	result := listImages(`{
  "apply-setters": {
    "v0.1": {
      "LatestPatchVersion": "v0.1.1",
      "Examples": {
        "apply-setters-simple": {
          "LocalExamplePath": "/apply-setters/v0.1/apply-setters-simple",
          "RemoteExamplePath": "https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/apply-setters/v0.1/examples/apply-setters-simple",
          "RemoteSourcePath": "https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/apply-setters/v0.1/functions/go/apply-setters"
        }
      }
    }
  },
  "gatekeeper": {
    "v0.1": {
      "LatestPatchVersion": "v0.1.3",
      "Examples": {
        "gatekeeper-warning-only": {
          "LocalExamplePath": "/gatekeeper/v0.1/gatekeeper-warning-only",
          "RemoteExamplePath": "https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/gatekeeper/v0.1/examples/gatekeeper-warning-only",
          "RemoteSourcePath": "https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/gatekeeper/v0.1/functions/go/gatekeeper"
        }
      }
    },
    "v0.2": {
      "LatestPatchVersion": "v0.2.1",
      "Examples": {
        "gatekeeper-warning-only": {
          "LocalExamplePath": "/gatekeeper/v0.2/gatekeeper-warning-only",
          "RemoteExamplePath": "https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/gatekeeper/v0.2/examples/gatekeeper-warning-only",
          "RemoteSourcePath": "https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/gatekeeper/v0.2/functions/go/gatekeeper"
        }
      }
    }
  }
}`)
	sort.Strings(result)
	assert.Equal(t, []string{"apply-setters:v0.1.1", "gatekeeper:v0.2.1"}, result)
}

func TestIsSupportedDockerVersion(t *testing.T) {
	tests := []struct {
		name   string
		inputV string
		errMsg string
	}{
		{
			name:   "greater than min version",
			inputV: "20.10.1",
		},
		{
			name:   "equal to min version",
			inputV: "20.10.0",
		},
		{
			name:   "less than min version",
			inputV: "20.9.1",
			errMsg: "docker client version must be v20.10.0 or greater: found v20.9.1",
		},
		{
			name:   "invalid semver",
			inputV: "20..12.1",
			errMsg: "docker client version must be v20.10.0 or greater: found invalid version v20..12.1",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			err := isSupportedDockerVersion(tt.inputV)
			if tt.errMsg != "" {
				require.NotNil(err)
				require.Contains(err.Error(), tt.errMsg)
			} else {
				require.NoError(err)
			}
		})
	}
}
