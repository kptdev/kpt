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
	"testing"

	"github.com/stretchr/testify/assert"
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
			content: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
`,
			expectedStdout: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
`,
		},
		{
			name:   "unwrapped output to stdout",
			dest:   "unwrap",
			writer: bytes.Buffer{},
			content: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
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
			content: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
`,
			expectedPkg: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    config.kubernetes.io/path: 'foo/bar/deployment.yaml'
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
  annotations:
    config.kubernetes.io/path: 'foo/bar/svc.yaml'
`,
		},
		{
			name:      "wrapped output to stdout by default if input is from stdin",
			fromStdin: true,
			writer:    bytes.Buffer{},
			content: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
`,
			expectedStdout: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
`,
		},
		{
			name:      "unwrapped output to stdout for input from stdin",
			fromStdin: true,
			dest:      "unwrap",
			writer:    bytes.Buffer{},
			content: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
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
			content: `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'deployment.yaml'
  - apiVersion: v1
    kind: Service
    metadata:
      name: nginx-svc
      annotations:
        config.kubernetes.io/index: '0'
        config.kubernetes.io/path: 'svc.yaml'
`,
			expectedPkg: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    config.kubernetes.io/path: 'foo/bar/deployment.yaml'
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
  annotations:
    config.kubernetes.io/path: 'foo/bar/svc.yaml'
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
