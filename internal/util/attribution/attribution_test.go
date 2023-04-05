// Copyright 2021 The kpt Authors
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

package attribution

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsageProcess(t *testing.T) {
	var tests = []struct {
		name     string
		input    string
		group    string
		disable  bool
		expected string
		errMsg   string
	}{
		{
			name: "Don't add to non-cnrm resource",
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: my-space
spec:
  replicas: 3
 `,
			expected: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: my-space
spec:
  replicas: 3
 `,
		},
		{
			name: "Create metrics annotation",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
 `,
			group: "pkg",
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: 'kpt-pkg'
 `,
		},
		{
			name: "Create metrics annotation",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
 `,
			group:   "pkg",
			disable: true,
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
 `,
		},
		{
			name: "Add new group to existing metrics annotation",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: 'kpt-pkg'
 `,
			group: "fn",
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: 'kpt-pkg-fn'
 `,
		},
		{
			name: "Add new group to existing metrics annotation 2",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: 'kpt-pkg-fn'
 `,
			group: "live",
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: 'kpt-pkg-fn-live'
 `,
		},
		{
			name: "no-op if group is already present",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: 'kpt-pkg-fn-live'
 `,
			group: "fn",
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: 'kpt-pkg-fn-live'
 `,
		},
		{
			name: "add kpt prefix to existing annotation",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: cnrm/landing-zone:networking/v0.4.0
 `,
			group: "fn",
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: cnrm/landing-zone:networking/v0.4.0,kpt-fn
 `,
		},
		{
			name: "add group for existing annotation and existing kpt suffix",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: cnrm/landing-zone:networking/v0.4.0,kpt-fn
 `,
			group: "pkg",
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: cnrm/landing-zone:networking/v0.4.0,kpt-pkg-fn
 `,
		},
		{
			name: "add group for existing annotation and existing kpt substring",
			input: `
apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: cnrm/landing-zone:networking/v0.4.0,kpt-fn,blueprints_controller
 `,
			group: "pkg",
			expected: `apiVersion: compute.cnrm.cloud.google.com/v1beta1
kind: ComputeSubnetwork
metadata:
  name: network-name-subnetwork
  annotations:
    cnrm.cloud.google.com/blueprint: cnrm/landing-zone:networking/v0.4.0,kpt-pkg-fn,blueprints_controller
 `,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			baseDir := t.TempDir()

			r, err := os.CreateTemp(baseDir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = os.WriteFile(r.Name(), []byte(test.input), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if test.disable {
				err = os.Setenv("KPT_DISABLE_ATTRIBUTION", "true")
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				defer os.Setenv("KPT_DISABLE_ATTRIBUTION", "")
			}

			a := Attributor{PackagePaths: []string{baseDir}, CmdGroup: test.group}
			a.Process()
			actualResources, err := os.ReadFile(r.Name())
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				strings.TrimSpace(test.expected),
				strings.TrimSpace(string(actualResources))) {
				t.FailNow()
			}
		})
	}
}
