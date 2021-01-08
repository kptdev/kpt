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
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
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
		expectedOut     string
	}{
		{
			name:            "autoset-recurse-subpackages",
			dataset:         "dataset-with-autosetters",
			setterName:      "gcloud.core.project",
			setterValue:     "my-project",
			expectedDataset: "dataset-with-autosetters-set",
			expectedOut: `automatically set 1 field(s) for setter "gcloud.core.project" to value "my-project" in package "${baseDir}/mysql" derived from gcloud config
automatically set 1 field(s) for setter "gcloud.project.projectNumber" to value "1234" in package "${baseDir}/mysql" derived from gcloud config
automatically set 1 field(s) for setter "gcloud.core.project" to value "my-project" in package "${baseDir}/mysql/storage" derived from gcloud config
automatically set 1 field(s) for setter "gcloud.project.projectNumber" to value "1234" in package "${baseDir}/mysql/storage" derived from gcloud config
`,
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
			out := &bytes.Buffer{}
			err = SetV2AutoSetter(test.setterName, test.setterValue, baseDir, out)
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

			// normalize path format for windows
			actualNormalized := strings.ReplaceAll(
				strings.ReplaceAll(out.String(), "\\", "/"),
				"//", "/")
			expectedOut := strings.ReplaceAll(test.expectedOut, "${baseDir}", baseDir)
			expectedNormalized := strings.ReplaceAll(expectedOut, "\\", "/")
			if !assert.Equal(t, expectedNormalized, actualNormalized) {
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
		expectedOut     string
	}{
		{
			name:    "autoset-recurse-subpackages",
			dataset: "dataset-with-autosetters",
			envVariables: []string{"KPT_SET_gcloud.core.project=my-project",
				"KPT_SET_gcloud.project.projectNumber=1234"},
			expectedDataset: "dataset-with-autosetters-set",
			expectedOut: `automatically set 1 field(s) for setter "gcloud.core.project" to value "my-project" in package "${baseDir}/mysql" derived from environment
automatically set 1 field(s) for setter "gcloud.core.project" to value "my-project" in package "${baseDir}/mysql/storage" derived from environment
automatically set 1 field(s) for setter "gcloud.project.projectNumber" to value "1234" in package "${baseDir}/mysql" derived from environment
automatically set 1 field(s) for setter "gcloud.project.projectNumber" to value "1234" in package "${baseDir}/mysql/storage" derived from environment
`,
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
			out := &bytes.Buffer{}
			a := AutoSet{
				Writer:      out,
				PackagePath: baseDir,
			}
			err = a.SetEnvAutoSetters()
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
			// normalize path format for windows
			actualNormalized := strings.ReplaceAll(
				strings.ReplaceAll(out.String(), "\\", "/"),
				"//", "/")
			expectedOut := strings.ReplaceAll(test.expectedOut, "${baseDir}", baseDir)
			expectedNormalized := strings.ReplaceAll(expectedOut, "\\", "/")
			if !assert.Equal(t, expectedNormalized, actualNormalized) {
				t.FailNow()
			}
		})
	}
}

