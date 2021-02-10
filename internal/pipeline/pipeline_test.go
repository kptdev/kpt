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
	"bytes"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestString(t *testing.T) {
	expected := "{ResourceMeta:{TypeMeta:{APIVersion:kpt.dev/v1alpha1 Kind:Pipeline} " +
		"ObjectMeta:{NameMeta:{Name:pipeline Namespace:} Labels:map[] Annotations:map[]}} " +
		"Mutators:[] Validators:[]}"
	actual := New().String()
	if !assert.EqualValues(t, expected, actual) {
		t.Fatalf("unexpected string value")
	}
}

func TestNew(t *testing.T) {
	p := New()
	expected := Pipeline{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: "kpt.dev/v1alpha1",
				Kind:       "Pipeline",
			},
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: "pipeline",
				},
			},
		},
	}
	if !assert.True(t, isPipelineEqual(t, *p, expected)) {
		t.FailNow()
	}
}

func checkOutput(t *testing.T, tc testcase, actual *Pipeline, err error) {
	if tc.Error {
		if !assert.Error(t, err, "error is expected.") {
			t.FailNow()
		}
		return
	} else if !assert.NoError(t, err, "error is not expected.") {
		t.FailNow()
	}
	if !assert.True(t, isPipelineEqual(t, tc.Expected, *actual),
		"pipelines don't equal.") {
		t.FailNow()
	}
}

func TestFromReader(t *testing.T) {
	for name, tc := range testcases {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			r := bytes.NewBufferString(tc.Input)
			actual, err := FromReader(r)
			checkOutput(t, tc, actual, err)
		})
	}
}

func preparePipelineFile(s string) (string, error) {
	tmp, err := ioutil.TempFile("", "kpt-pipeline-*")
	if err != nil {
		return "", err
	}
	_, err = tmp.WriteString(s)
	if err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func TestFromFile(t *testing.T) {
	for name, tc := range testcases {
		tc := tc
		name := name
		t.Run(name, func(t *testing.T) {
			path, err := preparePipelineFile(tc.Input)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			actual, err := FromFile(path)
			checkOutput(t, tc, actual, err)
			os.Remove(path)
		})

	}
}

func TestFromFileError(t *testing.T) {
	_, err := FromFile("not-exist")
	if err == nil {
		t.Fatalf("expect an error when open non-exist file")
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
		Expected: Pipeline{
			ResourceMeta: yaml.ResourceMeta{
				TypeMeta: yaml.TypeMeta{
					APIVersion: "kpt.dev/v1alpha1",
					Kind:       "Pipeline",
				},
				ObjectMeta: yaml.ObjectMeta{
					NameMeta: yaml.NameMeta{
						Name: "pipeline",
					},
				},
			},
		},
	},
	"with sources": {
		Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
`,
		Expected: Pipeline{
			ResourceMeta: yaml.ResourceMeta{
				TypeMeta: yaml.TypeMeta{
					APIVersion: "kpt.dev/v1alpha1",
					Kind:       "Pipeline",
				},
				ObjectMeta: yaml.ObjectMeta{
					NameMeta: yaml.NameMeta{
						Name: "pipeline",
					},
				},
			},
		},
	},
	"complex": {
		Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline

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
		Expected: Pipeline{
			ResourceMeta: yaml.ResourceMeta{
				TypeMeta: yaml.TypeMeta{
					APIVersion: "kpt.dev/v1alpha1",
					Kind:       "Pipeline",
				},
				ObjectMeta: yaml.ObjectMeta{
					NameMeta: yaml.NameMeta{
						Name: "pipeline",
					},
				},
			},
			Mutators: []Function{
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
			Validators: []Function{
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
	if !isFunctionSliceEqual(t, p1.Mutators, p2.Mutators) {
		return false
	}

	if !isFunctionSliceEqual(t, p1.Validators, p2.Validators) {
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
