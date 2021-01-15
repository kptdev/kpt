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

package cmdsearch

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/cmd/config/ext"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
)

type test struct {
	name              string
	input             string
	inputKptfile      string
	args              []string
	out               string
	expectedResources string
	errMsg            string
}

func TestSearchCommand(t *testing.T) {
	for _, tests := range [][]test{searchReplaceCases, putPatternCases} {
		for i := range tests {
			test := tests[i]
			t.Run(test.name, func(t *testing.T) {
				ext.KRMFileName = func() string {
					return kptfile.KptFileName
				}
				baseDir, err := ioutil.TempDir("", "")
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				defer os.RemoveAll(baseDir)

				r, err := ioutil.TempFile(baseDir, "k8s-cli-*.yaml")
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				defer os.Remove(r.Name())
				err = ioutil.WriteFile(r.Name(), []byte(test.input), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				if test.inputKptfile != "" {
					err = ioutil.WriteFile(filepath.Join(baseDir, kptfile.KptFileName), []byte(test.inputKptfile), 0600)
					if !assert.NoError(t, err) {
						t.FailNow()
					}
				} else {
					err = ioutil.WriteFile(filepath.Join(baseDir, kptfile.KptFileName), []byte(`apiVersion: v1alpha1
kind: Kptfile`), 0600)
					if !assert.NoError(t, err) {
						t.FailNow()
					}
				}

				runner := NewSearchRunner("")
				out := &bytes.Buffer{}
				runner.Command.SetOut(out)
				runner.Command.SetArgs(append([]string{baseDir}, test.args...))
				err = runner.Command.Execute()
				if test.errMsg != "" {
					if !assert.NotNil(t, err) {
						t.FailNow()
					}
					if !assert.Contains(t, err.Error(), test.errMsg) {
						t.FailNow()
					}
				}

				if test.errMsg == "" && !assert.NoError(t, err) {
					t.FailNow()
				}

				// normalize path format for windows
				actualNormalized := strings.ReplaceAll(
					strings.ReplaceAll(out.String(), "\\", "/"),
					"//", "/")

				expected := strings.ReplaceAll(test.out, "${baseDir}", baseDir)
				expected = strings.ReplaceAll(expected, "${filePath}", filepath.Base(r.Name()))
				expectedNormalized := strings.ReplaceAll(
					strings.ReplaceAll(expected, "\\", "/"),
					"//", "/")

				if test.errMsg == "" && !assert.Equal(t, expectedNormalized, actualNormalized) {
					t.FailNow()
				}

				actualResources, err := ioutil.ReadFile(r.Name())
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				if !assert.Equal(t,
					strings.TrimSpace(test.expectedResources),
					strings.TrimSpace(string(actualResources))) {
					t.FailNow()
				}
			})
		}
	}
}

func TestSearchSubPackages(t *testing.T) {
	var tests = []struct {
		name    string
		dataset string
		args    []string
		out     string
		errMsg  string
	}{
		{
			name:    "search-replace-recurse-subpackages",
			dataset: "dataset-with-autosetters",
			args:    []string{"--by-value", "myspace", "--put-literal", "otherspace"},
			out: `${baseDir}/mysql/deployment.yaml
fieldPath: metadata.namespace
value: otherspace # {"$openapi":"gcloud.core.project"}

${baseDir}/mysql/nosetters/deployment.yaml
fieldPath: metadata.namespace
value: otherspace

${baseDir}/mysql/storage/deployment.yaml
fieldPath: metadata.namespace
value: otherspace # {"$openapi":"gcloud.core.project"}

Mutated 3 field(s)
`,
		},
		{
			name:    "search-recurse-subpackages",
			dataset: "dataset-with-autosetters",
			args:    []string{"--by-value", "mysql"},
			out: `${baseDir}/mysql/deployment.yaml
fieldPath: spec.template.spec.containers[0].name
value: mysql

Matched 1 field(s)
`,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			testDataDir := filepath.Join("../", "testutil", "testdata")
			fieldmeta.SetShortHandRef("$kpt-set")
			sourceDir := filepath.Join(testDataDir, test.dataset)
			baseDir, err := ioutil.TempDir("", "")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			err = copyutil.CopyDir(sourceDir, baseDir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			ext.KRMFileName = func() string {
				return kptfile.KptFileName
			}
			defer os.RemoveAll(baseDir)
			runner := NewSearchRunner("")
			out := &bytes.Buffer{}
			runner.Command.SetOut(out)
			runner.Command.SetArgs(append([]string{baseDir}, test.args...))
			err = runner.Command.Execute()
			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}

			// normalize path format for windows
			actualNormalized := strings.ReplaceAll(
				strings.ReplaceAll(out.String(), "\\", "/"),
				"//", "/")

			expected := strings.ReplaceAll(test.out, "${baseDir}", baseDir)
			expectedNormalized := strings.ReplaceAll(
				strings.ReplaceAll(expected, "\\", "/"),
				"//", "/")

			if test.errMsg == "" && !assert.Equal(t, expectedNormalized, actualNormalized) {
				t.FailNow()
			}
		})
	}
}
