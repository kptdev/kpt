//go:build docker

// Copyright 2021,2026 The kpt Authors
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

package runtime

import (
	"bytes"
	"context"
	"testing"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
)

func TestContainerFn(t *testing.T) {
	var tests = []struct {
		input  string
		image  string
		output string
		name   string
		err    bool
	}{
		{
			name:  "no-op function",
			image: "ghcr.io/kptdev/krm-functions-catalog/no-op:latest",
			input: `apiVersion: config.kubernetes.io/v1
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
			output: `apiVersion: config.kubernetes.io/v1
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
			name:  "non-existing image",
			image: "foobar",
			err:   true,
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			errBuff := &bytes.Buffer{}
			instance := ContainerFn{
				Ctx:   printer.WithContext(ctx, printer.New(nil, errBuff)),
				Image: tt.image,
				FnResult: &fnresultv1.Result{
					Image: tt.image,
				},
			}
			input := bytes.NewBufferString(tt.input)
			output := &bytes.Buffer{}
			err := instance.Run(input, output)
			if tt.err && !assert.Error(t, err) {
				t.FailNow()
			}
			if !tt.err && !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t, tt.output, output.String()) {
				t.FailNow()
			}
		})
	}
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
			errMsg: "docker client version must be v20.10.0 or greater: found 20.9.1",
		},
		{
			name:   "invalid semver",
			inputV: "20..12.1",
			errMsg: "docker client version must be v20.10.0 or greater: found invalid version 20..12.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isSupportedDockerVersion(tt.inputV)
			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDockerCLIOutputFilter(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "should filter docker CLI output successfully",
			input: `Unable to find image 'ghcr.io/kptdev/krm-functions-catalog/starlark:latest' locally
v0.3: Pulling from ghcr.io/kptdev/krm-functions-catalog/starlark
4e9f2cdf4387: Already exists
aafbf7df3ddf: Pulling fs layer
aafbf7df3ddf: Verifying Checksum
aafbf7df3ddf: Download complete
aafbf7df3ddf: Pull complete
6b759ab96cb2: Waiting
Digest: sha256:c347e28606fa1a608e8e02e03541a5a46e4a0152005df4a11e44f6c4ab1edd9a
Status: Downloaded newer image for ghcr.io/kptdev/krm-functions-catalog/starlark:latest
`,
			expected: "",
		},
		{
			name: "should filter docker messages and shouldn't truncate trailing lines",
			input: `Unable to find image 'ghcr.io/kptdev/krm-functions-catalog/starlark:latest' locally
v0.3: Pulling from ghcr.io/kptdev/krm-functions-catalog/starlark
4e9f2cdf4387: Already exists
aafbf7df3ddf: Pulling fs layer
aafbf7df3ddf: Verifying Checksum
aafbf7df3ddf: Download complete
aafbf7df3ddf: Pull complete
6b759ab96cb2: Waiting
Digest: sha256:c347e28606fa1a608e8e02e03541a5a46e4a0152005df4a11e44f6c4ab1edd9a
Status: Downloaded newer image for ghcr.io/kptdev/krm-functions-catalog/starlark:latest
line before last line
lastline

`,
			expected: `line before last line
lastline
`,
		},
		{
			name: "should filter interleaved docker messages",
			input: `firstline
Unable to find image 'ghcr.io/kptdev/krm-functions-catalog/starlark:latest' locally
v0.3: Pulling from ghcr.io/kptdev/krm-functions-catalog/starlark
4e9f2cdf4387: Already exists
aafbf7df3ddf: Pulling fs layer
aafbf7df3ddf: Verifying Checksum
line in the middle
aafbf7df3ddf: Download complete
aafbf7df3ddf: Pull complete
6b759ab96cb2: Waiting
Digest: sha256:c347e28606fa1a608e8e02e03541a5a46e4a0152005df4a11e44f6c4ab1edd9a
Status: Downloaded newer image for ghcr.io/kptdev/krm-functions-catalog/starlark:latest
lastline
`,
			expected: `firstline
line in the middle
lastline`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s := filterDockerCLIOutput(bytes.NewBufferString(tc.input))
			assert.Equal(t, tc.expected, s)
		})
	}
}
