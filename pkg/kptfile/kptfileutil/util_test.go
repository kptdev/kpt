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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
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
	inv := &kptfilev1.Inventory{}
	isValid, err = ValidateInventory(inv)
	if isValid || err == nil {
		t.Errorf("empty inventory should not validate")
	}
	// Empty inventory parameters strings should not validate
	inv = &kptfilev1.Inventory{
		Namespace:   "",
		Name:        "",
		InventoryID: "",
	}
	isValid, err = ValidateInventory(inv)
	if isValid || err == nil {
		t.Errorf("empty inventory parameters strings should not validate")
	}
	// Inventory with non-empty namespace, name, and id should validate.
	inv = &kptfilev1.Inventory{
		Namespace:   "test-namespace",
		Name:        "test-name",
		InventoryID: "test-id",
	}
	isValid, err = ValidateInventory(inv)
	if !isValid || err != nil {
		t.Errorf("inventory with non-empty namespace, name, and id should validate")
	}
}

func TestUpdateKptfile(t *testing.T) {
	writeKptfileToTemp := func(name string, content string) string {
		dir, err := ioutil.TempDir("", name)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		err = ioutil.WriteFile(filepath.Join(dir, kptfilev1.KptFileName), []byte(content), 0600)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		return dir
	}

	testCases := map[string]struct {
		origin         string
		updated        string
		local          string
		updateUpstream bool
		expected       string
	}{
		"no pipeline and no upstream info": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: base
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: base
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
		},

		"upstream information is not copied from upstream unless updateUpstream is true": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/GoogleContainerTools/kpt
    directory: /
    ref: v1
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/GoogleContainerTools/kpt
    directory: /
    ref: v2
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
		},

		"upstream information is copied from upstream when updateUpstream is true": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/GoogleContainerTools/kpt
    directory: /
    ref: v1
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/GoogleContainerTools/kpt
    directory: /
    ref: v2
upstreamLock:
  type: git
  git:
    repo: github.com/GoogleContainerTools/kpt
    directory: /
    ref: v2
    commit: abc123
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updateUpstream: true,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
upstream:
  type: git
  git:
    repo: github.com/GoogleContainerTools/kpt
    directory: /
    ref: v2
upstreamLock:
  type: git
  git:
    repo: github.com/GoogleContainerTools/kpt
    directory: /
    ref: v2
    commit: abc123
`,
		},

		"pipeline in local remains if there are no changes in upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: foo:bar
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: foo:bar
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
      configMap:
        foo: bar
    - image: foo:bar
`,
			updateUpstream: true,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
      configMap:
        foo: bar
    - image: foo:bar
`,
		},

		"pipeline remains if it is only added locally": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
    - image: foo:bar
`,
			updateUpstream: true,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
    - image: foo:bar
`,
		},

		"pipeline in local is emptied if it is gone from upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: foo:bar
`,
			updated: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline:
  mutators:
    - image: my:image
    - image: foo:bar
`,
			updateUpstream: false,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: foo
pipeline: {}
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			files := map[string]string{
				"origin":  tc.origin,
				"updated": tc.updated,
				"local":   tc.local,
			}
			dirs := make(map[string]string)
			for n, content := range files {
				dir := writeKptfileToTemp(n, content)
				dirs[n] = dir
			}
			defer func() {
				for _, p := range dirs {
					_ = os.RemoveAll(p)
				}
			}()

			err := UpdateKptfile(dirs["local"], dirs["updated"], dirs["origin"], tc.updateUpstream)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			c, err := ioutil.ReadFile(filepath.Join(dirs["local"], kptfilev1.KptFileName))
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.Equal(t, strings.TrimSpace(tc.expected)+"\n", string(c))
		})
	}
}

func TestMerge(t *testing.T) {
	testCases := map[string]struct {
		origin   string
		update   string
		local    string
		expected string
		err      error
	}{
		// With no associative key, there is no merge, just a replacement
		// of the pipeline with upstream. This is aligned with the general behavior
		// of kyaml merge where in conflicts the upstream version win.s
		"no associative key, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/gen-folders
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/folder-ref
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/folder-ref
  - image: gcr.io/kpt/gen-folders
`,
		},

		"exec: no associative key, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - exec: gen-folders
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - exec: folder-ref
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - exec: folder-ref
  - exec: gen-folders
`,
		},

		"add new setter in upstream, update local setter value": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.0.1
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.0.1
        new-setter: new-setter-value // new setter is added
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        image: nginx
        tag: 1.2.0 // value of tag is updated
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/apply-setters:v0.1
    configMap:
      image: nginx
      new-setter: new-setter-value // new setter is added
      tag: 1.2.0 // value of tag is updated
`,
		},

		"both upstream and local configPath is updated, take upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters-updated.yaml
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters-local.yaml
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/apply-setters:v0.1
    configPath: setters-updated.yaml
