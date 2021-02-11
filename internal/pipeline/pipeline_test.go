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
package pipeline

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func checkOutput(t *testing.T, tc testcase, actual *v1alpha2.Pipeline, err error) {
	if tc.Error {
		if !assert.Error(t, err, "error is expected.") {
			t.FailNow()
		}
		return
	} else if !assert.NoError(t, err, "error is not expected.") {
		t.FailNow()
	}
	if !assert.True(t, isPipelineEqual(t, *tc.Expected, *actual),
		"pipelines don't equal.") {
		t.FailNow()
	}
}

func TestPipelineValidate(t *testing.T) {
	type input struct {
		Name  string
		Input string
		Valid bool
	}
	cases := []input{
		{
			Name: "have functions",
			Input: `
mutators:
- image: gcr.io/kpt-functions/generate-folders
  config:
    apiVersion: cft.dev/v1alpha1
    kind: ResourceHierarchy
    metadata:
      name: root-hierarchy
      namespace: hierarchy # {"$kpt-set":"namespace"}
- image: patch-strategic-merge
  configPath: ./patch.yaml
- image: gcr.io/kpt-functions/set-annotation
  configMap:
    environment: dev

validators:
- image: gcr.io/kpt-functions/policy-controller-validate
`,
			Valid: true,
		},
		{
			Name: "invalid function name",
			Input: `
mutators:
- image: patch@_@strategic-merge
  configPath: ./patch.yaml
`,
			Valid: false,
		},
		{
			Name: "more than 1 config",
			Input: `
mutators:
- image: patch-strategic-merge
  configPath: ./patch.yaml
  configMap:
    environment: dev
`,
			Valid: false,
		},
		{
			Name: "absolute config path",
			Input: `
mutators:
- image: patch-strategic-merge
  configPath: /patch.yaml
`,
			Valid: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			// FromReader will validate the pipeline
			p, err := fromString(c.Input)
			assert.NoError(t, err)
			err = ValidatePipeline(p)
			if c.Valid && err != nil {
				t.Fatalf("pipeline should be valid, %s", err)
			}
			if !c.Valid && err == nil {
				t.Fatal("pipeline should not be valid")
			}
		})

	}
}

type testcase struct {
	Input    string
	Expected *v1alpha2.Pipeline
	Error    bool
}

var testcases map[string]testcase = map[string]testcase{
	"complex": {
		Input: `
mutators:
- image: gcr.io/kpt-functions/generate-folders
  config:
    apiVersion: cft.dev/v1alpha1
    kind: ResourceHierarchy
    metadata:
      name: root-hierarchy
      namespace: hierarchy # {"$kpt-set":"namespace"}
- image: patch-strategic-merge
  configPath: ./patch.yaml
- image: gcr.io/kpt-functions/set-annotation
  configMap:
    environment: dev

validators:
- image: gcr.io/kpt-functions/policy-controller-validate
`,
		Expected: &v1alpha2.Pipeline{
			Mutators: []v1alpha2.Function{
				{
					Image: "gcr.io/kpt-functions/generate-folders",
					Config: *yaml.MustParse(`apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy # {"$kpt-set":"namespace"}`).YNode(),
				},
				{
					Image:      "patch-strategic-merge",
					ConfigPath: "./patch.yaml",
				},
				{
					Image: "gcr.io/kpt-functions/set-annotation",
					ConfigMap: map[string]string{
						"environment": "dev",
					},
				},
			},
			Validators: []v1alpha2.Function{
				{
					Image: "gcr.io/kpt-functions/policy-controller-validate",
				},
			},
		},
	},
	"error": {
		Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
unknown
`,
		Error: true,
	},
}

func TestFromString(t *testing.T) {
	for name, tc := range testcases {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			actual, err := fromString(tc.Input)
			checkOutput(t, tc, actual, err)
		})
	}
}

func isFunctionEqual(t *testing.T, f1, f2 v1alpha2.Function) bool {
	if reflect.DeepEqual(f1.Config, f2.Config) {
		return reflect.DeepEqual(f1, f2)
	}
	// Config objects cannot be compared directly
	f1ConfigString, err := yaml.String(&f1.Config)
	assert.NoError(t, err)
	f2ConfigString, err := yaml.String(&f2.Config)
	assert.NoError(t, err)

	// Compare the functions
	result := assert.EqualValues(t, f1ConfigString, f2ConfigString) &&
		assert.EqualValues(t, f1.ConfigMap, f2.ConfigMap) &&
		assert.EqualValues(t, f1.ConfigPath, f2.ConfigPath) &&
		assert.EqualValues(t, f1.Image, f2.Image)
	return result
}

func isFunctionSliceEqual(t *testing.T, fs1, fs2 []v1alpha2.Function) bool {
	if len(fs1) != len(fs2) {
		return false
	}
	for i := range fs1 {
		if !isFunctionEqual(t, fs1[i], fs2[i]) {
			return false
		}
	}
	return true
}

func isPipelineEqual(t *testing.T, p1, p2 v1alpha2.Pipeline) bool {
	if !isFunctionSliceEqual(t, p1.Mutators, p2.Mutators) {
		return false
	}

	if !isFunctionSliceEqual(t, p1.Validators, p2.Validators) {
		return false
	}

	return true
}