func TestSetInheritedSetters(t *testing.T) {
	var tests = []struct {
		name                     string
		parentPath               string
		childPath                string
		parentKptfile            string
		childKptfile             string
		childConfigFile          string
		nestedKptfile            string
		nestedConfigFile         string
		expectedChildKptfile     string
		expectedChildConfigFile  string
		expectedNestedKptfile    string
		expectedNestedConfigFile string
		expectedOut              string
	}{
		{
			name:       "autoset-inherited-setters",
			parentPath: "${baseDir}/parentPath",
			childPath:  "${baseDir}/parentPath/somedir/childPath",
			parentKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: parent
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: parent_namespace
          isSet: true
    io.k8s.cli.setters.name:
      x-k8s-cli:
        setter:
          name: name
          value: parent_name`,
			nestedConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}
  name: child_name # {"$kpt-set":"name"}`,
			nestedKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
    io.k8s.cli.setters.name:
      x-k8s-cli:
        setter:
          name: name
          value: child_name
`,
			childConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}
  name: child_name # {"$kpt-set":"name"}`,
			childKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
    io.k8s.cli.setters.name:
      x-k8s-cli:
        setter:
          name: name
          value: child_name
`,
			expectedChildKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: parent_namespace
          isSet: true
    io.k8s.cli.setters.name:
      x-k8s-cli:
        setter:
          name: name
          value: parent_name
`,
			expectedChildConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: parent_namespace # {"$kpt-set":"namespace"}
  name: parent_name # {"$kpt-set":"name"}
`,
			expectedOut: `automatically set 1 field(s) for setter "namespace" to value "parent_namespace" in package "${childPkg}" derived from parent "${parentPkgKptfile}"
automatically set 1 field(s) for setter "name" to value "parent_name" in package "${childPkg}" derived from parent "${parentPkgKptfile}"
automatically set 1 field(s) for setter "namespace" to value "parent_namespace" in package "${nestedPkg}" derived from parent "${childPkg}/Kptfile"
automatically set 1 field(s) for setter "name" to value "parent_name" in package "${nestedPkg}" derived from parent "${childPkg}/Kptfile"
`,
		},
		{
			name:       "child-has-no-parent",
			parentPath: "${baseDir}/parentPath",
			childPath:  "${baseDir}/childPath",
			parentKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: parent
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: parent_namespace`,
			childConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}
`,
			childKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
`,
			expectedChildKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
`,
			expectedChildConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}
`,
		},
		{
			name:       "autoset-inherited-setters-error",
			parentPath: "${baseDir}/parentPath",
			childPath:  "${baseDir}/parentPath/somedir/childPath",
			parentKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: parent
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: parent_namespace`,
			childConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}
`,
			childKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      maxLength: 15
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
`,
			expectedChildKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      maxLength: 15
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
`,
			expectedChildConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}
`,
			expectedOut: `failed to set "namespace" automatically in package "${childPkg}" with error: ` +
				`The input value doesn't validate against provided OpenAPI schema: validation failure list:
namespace in body should be at most 15 chars long

`,
		},
		{
			name:       "skip-autoset-inherited-setters-already-set",
			parentPath: "${baseDir}/parentPath",
			childPath:  "${baseDir}/parentPath/somedir/childPath",
			parentKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: parent
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: parent_namespace`,
			childConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}`,
			childKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
          isSet: true
`,
			expectedChildKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
          isSet: true
