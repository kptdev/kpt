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

package kptfileutil

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestValidateInventory tests the ValidateInventory function.
func TestValidateInventory(t *testing.T) {
	// nil inventory should not validate
	isValid, err := ValidateInventory(nil)
	if isValid || err == nil {
		t.Errorf("nil inventory should not validate")
	}
	// Empty inventory should not validate
	inv := &kptfilev1alpha2.Inventory{}
	isValid, err = ValidateInventory(inv)
	if isValid || err == nil {
		t.Errorf("empty inventory should not validate")
	}
	// Empty inventory parameters strings should not validate
	inv = &kptfilev1alpha2.Inventory{
		Namespace:   "",
		Name:        "",
		InventoryID: "",
	}
	isValid, err = ValidateInventory(inv)
	if isValid || err == nil {
		t.Errorf("empty inventory parameters strings should not validate")
	}
	// Inventory with non-empty namespace, name, and id should validate.
	inv = &kptfilev1alpha2.Inventory{
		Namespace:   "test-namespace",
		Name:        "test-name",
		InventoryID: "test-id",
	}
	isValid, err = ValidateInventory(inv)
	if !isValid || err != nil {
		t.Errorf("inventory with non-empty namespace, name, and id should validate")
	}
}

// TestReadFile tests the ReadFile function.
func TestReadFile(t *testing.T) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-pkgfile-read", "test-kpt"))
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(dir, kptfilev1alpha2.KptFileName), []byte(`apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: cockroachdb
upstreamLock:
  type: git
  gitLock:
    commit: dd7adeb5492cca4c24169cecee023dbe632e5167
    directory: staging/cockroachdb
    ref: refs/heads/owners-update
    repo: https://github.com/kubernetes/examples
`), 0600)
	assert.NoError(t, err)

	f, err := ReadFile(dir)
	assert.NoError(t, err)
	assert.Equal(t, kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: "cockroachdb",
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.TypeMeta.APIVersion,
				Kind:       kptfilev1alpha2.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: "git",
			GitLock: &kptfilev1alpha2.GitLock{
				Commit:    "dd7adeb5492cca4c24169cecee023dbe632e5167",
				Directory: "staging/cockroachdb",
				Ref:       "refs/heads/owners-update",
				Repo:      "https://github.com/kubernetes/examples",
			},
		},
	}, f)
}

// TestReadFile_failRead verifies an error is returned if the file cannot be read
func TestReadFile_failRead(t *testing.T) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-pkgfile-read", "test-kpt"))
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(dir, " KptFileError"), []byte(`apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: cockroachdb
upstream:
  type: git
  git:
    commit: dd7adeb5492cca4c24169cecee023dbe632e5167
    directory: staging/cockroachdb
    ref: refs/heads/owners-update
    repo: https://github.com/kubernetes/examples
`), 0600)
	assert.NoError(t, err)

	f, err := ReadFile(dir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
	assert.Equal(t, kptfilev1alpha2.KptFile{}, f)
}

// TestReadFile_failUnmarshal verifies an error is returned if the file contains any unrecognized fields.
func TestReadFile_failUnmarshal(t *testing.T) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-pkgfile-read", "test-kpt"))
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(dir, kptfilev1alpha2.KptFileName), []byte(`apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: cockroachdb
upstreamBadField:
  type: git
  git:
    commit: dd7adeb5492cca4c24169cecee023dbe632e5167
    directory: staging/cockroachdb
    ref: refs/heads/owners-update
    repo: https://github.com/kubernetes/examples
`), 0600)
	assert.NoError(t, err)

	f, err := ReadFile(dir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "upstreamBadField not found")
	assert.Equal(t, kptfilev1alpha2.KptFile{}, f)
}

func TestMerge(t *testing.T) {

}