`,
		},

		"both upstream and local version is updated, take upstream": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1.2
      configPath: setters.yaml
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1.1
      configPath: setters.yaml
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/apply-setters:v0.1.2
    configPath: setters.yaml
`,
		},

		"newly added upstream function": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/generate-folders:v0.1
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: foo
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/apply-setters:v0.1
    configPath: setters.yaml
  - image: gcr.io/kpt-fn/set-namespace:v0.1
    configMap:
      namespace: foo
  - image: gcr.io/kpt-fn/generate-folders:v0.1
`,
		},

		"deleted function in the upstream, deleted on local if not changed": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/generate-folders:v0.1
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configPath: setters.yaml
    - image: gcr.io/kpt-fn/generate-folders:v0.1
    - image: gcr.io/kpt-fn/set-namespace:v0.1
      configMap:
        namespace: foo
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  validators:
  - image: gcr.io/kpt-fn/apply-setters:v0.1
    configPath: setters.yaml
  - image: gcr.io/kpt-fn/set-namespace:v0.1
    configMap:
      namespace: foo
`,
		},

		"multiple declarations of same function": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar-new
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/generate-folders:v0.1
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        app: db
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: foo
      put-value: bar-new
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
`,
		},

		"add function at random location with name specified": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar-new
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: my-new-function
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
    - image: gcr.io/kpt-fn/generate-folders:v0.1
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        app: db
    - image: gcr.io/kpt-fn/search-replace:v0.1
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: foo
      put-value: bar-new
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
`,
		},

		"Ideal deterministic behavior: add function at random location with name specified in all sources": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr1
      configMap:
        by-value: foo
        put-value: bar-new
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${updated-setter-name}
`,
			local: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: my-new-function
      configMap:
        by-value: YOUR_TEAM
        put-value: my-team
    - image: gcr.io/kpt-fn/generate-folders:v0.1
      name: gf1
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr1
      configMap:
        by-value: foo
        put-value: bar
    - image: gcr.io/kpt-fn/set-labels:v0.1
      name: sl1
      configMap:
        app: db
    - image: gcr.io/kpt-fn/search-replace:v0.1
      name: sr2
      configMap:
        by-value: abc
        put-comment: ${some-setter-name}
`,
			expected: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: YOUR_TEAM
      put-value: my-team
    name: my-new-function
  - image: gcr.io/kpt-fn/generate-folders:v0.1
    name: gf1
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: foo
      put-value: bar-new
    name: sr1
  - image: gcr.io/kpt-fn/set-labels:v0.1
    configMap:
      app: db
    name: sl1
  - image: gcr.io/kpt-fn/search-replace:v0.1
    configMap:
      by-value: abc
      put-comment: ${updated-setter-name}
    name: sr2
`,
		},

		// When adding an associative key, we get a real merge of the pipeline.
		// In this case, we have an initial empty list in origin and different
		// functions are added in upstream and local. In this case the element
		// added in local are placed first in the resulting list.
		"associative key name, additions in both upstream and local": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - name: gen-folders
    image: gcr.io/kpt/gen-folders
`,
			local: `
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/folder-ref
    name: folder-ref
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
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: x-gcr.io/kpt/gen-folders
    name: x-local
  - image: b-gcr.io/kpt/gen-folders
    name: b-local
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
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/ref-folders
    configMap:
      bar: foo
      foo: bar
    name: ref-folders
  - image: gcr.io/kpt/gen-folders
    name: gen-folder-local
  - image: gcr.io/kpt/gen-folders
    name: gen-folder-upstream
`,
		},

		// If a field are set in both upstream and local, the value from
		// upstream will be chosen.
		"If there is a field-level conflict, upstream will win": {
			origin: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
`,
			update: `
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
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
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: pipeline
pipeline:
  mutators:
  - image: gcr.io/kpt/ref-folders
    configMap:
      band: sleater-kinney
    name: ref-folders
`,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			localKf, err := pkg.DecodeKptfile(strings.NewReader(tc.local))
			assert.NoError(t, err)
			updatedKf, err := pkg.DecodeKptfile(strings.NewReader(tc.update))
			assert.NoError(t, err)
			originKf, err := pkg.DecodeKptfile(strings.NewReader(tc.origin))
			assert.NoError(t, err)
			err = merge(localKf, updatedKf, originKf)
			if tc.err == nil {
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				actual, err := yaml.Marshal(localKf)
				assert.NoError(t, err)
				if !assert.Equal(t,
					strings.TrimSpace(tc.expected), strings.TrimSpace(string(actual))) {
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
