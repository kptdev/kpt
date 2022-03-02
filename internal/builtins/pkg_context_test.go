// Copyright 2022 Google LLC
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

package builtins

import (
	"bytes"
	"testing"
)

type test struct {
	name   string
	in     string
	exp    string
	expErr error
}

func TestPkgContextGenerator(t *testing.T) {

	tests := []test{
		{
			name: "pkg context should succeed on a non-nested package",
			in: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: kpt.dev/v1
    kind: Kptfile
    metadata:
      name: order-service
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'Kptfile'
        internal.config.kubernetes.io/seqindent: 'compact'
  - apiVersion: v1
    kind: Namespace
    metadata:
      name: example-ns
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'ns.yaml'
        internal.config.kubernetes.io/seqindent: 'compact'
`,
			exp: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: order-service
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'Kptfile'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: v1
  kind: Namespace
  metadata:
    name: example-ns
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'ns.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: kptfile.kpt.dev
    annotations:
      config.kubernetes.io/local-config: "true"
      internal.config.kubernetes.io/path: 'package-context.yaml'
  data:
    name: order-service
results:
- message: generated package context
  severity: info
  file:
    path: package-context.yaml
`,
		},
		{
			name: "pkg context should generate on a non-nested package with existing package context",
			in: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: kpt.dev/v1
    kind: Kptfile
    metadata:
      name: order-service
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'Kptfile'
        internal.config.kubernetes.io/seqindent: 'compact'
  - apiVersion: v1
    kind: Namespace
    metadata:
      name: example-ns
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'ns.yaml'
        internal.config.kubernetes.io/seqindent: 'compact'
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: kptfile.kpt.dev
      annotations:
        config.kubernetes.io/local-config: "true"
        internal.config.kubernetes.io/path: 'package-context.yaml'
    data:
      name: order-service
`,
			exp: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: order-service
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'Kptfile'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: v1
  kind: Namespace
  metadata:
    name: example-ns
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'ns.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: kptfile.kpt.dev
    annotations:
      config.kubernetes.io/local-config: "true"
      internal.config.kubernetes.io/path: 'package-context.yaml'
  data:
    name: order-service
results:
- message: generated package context
  severity: info
  file:
    path: package-context.yaml
`,
		},
		{
			name: "pkg context should succeed on package with nested package",
			in: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
  - apiVersion: kpt.dev/v1
    kind: Kptfile
    metadata:
      name: order-service
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'Kptfile'
        internal.config.kubernetes.io/seqindent: 'compact'
  - apiVersion: v1
    kind: Namespace
    metadata:
      name: example-ns
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'ns.yaml'
        internal.config.kubernetes.io/seqindent: 'compact'
  - apiVersion: kpt.dev/v1
    kind: Kptfile
    metadata:
      name: subpkg
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'subpkg/Kptfile'
        internal.config.kubernetes.io/seqindent: 'compact'
  - apiVersion: v1
    kind: Namespace
    metadata:
      name: example-ns
      annotations:
        internal.config.kubernetes.io/index: '0'
        internal.config.kubernetes.io/path: 'subpkg/ns.yaml'
        internal.config.kubernetes.io/seqindent: 'compact'
`,
			exp: `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: order-service
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'Kptfile'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: v1
  kind: Namespace
  metadata:
    name: example-ns
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'ns.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: kpt.dev/v1
  kind: Kptfile
  metadata:
    name: subpkg
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'subpkg/Kptfile'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: v1
  kind: Namespace
  metadata:
    name: example-ns
    annotations:
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'subpkg/ns.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: kptfile.kpt.dev
    annotations:
      config.kubernetes.io/local-config: "true"
      internal.config.kubernetes.io/path: 'package-context.yaml'
  data:
    name: order-service
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: kptfile.kpt.dev
    annotations:
      config.kubernetes.io/local-config: "true"
      internal.config.kubernetes.io/path: 'subpkg/package-context.yaml'
  data:
    name: subpkg
results:
- message: generated package context
  severity: info
  file:
    path: package-context.yaml
- message: generated package context
  severity: info
  file:
    path: subpkg/package-context.yaml
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkgCtxGenerator := &PackageContextGenerator{}
			out := &bytes.Buffer{}
			err := pkgCtxGenerator.Run(bytes.NewReader([]byte(test.in)), out)
			if err != test.expErr {
				t.Errorf("exp: %v got: %v", test.expErr, err)
			}
			if out.String() != test.exp {
				t.Errorf("got: %s exp: %s\n", out.String(), test.exp)
			}
		})
	}
}
