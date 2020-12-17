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
	"testing"

	. "github.com/GoogleContainerTools/kpt/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNewPipeline(t *testing.T) {
	p := NewPipeline()
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

func TestFromBytes(t *testing.T) {
	for name, tc := range testcases {
		actual, err := FromBytes([]byte(tc.Input))
		checkOutput(t, name, tc, actual, err)
	}
}

func TestFromString(t *testing.T) {
	for name, tc := range testcases {
		actual, err := FromString(tc.Input)
		checkOutput(t, name, tc, actual, err)
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
