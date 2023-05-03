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

package packagevariant

import (
	//"context"
	"fmt"
	"sort"
	"testing"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	//api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestFindInjectionPoints(t *testing.T) {

	prrBase := `apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevisionResources
metadata:
  name: prr
  namespace: default
spec:
  packageName: nephio-system
  repository: nephio-packages
  resources:
    Kptfile: |
      apiVersion: kpt.dev/v1
      kind: Kptfile
      metadata:
        name: prr
        annotations:
          config.kubernetes.io/local-config: "true"
      info:
        description: Example
    package-context.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: kptfile.kpt.dev
        annotations:
          config.kubernetes.io/local-config: "true"
      data:
        name: example
`

	testCases := map[string]struct {
		resources   string
		expectedErr string
		expected    []*injectionPoint
	}{
		"parse error": {
			resources: `    junk: |
      baddata`,
			expectedErr: "junk: failed to extract objects: unhandled node kind 8",
		},
		"no injection points": {
			resources: ``,
			expected:  nil,
		},
		"one optional injection point": {
			resources: `    file.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: foo
        annotations:
          config.kubernetes.io/local-config: "true"
          kpt.dev/config-injection: "optional"
      data:
        foo: bar
`,
			expected: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
				},
			},
		},
		"one invalid injection point": {
			resources: `    file.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: foo
        annotations:
          config.kubernetes.io/local-config: "true"
          kpt.dev/config-injection: "invalid"
      data:
        foo: bar
`,
			expected: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
					errors:        []string{"file.yaml: ConfigMap/foo has invalid \"kpt.dev/config-injection\" annotation value of \"invalid\""},
				},
			},
		},
		"one required injection point": {
			resources: `    file2.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: foo
        annotations:
          kpt.dev/config-injection: "required"
      data:
        foo: bar
`,
			expected: []*injectionPoint{
				{
					file:          "file2.yaml",
					required:      true,
					conditionType: "config.injection.ConfigMap.foo",
				},
			},
		},
		"multiple injection points in one file": {
			resources: `    file.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: foo
        annotations:
          kpt.dev/config-injection: "required"
      data:
        foo: bar
      ---
      apiVersion: bigco.com/v2
      kind: MyType
      metadata:
        name: foo
        annotations:
          kpt.dev/config-injection: "optional"
      spec:
        foo: bar
        bar: foofoo
`,
			expected: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      true,
					conditionType: "config.injection.ConfigMap.foo",
				},
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.MyType.foo",
				},
			},
		},
		"multiple injection points across files": {
			resources: `    file.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: foo
        annotations:
          kpt.dev/config-injection: "required"
      data:
        foo: bar
      ---
      apiVersion: bigco.com/v2
      kind: MyType
      metadata:
        name: foo
        annotations:
          kpt.dev/config-injection: "optional"
      spec:
        foo: bar
        bar: foofoo
    some-file.yaml: |
      apiVersion: bigco.com/v2
      kind: MyType
      metadata:
        name: foo2
        annotations:
          kpt.dev/config-injection: "required"
      spec:
        foo: bar
        bar: foofoo
`,
			expected: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      true,
					conditionType: "config.injection.ConfigMap.foo",
				},
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.MyType.foo",
				},
				{
					file:          "some-file.yaml",
					required:      true,
					conditionType: "config.injection.MyType.foo2",
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var prr porchapi.PackageRevisionResources
			require.NoError(t, yaml.Unmarshal([]byte(prrBase+tc.resources), &prr))

			actualInjectionPoints, actualErr := findInjectionPoints(&prr)
			if tc.expectedErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.expectedErr)
			}

			require.Equal(t, len(tc.expected), len(actualInjectionPoints))

			// ensure a stable ordering
			sort.Slice(actualInjectionPoints,
				func(i, j int) bool {
					return actualInjectionPoints[i].conditionType < actualInjectionPoints[j].conditionType
				})
			for i, ip := range actualInjectionPoints {
				require.Equal(t, tc.expected[i].conditionType, ip.conditionType)
				require.Equal(t, tc.expected[i].file, ip.file)
				require.Equal(t, tc.expected[i].required, ip.required)
				require.Equal(t, tc.expected[i].errors, ip.errors)
			}
		})
	}
}

func TestValidateInjectionPoints(t *testing.T) {
	testCases := map[string]struct {
		injectionPoints []*injectionPoint
		expected        error
	}{
		"no injection points": {
			injectionPoints: nil,
			expected:        nil,
		},
		"one optional injection point": {
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
				},
			},
			expected: nil,
		},
		"one invalid injection point": {
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
					errors:        []string{"e1"},
				},
			},
			expected: fmt.Errorf("errors in injection points: e1"),
		},
		"multiple distinct, valid injection points": {
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      true,
					conditionType: "config.injection.ConfigMap.foo",
				},
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.MyType.foo",
				},
			},
			expected: nil,
		},
		"multiple distinct, invalid injection points": {
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
					errors:        []string{"e1", "e2"},
				},
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo2",
					errors:        []string{"e3"},
				},
			},
			expected: fmt.Errorf("errors in injection points: e1, e2, e3"),
		},
		"multiple ambiguous injection points": {
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
				},
				{
					file:          "file2.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
				},
				{
					file:          "file3.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
				},
			},
			expected: fmt.Errorf("errors in injection points: duplicate injection conditionType \"config.injection.ConfigMap.foo\" (file.yaml and file2.yaml), duplicate injection conditionType \"config.injection.ConfigMap.foo\" (file2.yaml and file3.yaml)"),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			allErrs := validateInjectionPoints(tc.injectionPoints)
			require.Equal(t, tc.expected, allErrs)
		})
	}
}
