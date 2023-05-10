// Copyright 2023 The kpt Authors
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
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
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
			resources: `    junk.yaml: |
      baddata`,
			expectedErr: "junk.yaml: failed to extract objects: unhandled node kind 8",
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
          kpt.dev/config-injection: optional
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
          kpt.dev/config-injection: invalid
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
          kpt.dev/config-injection: required
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
          kpt.dev/config-injection: required
      data:
        foo: bar
      ---
      apiVersion: bigco.com/v2
      kind: MyType
      metadata:
        name: foo
        annotations:
          kpt.dev/config-injection: optional
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
          kpt.dev/config-injection: required
      data:
        foo: bar
      ---
      apiVersion: bigco.com/v2
      kind: MyType
      metadata:
        name: foo
        annotations:
          kpt.dev/config-injection: optional
      spec:
        foo: bar
        bar: foofoo
    some-file.yaml: |
      apiVersion: bigco.com/v2
      kind: MyType
      metadata:
        name: foo2
        annotations:
          kpt.dev/config-injection: required
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

			actualFiles, actualErr := parseFiles(&prr)
			if tc.expectedErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.expectedErr)
			}

			actualInjectionPoints := findInjectionPoints(actualFiles)
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

func TestSetInjectionPointConditionsAndGates(t *testing.T) {
	kptfileWithGates := &kptfilev1.KptFile{
		Info: &kptfilev1.PackageInfo{
			ReadinessGates: []kptfilev1.ReadinessGate{
				{
					ConditionType: "test",
				},
				{
					ConditionType: "test3",
				},
			},
		},
		Status: &kptfilev1.Status{
			Conditions: []kptfilev1.Condition{
				{
					Type:    "test",
					Status:  "False",
					Reason:  "test",
					Message: "test",
				},
				{
					Type:    "test2",
					Status:  "True",
					Reason:  "test2",
					Message: "test2",
				},
				{
					Type:    "test3",
					Status:  "True",
					Reason:  "test3",
					Message: "test3",
				},
			},
		},
	}

	testCases := map[string]struct {
		initialKptfile  *kptfilev1.KptFile
		injectionPoints []*injectionPoint
		expectedKptfile *kptfilev1.KptFile
		expectedErr     string
	}{
		"no injection points": {
			initialKptfile:  &kptfilev1.KptFile{},
			injectionPoints: nil,
			expectedKptfile: &kptfilev1.KptFile{},
		},
		"no injection points, existing gates and conditions": {
			initialKptfile:  kptfileWithGates,
			injectionPoints: nil,
			expectedKptfile: kptfileWithGates,
		},
		"optional, not injected": {
			initialKptfile: &kptfilev1.KptFile{},
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
				},
			},
			expectedKptfile: &kptfilev1.KptFile{
				Status: &kptfilev1.Status{
					Conditions: []kptfilev1.Condition{
						{
							Type:    "config.injection.ConfigMap.foo",
							Status:  "False",
							Reason:  "NoResourceSelected",
							Message: "no resource matched any injection selector for this injection point",
						},
					},
				},
			},
		},
		"required, not injected": {
			initialKptfile: &kptfilev1.KptFile{},
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      true,
					conditionType: "config.injection.ConfigMap.foo",
				},
			},
			expectedKptfile: &kptfilev1.KptFile{
				Info: &kptfilev1.PackageInfo{
					ReadinessGates: []kptfilev1.ReadinessGate{
						{
							ConditionType: "config.injection.ConfigMap.foo",
						},
					},
				},
				Status: &kptfilev1.Status{
					Conditions: []kptfilev1.Condition{
						{
							Type:    "config.injection.ConfigMap.foo",
							Status:  "False",
							Reason:  "NoResourceSelected",
							Message: "no resource matched any injection selector for this injection point",
						},
					},
				},
			},
		},
		"optional, injected": {
			initialKptfile: &kptfilev1.KptFile{},
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
					injected:      true,
					injectedName:  "my-injected-resource",
				},
			},
			expectedKptfile: &kptfilev1.KptFile{
				Status: &kptfilev1.Status{
					Conditions: []kptfilev1.Condition{
						{
							Type:    "config.injection.ConfigMap.foo",
							Status:  "True",
							Reason:  "ConfigInjected",
							Message: "injected resource \"my-injected-resource\" from cluster",
						},
					},
				},
			},
		},
		"multiple optional": {
			initialKptfile: &kptfilev1.KptFile{},
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
					injected:      true,
					injectedName:  "my-injected-resource",
				},
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.SomeResource.foo",
					injected:      true,
					injectedName:  "some-injected-resource",
				},
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.AnotherResource.foo",
					injected:      false,
					injectedName:  "another-injected-resource",
				},
			},
			expectedKptfile: &kptfilev1.KptFile{
				Status: &kptfilev1.Status{
					Conditions: []kptfilev1.Condition{
						{
							Type:    "config.injection.AnotherResource.foo",
							Status:  "False",
							Reason:  "NoResourceSelected",
							Message: "no resource matched any injection selector for this injection point",
						},
						{
							Type:    "config.injection.ConfigMap.foo",
							Status:  "True",
							Reason:  "ConfigInjected",
							Message: "injected resource \"my-injected-resource\" from cluster",
						},
						{
							Type:    "config.injection.SomeResource.foo",
							Status:  "True",
							Reason:  "ConfigInjected",
							Message: "injected resource \"some-injected-resource\" from cluster",
						},
					},
				},
			},
		},
		"mixed existing, optional, required": {
			initialKptfile: kptfileWithGates,
			injectionPoints: []*injectionPoint{
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.ConfigMap.foo",
					injected:      true,
					injectedName:  "my-injected-resource",
				},
				{
					file:          "file.yaml",
					required:      true,
					conditionType: "config.injection.SomeResource.foo",
					injected:      true,
					injectedName:  "some-injected-resource",
				},
				{
					file:          "file.yaml",
					required:      false,
					conditionType: "config.injection.AnotherResource.foo",
					injected:      false,
					injectedName:  "another-injected-resource",
				},
			},
			expectedKptfile: &kptfilev1.KptFile{
				Info: &kptfilev1.PackageInfo{
					ReadinessGates: []kptfilev1.ReadinessGate{
						{
							ConditionType: "config.injection.SomeResource.foo",
						},
						{
							ConditionType: "test",
						},
						{
							ConditionType: "test3",
						},
					},
				},
				Status: &kptfilev1.Status{
					Conditions: []kptfilev1.Condition{
						{
							Type:    "config.injection.AnotherResource.foo",
							Status:  "False",
							Reason:  "NoResourceSelected",
							Message: "no resource matched any injection selector for this injection point",
						},
						{
							Type:    "config.injection.ConfigMap.foo",
							Status:  "True",
							Reason:  "ConfigInjected",
							Message: "injected resource \"my-injected-resource\" from cluster",
						},
						{
							Type:    "config.injection.SomeResource.foo",
							Status:  "True",
							Reason:  "ConfigInjected",
							Message: "injected resource \"some-injected-resource\" from cluster",
						},
						{
							Type:    "test",
							Status:  "False",
							Reason:  "test",
							Message: "test",
						},
						{
							Type:    "test2",
							Status:  "True",
							Reason:  "test2",
							Message: "test2",
						},
						{
							Type:    "test3",
							Status:  "True",
							Reason:  "test3",
							Message: "test3",
						},
					},
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ko, err := fn.NewFromTypedObject(tc.initialKptfile)
			require.NoError(t, err)
			err = setInjectionPointConditionsAndGates(ko, tc.injectionPoints)
			if tc.expectedErr == "" {
				require.NoError(t, err)
				var actualKptfile kptfilev1.KptFile
				err = ko.As(&actualKptfile)
				require.NoError(t, err)
				require.Equal(t, tc.expectedKptfile, &actualKptfile)
			} else {
				require.EqualError(t, err, tc.expectedErr)
			}
		})
	}
}
func TestEnsureConfigInjection(t *testing.T) {

	pvBase := `apiVersion: config.porch.kpt.dev
kind: PackageVariant
metadata:
  name: my-pv
  uid: pv-uid
spec:
  upstream:
    repo: blueprints
    package: foo
    revision: v1
  downstream:
    repo: deployments
    package: bar
`

	prrBase := `apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevisionResources
metadata:
  name: prr
  namespace: default
spec:
  packageName: nephio-system
  repository: nephio-packages
  resources:
    package-context.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: kptfile.kpt.dev
        annotations:
          config.kubernetes.io/local-config: "true"
      data:
        name: example`

	baseKptfile := `
    Kptfile: |
      apiVersion: kpt.dev/v1
      kind: Kptfile
      metadata:
        name: prr
        annotations:
          config.kubernetes.io/local-config: "true"
      info:
        description: Example
`

	testCases := map[string]struct {
		injectors       string
		injectionPoints string
		expectedErr     string
		expectedPRR     string
	}{
		"empty injectors": {
			injectors:       ``,
			injectionPoints: ``,
			expectedErr:     "",
			expectedPRR:     prrBase + baseKptfile,
		},
		"one ConfigMap injection point": {
			injectors: `  injectors:
  - name: us-east1-endpoints
`,
			injectionPoints: `    configmap.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: regional-endpoints
        annotations:
          kpt.dev/config-injection: required
      data:
        db: example
`,
			expectedErr: "",
			expectedPRR: prrBase + `
    Kptfile: |
      apiVersion: kpt.dev/v1
      kind: Kptfile
      metadata:
        name: prr
        annotations:
          config.kubernetes.io/local-config: "true"
      info:
        readinessGates:
        - conditionType: config.injection.ConfigMap.regional-endpoints
        description: Example
      status:
        conditions:
        - type: config.injection.ConfigMap.regional-endpoints
          status: "True"
          message: injected resource "us-east1-endpoints" from cluster
          reason: ConfigInjected
    configmap.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: regional-endpoints
        annotations:
          kpt.dev/config-injection: required
          kpt.dev/injected-resource: us-east1-endpoints
      data:
        db: db.us-east1.example.com
`,
		},
		"one non-ConfigMap injection point": {
			injectors: `  injectors:
  - name: dev-team-beta
    group: hr.example.com
    kind: Team
`,
			injectionPoints: `    team.yaml: |
      apiVersion: hr.example.com/v1alpha1
      kind: Team
      metadata:
        name: team
        annotations:
          kpt.dev/config-injection: required
      spec:
        chargeCode: example
`,
			expectedErr: "",
			expectedPRR: prrBase + `
    Kptfile: |
      apiVersion: kpt.dev/v1
      kind: Kptfile
      metadata:
        name: prr
        annotations:
          config.kubernetes.io/local-config: "true"
      info:
        readinessGates:
        - conditionType: config.injection.Team.team
        description: Example
      status:
        conditions:
        - type: config.injection.Team.team
          status: "True"
          message: injected resource "dev-team-beta" from cluster
          reason: ConfigInjected
    team.yaml: |
      apiVersion: hr.example.com/v1alpha1
      kind: Team
      metadata:
        name: team
        annotations:
          kpt.dev/config-injection: required
          kpt.dev/injected-resource: dev-team-beta
      spec:
        chargeCode: cd
`,
		},
		"mixed injection points": {
			injectors: `  injectors:
  - name: us-east2-endpoints
  - name: dev-team-beta
    group: hr.example.com
    kind: Team
`,
			injectionPoints: `    more.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: my-cm
        annotations:
          kpt.dev/config-injection: optional
      data:
        db: db.example.com
      ---
      apiVersion: hr.example.com/v1alpha1
      kind: Team
      metadata:
        name: team
        annotations:
          kpt.dev/config-injection: required
      spec:
        chargeCode: example
`,
			expectedErr: "",
			expectedPRR: prrBase + `
    Kptfile: |
      apiVersion: kpt.dev/v1
      kind: Kptfile
      metadata:
        name: prr
        annotations:
          config.kubernetes.io/local-config: "true"
      info:
        readinessGates:
        - conditionType: config.injection.Team.team
        description: Example
      status:
        conditions:
        - type: config.injection.ConfigMap.my-cm
          status: "True"
          message: injected resource "us-east2-endpoints" from cluster
          reason: ConfigInjected
        - type: config.injection.Team.team
          status: "True"
          message: injected resource "dev-team-beta" from cluster
          reason: ConfigInjected
    more.yaml: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: my-cm
        annotations:
          kpt.dev/config-injection: optional
          kpt.dev/injected-resource: us-east2-endpoints
      data:
        db: db.us-east2.example.com
      ---
      apiVersion: hr.example.com/v1alpha1
      kind: Team
      metadata:
        name: team
        annotations:
          kpt.dev/config-injection: required
          kpt.dev/injected-resource: dev-team-beta
      spec:
        chargeCode: cd
`,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pv api.PackageVariant
			require.NoError(t, yaml.Unmarshal([]byte(pvBase+tc.injectors), &pv))
			var prr porchapi.PackageRevisionResources
			require.NoError(t, yaml.Unmarshal([]byte(prrBase+baseKptfile+tc.injectionPoints), &prr))

			c := &fakeClient{}
			actualErr := ensureConfigInjection(context.Background(), c, &pv, &prr)
			if tc.expectedErr == "" {
				require.NoError(t, actualErr)
			} else {
				require.EqualError(t, actualErr, tc.expectedErr)
			}

			var expectedPRR porchapi.PackageRevisionResources
			require.NoError(t, yaml.Unmarshal([]byte(tc.expectedPRR), &expectedPRR))

			require.Equal(t, expectedPRR, prr)
		})
	}
}
