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
	"bytes"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	. "github.com/GoogleContainerTools/kpt/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestString(t *testing.T) {
	expected := "{ResourceMeta:{TypeMeta:{APIVersion:kpt.dev/v1alpha1 Kind:Pipeline} " +
		"ObjectMeta:{NameMeta:{Name:pipeline Namespace:} Labels:map[] Annotations:map[]}} " +
		"Sources:[./*] Generators:[] Transformers:[] Validators:[]}"
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
		Sources: []string{
			"./*",
		},
	}
	if !assert.True(t, isPipelineEqual(t, *p, expected)) {
		t.FailNow()
	}
}

func checkOutput(t *testing.T, name string, tc testcase, actual *Pipeline, err error) {
	if tc.Error {
		if !assert.Errorf(t, err, "error is expected. Test case: %s", name) {
			t.FailNow()
		}
		return
	} else if !assert.NoError(t, err, "error is not expected. Test case: %s", name) {
		t.FailNow()
	}
	if !assert.Truef(t, isPipelineEqual(t, *actual, tc.Expected),
		"pipelines don't equal. Test case: %s", name) {
		t.FailNow()
	}
}

func TestFromReader(t *testing.T) {
	for name, tc := range testcases {
		r := bytes.NewBufferString(tc.Input)
		actual, err := FromReader(r)
		checkOutput(t, name, tc, actual, err)
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
		path, err := preparePipelineFile(tc.Input)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		actual, err := FromFile(path)
		checkOutput(t, name, tc, actual, err)
		os.Remove(path)
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
			Sources: []string{"./*"},
		},
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
			Sources: []string{
				"./base",
				"./*",
			},
		},
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
			Sources: []string{
				"./base",
				"./*",
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

func TestValidateFunctionName(t *testing.T) {
	type input struct {
		Name  string
		Valid bool
	}
	inputs := []input{
		{
			Name:  "gcr.io/kpt-functions/generate-folders",
			Valid: true,
		},
		{
			Name:  "patch-strategic-merge",
			Valid: true,
		},
		{
			Name:  "a.b.c:1234/foo/bar/generate-folders",
			Valid: true,
		},
		{
			Name:  "ab-.b/c",
			Valid: false,
		},
		{
			Name:  "a/a/",
			Valid: false,
		},
		{
			Name:  "a//a/a",
			Valid: false,
		},
		{
			Name:  "example.com/.dots/myimage",
			Valid: false,
		},
		{
			Name:  "registry.io/foo/project--id.module--name.ver---sion--name",
			Valid: true,
		},
		{
			Name:  "Foo/FarB",
			Valid: false,
		},
	}

	for _, n := range inputs {
		err := ValidateFunctionName(n.Name)
		if n.Valid && err != nil {
			t.Fatalf("function name %s should be valid", n.Name)
		}
		if !n.Valid && err == nil {
			t.Fatalf("function name %s should not be valid", n.Name)
		}
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
			Name: "no sources, no functions",
			Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
`,
			Valid: true,
		},
		{
			Name: "have sources, no functions",
			Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
sources:
- ./base
`,
			Valid: true,
		},
		{
			Name: "have sources and functions",
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
			Valid: true,
		},
		{
			Name: "invalid apiversion",
			Input: `
apiVersion: kpt.dev/v1
kind: Pipeline
metadata:
  name: pipeline
sources:
- ./base
`,
			Valid: false,
		},
		{
			Name: "absolute source path",
			Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
sources:
- /foo/bar
`,
			Valid: false,
		},
		{
			Name: "invalid function name",
			Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
sources:
- ./*
transformers:
- image: patch@_@strategic-merge
  configPath: ./patch.yaml
`,
			Valid: false,
		},
		{
			Name: "more than 1 config",
			Input: `
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
sources:
- ./*
transformers:
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
apiVersion: kpt.dev/v1alpha1
kind: Pipeline
metadata:
  name: pipeline
sources:
- ./*
transformers:
- image: patch-strategic-merge
  configPath: /patch.yaml
`,
			Valid: false,
		},
	}

	for _, c := range cases {
		b := bytes.NewBufferString(c.Input)
		// FromReader will validate the pipeline
		_, err := FromReader(b)
		if c.Valid && err != nil {
			t.Fatalf("%s: pipeline should be valid, %s", c.Name, err)
		}
		if !c.Valid && err == nil {
			t.Fatalf("%s: pipeline should not be valid", c.Name)
		}
	}
}

func TestValidatePath(t *testing.T) {
	type input struct {
		Path  string
		Valid bool
	}

	cases := []input{
		{
			Path:  "a/b/c",
			Valid: true,
		},
		{
			Path:  "/a/b",
			Valid: false,
		},
		{
			Path:  ".",
			Valid: true,
		},
		{
			Path:  "a\\b",
			Valid: false,
		},
		{
			Path:  "a:\\b\\c",
			Valid: false,
		},
		{
			Path:  "../a/../b",
			Valid: false,
		},
		{
			Path:  "a//b",
			Valid: false,
		},
		{
			Path:  "a/b/.",
			Valid: false,
		},
	}

	for _, c := range cases {
		ret := ValidatePath(c.Path)
		if (ret == nil) != c.Valid {
			t.Fatalf("returned value for path %s should be %t, got %t",
				c.Path, c.Valid, (ret == nil))
		}
	}
}
