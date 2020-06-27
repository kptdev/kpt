// Copyright 2019 Google LLC
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

package setters

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

func TestDefExists(t *testing.T) {
	var tests = []struct {
		name           string
		inputKptfile   string
		setterName     string
		expectedResult bool
	}{
		{
			name: "def exists",
			inputKptfile: `
apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.gcloud.project.projectNumber:
      description: hello world
      x-k8s-cli:
        setter:
          name: gcloud.project.projectNumber
          value: 123
          setBy: me
 `,
			setterName:     "gcloud.project.projectNumber",
			expectedResult: true,
		},
		{
			name: "def doesn't exist",
			inputKptfile: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
 `,
			setterName:     "gcloud.project.projectNumber",
			expectedResult: false,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()
			dir, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			defer os.RemoveAll(dir)
			err = ioutil.WriteFile(filepath.Join(dir, "Kptfile"), []byte(test.inputKptfile), 0600)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedResult, DefExists(dir, test.setterName))
		})
	}
}

func TestCheckRequiredSettersSet(t *testing.T) {
	var tests = []struct {
		name          string
		inputKptfile  string
		expectedError bool
	}{
		{
			name: "no required, no isSet",
			inputKptfile: `
apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.gcloud.project.projectNumber:
      description: hello world
      x-k8s-cli:
        setter:
          name: gcloud.project.projectNumber
          value: 123
          setBy: me
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
 `,
			expectedError: false,
		},
		{
			name: "required true, no isSet",
			inputKptfile: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: true
 `,
			expectedError: true,
		},
		{
			name: "required true, isSet true",
			inputKptfile: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: true
          isSet: true
 `,
			expectedError: false,
		},

		{
			name: "required false, isSet true",
			inputKptfile: `
apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.gcloud.project.projectNumber:
      description: hello world
      x-k8s-cli:
        setter:
          name: gcloud.project.projectNumber
          value: 123
          setBy: me
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: false
          isSet: true
 `,
			expectedError: false,
		},

		{
			name: "required true, isSet false",
			inputKptfile: `
apiVersion: v1alpha1
kind: Kptfile
openAPI:
  definitions:
    io.k8s.cli.setters.gcloud.project.projectNumber:
      description: hello world
      x-k8s-cli:
        setter:
          name: gcloud.project.projectNumber
          value: 123
          setBy: me
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: true
          isSet: false
 `,
			expectedError: true,
		},

		{
			name:          "no Kptfile",
			inputKptfile:  ``,
			expectedError: false,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()
			dir, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			defer os.RemoveAll(dir)
			if test.inputKptfile != "" {
				err = ioutil.WriteFile(filepath.Join(dir, "Kptfile"), []byte(test.inputKptfile), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}
			err = CheckRequiredSettersSet(dir)
			if test.expectedError && !assert.Error(t, err) {
				t.FailNow()
			}
			if !test.expectedError && !assert.NoError(t, err) {
				t.FailNow()
			}
		})
	}
}
