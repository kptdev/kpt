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
	"strings"
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

func TestKptFile_MergeSubpackages(t *testing.T) {
	testCases := map[string]struct {
		updated  string
		local    string
		original string
		expected string
	}{
		"no updates in upstream or local": {
			updated: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			local: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			original: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
		},

		"additional subpackage added in local": {
			updated: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			local: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			original: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
`,
		},

		"additional subpackage added in upstream": {
			updated: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
`,
			local: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			original: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
`,
		},

		"subpackage removed from upstream": {
			updated: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			local: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
		},

		"subpackage removed from local": {
			updated: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			local: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
		},

		"all subpackages removed from local": {
			updated: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			local: `[]`,
			original: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `[]`,
		},

		"all subpackages removed from upstream": {
			updated: `[]`,
			local: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `[]`,
		},

		"subpackage deleted from upstream but changed in local": {
			updated: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
`,
			local: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: v1.0
    updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
    updateStrategy: resource-merge
`,
			expected: `
- localDir: bar
  upstream:
    git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
    updateStrategy: fast-forward
- localDir: foo
  upstream:
    git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: v1.0
    updateStrategy: resource-merge
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var updated []kptfilev1alpha2.Subpackage
			if !assert.NoError(t, yaml.Unmarshal([]byte(tc.updated), &updated)) {
				t.FailNow()
			}

			var local []kptfilev1alpha2.Subpackage
			if !assert.NoError(t, yaml.Unmarshal([]byte(tc.local), &local)) {
				t.FailNow()
			}

			var original []kptfilev1alpha2.Subpackage
			if !assert.NoError(t, yaml.Unmarshal([]byte(tc.original), &original)) {
				t.FailNow()
			}

			res, err := MergeSubpackages(local, updated, original)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			b, err := yaml.Marshal(res)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			node, err := yaml.Parse(string(b))
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			actual, err := node.String()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if !assert.Equal(t,
				strings.TrimSpace(tc.expected),
				strings.TrimSpace(actual)) {
				t.FailNow()
			}
		})
	}
}
