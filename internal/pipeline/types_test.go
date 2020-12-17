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
package pipeline_test

import (
	"reflect"
	"testing"

	. "github.com/GoogleContainerTools/kpt/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestString(t *testing.T) {
	expected := `apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
    name: pipeline
sources:
  - ./*
`
	actual, err := NewPipeline().String()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.EqualValues(t, expected, actual) {
		t.Fatalf("unexpected string value")
	}
}

var expected Pipeline = Pipeline{
	Sources: []string{
		"a",
		"b",
	},
	Generators: []Function{
		{
			Image: "gcr.io/kpt-functions/generate-folders",
			Config: *yaml.MustParse(`apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
name: root-hierarchy
namespace: hierarchy # {"$kpt-set":"namespace"}`).YNode(),
		},
	},
	Transformers: []Function{
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
	Validators: []Function{
		{
			Image: "gcr.io/kpt-functions/policy-controller-validate",
		},
	},
}

func TestAdd(t *testing.T) {
	actual := (&Pipeline{}).
		AddSources("a", "b").
		AddGenerators(expected.Generators...).
		AddTransformers(expected.Transformers...).
		AddValidators(expected.Validators...)

	if !isPipelineEqual(t, *actual, expected) {
		t.Fatalf("build result is different from expected")
	}
}

func TestSet(t *testing.T) {
	actual := NewPipeline().
		SetName("").
		SetKind("").
		SetAPIVersion("").
		SetSources([]string{"a", "b"}).
		SetGenerators(expected.Generators).
		SetTransformers(expected.Transformers).
		SetValidators(expected.Validators)
	if !isPipelineEqual(t, *actual, expected) {
		t.Fatalf("build result is different from expected")
	}
}

type testcase struct {
	Input    string
	Expected Pipeline
	Error    bool
}

var testcases map[string]testcase = map[string]testcase{
	"simple": {
		Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
`,
		Expected: *NewPipeline().
			SetAPIVersion("kpt.dev/v1alpha1").
			SetKind("Pipeline").
			SetName("pipeline").
			SetSources(nil),
	},
	"with sources": {
		Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
sources:
- ./base
- ./*
`,
		Expected: *NewPipeline().
			SetAPIVersion("kpt.dev/v1alpha1").
			SetKind("Pipeline").
			SetName("pipeline").
			SetSources([]string{"./base", "./*"}),
	},
	"complex": {
		Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
sources:
- ./base
- ./*

generators:
- image: gcr.io/kpt-functions/generate-folders
  config:
    apiVersion: cft.dev/v1alpha1
    kind: ResourceHierarchy
    metadata:
      name: root-hierarchy
      namespace: hierarchy # {"$kpt-set":"namespace"}
transformers:
- image: patch-strategic-merge
  configPath: ./patch.yaml
- image: gcr.io/kpt-functions/set-annotation
  configMap:
    environment: dev

validators:
- image: gcr.io/kpt-functions/policy-controller-validate
`,
		Expected: *NewPipeline().
			SetAPIVersion("kpt.dev/v1alpha1").
			SetKind("Pipeline").
			SetName("pipeline").
			SetSources([]string{"./base", "./*"}).
			AddGenerators(
				Function{
					Image: "gcr.io/kpt-functions/generate-folders",
					Config: *yaml.MustParse(`apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy # {"$kpt-set":"namespace"}`).YNode(),
				},
			).
			AddTransformers(
				Function{
					Image:      "patch-strategic-merge",
					ConfigPath: "./patch.yaml",
				},
				Function{
					Image: "gcr.io/kpt-functions/set-annotation",
					ConfigMap: map[string]string{
						"environment": "dev",
					},
				},
			).
			AddValidators(
				Function{
					Image: "gcr.io/kpt-functions/policy-controller-validate",
				},
			),
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

func isFunctionEqual(t *testing.T, f1, f2 Function) bool {
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

func isFunctionSliceEqual(t *testing.T, fs1, fs2 []Function) bool {
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

func isPipelineEqual(t *testing.T, p1, p2 Pipeline) bool {
	if !isFunctionSliceEqual(t, p1.Transformers, p2.Transformers) {
		return false
	}

	if !isFunctionSliceEqual(t, p1.Generators, p2.Generators) {
		return false
	}

	if !isFunctionSliceEqual(t, p1.Validators, p2.Validators) {
		return false
	}

	if !assert.EqualValues(t, p1.Sources, p2.Sources) {
		return false
	}

	if !assert.EqualValues(t, p1.Name, p2.Name) {
		return false
	}

	if !assert.EqualValues(t, p1.Kind, p2.Kind) {
		return false
	}

	if !assert.EqualValues(t, p1.APIVersion, p2.APIVersion) {
		return false
	}

	return true
}