`,
			expectedChildConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}`,
		},
		{
			name:       "inherit-defalut-values-from-parent",
			parentPath: "${baseDir}/parentPath",
			childPath:  "${baseDir}/parentPath/somedir/childPath",
			parentKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: parent
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: parent_namespace`,
			childConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: child_namespace # {"$kpt-set":"namespace"}`,
			childKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: child_namespace
`,
			expectedChildKptfile: `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: child
openAPI:
  definitions:
    io.k8s.cli.setters.namespace:
      x-k8s-cli:
        setter:
          name: namespace
          value: parent_namespace
`,
			expectedChildConfigFile: `apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: parent_namespace # {"$kpt-set":"namespace"}
`,
			expectedOut: `automatically set 1 field(s) for setter "namespace" to value "parent_namespace" in package "${childPkg}" derived from parent "${parentPkgKptfile}"
`,
		},
	}
	for i := range tests {
		test := tests[i]
		fieldmeta.SetShortHandRef("$kpt-set")
		t.Run(test.name, func(t *testing.T) {
			// reset the openAPI afterward
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(baseDir)
			parentPkg := strings.ReplaceAll(test.parentPath, "${baseDir}", baseDir)
			childPkg := strings.ReplaceAll(test.childPath, "${baseDir}", baseDir)
			nestedPkg := strings.ReplaceAll(filepath.Join(test.childPath, "nested_pkg"), "${baseDir}", baseDir)
			err = os.MkdirAll(parentPkg, 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = os.MkdirAll(nestedPkg, 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = ioutil.WriteFile(filepath.Join(parentPkg, "Kptfile"), []byte(test.parentKptfile), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = ioutil.WriteFile(filepath.Join(childPkg, "Kptfile"), []byte(test.childKptfile), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = ioutil.WriteFile(filepath.Join(childPkg, "deploy.yaml"), []byte(test.childConfigFile), 0700)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if test.nestedKptfile != "" {
				err = ioutil.WriteFile(filepath.Join(nestedPkg, "Kptfile"), []byte(test.nestedKptfile), 0700)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				err = ioutil.WriteFile(filepath.Join(nestedPkg, "deploy.yaml"), []byte(test.nestedConfigFile), 0700)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

			out := &bytes.Buffer{}
			a := AutoSet{
				Writer:      out,
				PackagePath: childPkg,
			}

			err = a.SetInheritedSetters()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			actualChildConfigFile, err := ioutil.ReadFile(filepath.Join(childPkg, "deploy.yaml"))
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			actualChildKptfile, err := ioutil.ReadFile(filepath.Join(childPkg, "Kptfile"))
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if !assert.Equal(t, test.expectedChildConfigFile, string(actualChildConfigFile)) {
				t.FailNow()
			}

			if !assert.Equal(t, test.expectedChildKptfile, string(actualChildKptfile)) {
				t.FailNow()
			}

			if test.nestedKptfile != "" {
				actualNestedKptfile, err := ioutil.ReadFile(filepath.Join(nestedPkg, "Kptfile"))
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				if !assert.Equal(t, test.expectedChildKptfile, string(actualNestedKptfile)) {
					t.FailNow()
				}

				actualNestedConfigFile, err := ioutil.ReadFile(filepath.Join(nestedPkg, "deploy.yaml"))
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				if !assert.Equal(t, test.expectedChildConfigFile, string(actualNestedConfigFile)) {
					t.FailNow()
				}
			}

			actualParentKptfile, err := ioutil.ReadFile(filepath.Join(parentPkg, "Kptfile"))
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// parent package should not be modified
			if !assert.Equal(t, test.parentKptfile, string(actualParentKptfile)) {
				t.FailNow()
			}

			expectedOut := strings.ReplaceAll(test.expectedOut, "${childPkg}", childPkg)
			expectedOut = strings.ReplaceAll(expectedOut, "${nestedPkg}", nestedPkg)
			expectedOut = strings.ReplaceAll(expectedOut, "${parentPkgKptfile}", filepath.Join(parentPkg, kptfile.KptFileName))

			if !assert.Equal(t, expectedOut, out.String()) {
				t.FailNow()
			}
		})
	}
}

func TestCheckRequiredSettersSet(t *testing.T) {
	var tests = []struct {
		name             string
		inputOpenAPIfile string
		expectedError    bool
	}{
		{
			name: "required true, isSet false",
			inputOpenAPIfile: `
apiVersion: v1alpha1
kind: OpenAPIfile
openAPI:
  definitions:
    io.k8s.cli.setters.gcloud.project.projectNumber:
      description: hello world
      x-k8s-cli:
        setter:
          name: gcloud.project.projectNumber
          value: "123"
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
			name:             "no file, no error",
			inputOpenAPIfile: ``,
			expectedError:    false,
		},
		{
			name: "no setter defs, no error",
			inputOpenAPIfile: `apiVersion: v1alpha1
kind: OpenAPIfile`,
			expectedError: false,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			defer os.RemoveAll(dir)
			if test.inputOpenAPIfile != "" {
				err = ioutil.WriteFile(filepath.Join(dir, "Kptfile"), []byte(test.inputOpenAPIfile), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}
			err = CheckForRequiredSetters(dir)
			if test.expectedError && !assert.Error(t, err) {
				t.FailNow()
			}
			if !test.expectedError && !assert.NoError(t, err) {
				t.FailNow()
			}
		})
	}
}
