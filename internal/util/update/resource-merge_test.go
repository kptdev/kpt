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

package update

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

func TestMergeSubPackages(t *testing.T) {
	// this test simulates the end to end scenario of merging subpackages
	// original and updated/upstream are same initially and they deviate
	// due to upstream changes, localDataSet deviates from original as the
	// setters are set with local values
	updatedDataSet := "dataset-with-autosetters/mysql"
	originalDataSet := "dataset-with-autosetters/mysql"
	localDataset := "dataset-with-autosetters-set/mysql"

	// reset the openAPI afterward
	openapi.ResetOpenAPI()
	defer openapi.ResetOpenAPI()
	testDataDir := filepath.Join("../../", "testutil", "testdata")
	updatedRoot, err := ioutil.TempDir("", "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	localRoot, err := ioutil.TempDir("", "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	originalRoot, err := ioutil.TempDir("", "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// updated is the upstream dataset with setters not set
	err = copyutil.CopyDir(filepath.Join(testDataDir, updatedDataSet), updatedRoot)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// original is the upstream dataset with setters not set
	err = copyutil.CopyDir(filepath.Join(testDataDir, originalDataSet), originalRoot)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// local root has the setters set to the local values
	err = copyutil.CopyDir(filepath.Join(testDataDir, localDataset), localRoot)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	defer os.RemoveAll(updatedRoot)
	defer os.RemoveAll(localRoot)
	defer os.RemoveAll(originalRoot)

	// modify updated/upstream by adding a new setter definition to one of the subpackages Kptfile
	nosettersUpdated := `apiVersion: krm.dev/v1alpha1
kind: Kptfile
metadata:
  name: nosetters
packageMetadata:
  shortDescription: sample description
openAPI:
  definitions:
    io.k8s.cli.setters.new-setter:
      x-k8s-cli:
        setter:
          name: new-setter
          value: some-value
`

	err = ioutil.WriteFile(filepath.Join(updatedRoot, "nosetters", "Kptfile"), []byte(nosettersUpdated), 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// updated has deviated from original with new setter definition added above
	// local has deviated from original with auto-setters set to local values
	// if update is triggered now, it merges the subpackages using MergeSubPkgsKptfiles
	err = MergeSubPackages(localRoot, updatedRoot, originalRoot)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// the updated subpackage file must have the setters set to the values in local
	actualSubPkgFile, err := ioutil.ReadFile(filepath.Join(updatedRoot, "storage", "deployment.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expectedSubPkgFile, err := ioutil.ReadFile(filepath.Join(localRoot, "storage", "deployment.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, string(expectedSubPkgFile), string(actualSubPkgFile)) {
		t.FailNow()
	}

	// the updated root package file must remain the same as this method should only merge subpackages
	actualRootPkgFile, err := ioutil.ReadFile(filepath.Join(updatedRoot, "deployment.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expectedRootPkgFile, err := ioutil.ReadFile(filepath.Join(testDataDir, updatedDataSet, "deployment.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, string(expectedRootPkgFile), string(actualRootPkgFile)) {
		t.FailNow()
	}

	// make sure that the updated new-setter definition in nosetters subpackage is pulled onto local
	actualSubPkgKptfile, err := ioutil.ReadFile(filepath.Join(localRoot, "nosetters", "Kptfile"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expectedSubPkgKptfile, err := ioutil.ReadFile(filepath.Join(updatedRoot, "nosetters", "Kptfile"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, string(expectedSubPkgKptfile), string(actualSubPkgKptfile)) {
		t.FailNow()
	}

	// delete the Kptfile in nosetters subpackage in the upstream and make sure it is retained on
	// local
	err = os.Remove(filepath.Join(updatedRoot, "nosetters", "Kptfile"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = MergeSubPackages(localRoot, updatedRoot, originalRoot)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	actualSubPkgKptfile, err = ioutil.ReadFile(filepath.Join(localRoot, "nosetters", "Kptfile"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// make sure that the local Kptfile is retained even if the upstream Kptfile is deleted
	if !assert.Equal(t, nosettersUpdated, string(actualSubPkgKptfile)) {
		t.FailNow()
	}

	// delete the Kptfile in nosetters subpackage at the origin and make sure it is retained on
	// local
	err = os.Remove(filepath.Join(originalRoot, "nosetters", "Kptfile"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = MergeSubPackages(localRoot, updatedRoot, originalRoot)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	actualSubPkgKptfile, err = ioutil.ReadFile(filepath.Join(localRoot, "nosetters", "Kptfile"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// make sure that the local Kptfile is retained even if the upstream Kptfile is deleted
	if !assert.Equal(t, nosettersUpdated, string(actualSubPkgKptfile)) {
		t.FailNow()
	}
}
