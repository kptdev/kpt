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
	"sigs.k8s.io/kustomize/kyaml/copyutil"
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

func TestSetV2AutoSetter(t *testing.T) {
	var tests = []struct {
		name            string
		setterName      string
		setterValue     string
		dataset         string
		expectedDataset string
	}{
		{
			name:            "autoset-recurse-subpackages",
			dataset:         "dataset-with-autosetters",
			setterName:      "gcloud.core.project",
			setterValue:     "my-project",
			expectedDataset: "dataset-with-autosetters-set",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			// reset the openAPI afterward
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()
			testDataDir := filepath.Join("../../", "testutil", "testdata")
			sourceDir := filepath.Join(testDataDir, test.dataset)
			expectedDir := filepath.Join(testDataDir, test.expectedDataset)
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = copyutil.CopyDir(sourceDir, baseDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)
			GetProjectNumberFromProjectID = func(projectID string) (string, error) {
				return "1234", nil
			}
			err = SetV2AutoSetter(test.setterName, test.setterValue, baseDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			diff, err := copyutil.Diff(baseDir, expectedDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t, 0, diff.Len()) {
				t.FailNow()
			}
		})
	}
}

func TestEnvironmentSetters(t *testing.T) {
	var tests = []struct {
		name            string
		envVariables    []string
		dataset         string
		expectedDataset string
	}{
		{
			name:    "autoset-recurse-subpackages",
			dataset: "dataset-with-autosetters",
			envVariables: []string{"KPT_SET_gcloud.core.project=my-project",
				"KPT_SET_gcloud.project.projectNumber=1234"},
			expectedDataset: "dataset-with-autosetters-set",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			// reset the openAPI afterward
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()
			testDataDir := filepath.Join("../../", "testutil", "testdata")
			sourceDir := filepath.Join(testDataDir, test.dataset)
			expectedDir := filepath.Join(testDataDir, test.expectedDataset)
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = copyutil.CopyDir(sourceDir, baseDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)
			environmentVariables = func() []string {
				return test.envVariables
			}
			err = SetEnvAutoSetters(baseDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			diff, err := copyutil.Diff(baseDir, expectedDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t, 0, diff.Len()) {
				t.FailNow()
			}
		})
	}
}
