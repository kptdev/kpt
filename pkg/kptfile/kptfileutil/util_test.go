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
	"sigs.k8s.io/kustomize/kyaml/yaml/merge3"
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

func TestMerge(t *testing.T) {
	testCases := map[string]struct{
		origin      string
		update      string
		local       string
		expected    string
		err error
	} {
		// With no associative key, there is no merge, just a replacement
		// of the pipeline with upstream. This is aligned with the general behavior
		// of kyaml merge where in conflicts the upstream version win.s
		"no associative key, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/gen-folders
`,
			local: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/folder-ref
`,
			expected: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/gen-folders
`,
		},

		// When adding an associative key, we get a real merge of the pipeline.
		// In this case, we have an initial empty list in origin and different
		// functions are added in upstream and local. In this case the element
		// added in local are placed first in the resulting list.
		"associative key name, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: gen-folders
    image: gcr.io/kpt/gen-folders
`,
			local: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: folder-ref
    image: gcr.io/kpt/folder-ref
`,
			// The reordering of elements in the results is a bug in the
			// merge logic I think.
			expected: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: folder-ref
    image: gcr.io/kpt/folder-ref
  - image: gcr.io/kpt/gen-folders
    name: gen-folders 
`,
		},


		// Even with multiple elements added in both upstream and local, all
		// elements from local comes before upstream, and the order of elements
		// from each source is preserved. There is no lexicographical
		// ordering.
		"associative key name, multiple additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: z-upstream
    image: z-gcr.io/kpt/gen-folders
  - name: a-upstream
    image: a-gcr.io/kpt/gen-folders
`,
			local: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: x-local
    image: x-gcr.io/kpt/gen-folders
  - name: b-local
    image: b-gcr.io/kpt/gen-folders
`,
			expected: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: x-local
    image: x-gcr.io/kpt/gen-folders
  - name: b-local
    image: b-gcr.io/kpt/gen-folders
  - image: z-gcr.io/kpt/gen-folders
    name: z-upstream
  - image: a-gcr.io/kpt/gen-folders
    name: a-upstream
`,
		},


		// If elements with the same associative key are added in both upstream
		// and local, it will be merged. It will keep the location in the list
		// from local.
		"same element in both local and upstream does not create duplicate": {
			origin: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: gen-folder-upstream
    image: gcr.io/kpt/gen-folders
  - name: ref-folders
    image: gcr.io/kpt/ref-folders
    configMap:
      foo: bar
`,
			local: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: gcr.io/kpt/ref-folders
    configMap:
      bar: foo
  - name: gen-folder-local
    image: gcr.io/kpt/gen-folders
`,
			expected: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: gcr.io/kpt/ref-folders
    configMap:
      bar: foo
      foo: bar
  - name: gen-folder-local
    image: gcr.io/kpt/gen-folders
  - image: gcr.io/kpt/gen-folders
    name: gen-folder-upstream
`,
		},


		// If a field are set in both upstream and local, the value from
		// upstream will be chosen.
		"If there is a field-level conflict, upstream will win": {
			origin: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: gcr.io/kpt/ref-folders
    configMap:
      band: sleater-kinney
`,
			local: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: gcr.io/kpt/ref-folders
    configMap:
      band: Hüsker Dü
`,
			expected: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: ref-folders
    image: gcr.io/kpt/ref-folders
    configMap:
      band: sleater-kinney
`,
		},


		// KRM resources as parameters to functions are also merged.
		"KRM resources in the pipeline are merged": {
			origin: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: myCRD-gen
    image: gcr.io/kpt/crd-gen
    config:
      apiVersion: kpt.dev/v1alpha2
      kind: MyCRD
      metadata:
        name: foo
        namespace: default
        labels:
          upstream: def456
      spec:
        replicas: 4
        list:
        - a
        - b
`,
			local: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: myCRD-gen
    image: gcr.io/kpt/crd-gen
    config:
      apiVersion: kpt.dev/v1alpha2
      kind: MyCRD
      metadata:
        name: doo
        namespace: default
        labels:
          local: abc123
      spec:
        replicas: 42
        list:
        - a
        - c
`,
			expected: `
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: myCRD-gen
    image: gcr.io/kpt/crd-gen
    config:
      apiVersion: kpt.dev/v1alpha2
      kind: MyCRD
      metadata:
        name: foo
        namespace: default
        labels:
          local: abc123
          upstream: def456
      spec:
        replicas: 4
        list:
        - a
        - b
`,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actual, err := merge3.MergeStrings(tc.local, tc.origin, tc.update, true)
			if tc.err == nil {
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				if !assert.Equal(t,
					strings.TrimSpace(tc.expected), strings.TrimSpace(actual)) {
					t.FailNow()
				}
			} else {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, tc.err.Error(), err.Error()) {
					t.FailNow()
				}
			}
		})
	}
}



