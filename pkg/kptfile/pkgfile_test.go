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

package kptfile_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestReadFile tests the ReadFile function.
func TestReadFile(t *testing.T) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-pkgfile-read", testutil.TmpDirPrefix))
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(dir, KptFileName), []byte(`apiVersion: kpt.dev/v1alpha1
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

	f, err := kptfileutil.ReadFile(dir)
	assert.NoError(t, err)
	assert.Equal(t, KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: "cockroachdb",
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: TypeMeta.APIVersion,
				Kind:       TypeMeta.Kind},
		},
		Upstream: Upstream{
			Type: "git",
			Git: Git{
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
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-pkgfile-read", testutil.TmpDirPrefix))
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

	f, err := kptfileutil.ReadFile(dir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
	assert.Equal(t, KptFile{}, f)
}

// TestReadFile_failUnmarshal verifies an error is returned if the file contains any unrecognized fields.
func TestReadFile_failUnmarshal(t *testing.T) {
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-pkgfile-read", testutil.TmpDirPrefix))
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(dir, KptFileName), []byte(`apiVersion: kpt.dev/v1alpha1
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

	f, err := kptfileutil.ReadFile(dir)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "upstreamBadField not found")
	assert.Equal(t, KptFile{}, f)
}

func TestKptFile_MergeOpenAPI(t *testing.T) {
	tests := []struct {
		name     string
		updated  string
		local    string
		original string
		expected string
	}{
		{
			name: "add one delete one",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
`,
			original: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.image:
            x-k8s-cli:
                setter:
                    name: image
                    value: nginx
`,
		},
		{
			name: "keep locally changed value",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			original: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.image:
            x-k8s-cli:
                setter:
                    name: image
                    value: nginx
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.8.0
`,
		},
		{
			name: "and one and copy value from updated to local",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.1"
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			original: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.image:
            x-k8s-cli:
                setter:
                    name: image
                    value: nginx
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.8.1
`,
		},
		{
			name: "keep local",
			updated: `
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			original: `
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.8.0
`,
		},
		{
			name: "add definition from updated",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			local: `
`,
			original: `
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.8.0
`,
		},
		{
			name: "local, updated, original diverged",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
`,
			original: `
openAPI:
  definitions:
    io.k8s.cli.setters.nomatch:
      x-k8s-cli:
        setter:
          name: "nomatch"
          value: "something"
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.image:
            x-k8s-cli:
                setter:
                    name: image
                    value: nginx
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.7.9
`,
		},
		{
			name: "delete updated",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			original: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.image:
            x-k8s-cli:
                setter:
                    name: image
                    value: nginx
`,
		},
		{
			name: "keep deleted",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			original: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.7.9"
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.8.0
`,
		},
		{
			name: "no defs in origin",
			updated: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.1"
`,
			local: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			original: `
`,
			expected: `
openAPI:
    definitions:
        io.k8s.cli.setters.image:
            x-k8s-cli:
                setter:
                    name: image
                    value: nginx
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.8.1
`,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			kUpdated := KptFile{}
			if !assert.NoError(t, yaml.Unmarshal([]byte(test.updated), &kUpdated)) {
				t.FailNow()
			}

			kLocal := KptFile{}
			if !assert.NoError(t, yaml.Unmarshal([]byte(test.local), &kLocal)) {
				t.FailNow()
			}

			kOriginal := KptFile{}
			if !assert.NoError(t, yaml.Unmarshal([]byte(test.original), &kOriginal)) {
				t.FailNow()
			}

			mergedOpenAPI, err := MergeOpenAPI(kLocal.OpenAPI, kUpdated.OpenAPI, kOriginal.OpenAPI)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			kUpdated.OpenAPI = mergedOpenAPI

			b, err := yaml.Marshal(kUpdated)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if !assert.Equal(t,
				strings.TrimSpace(test.expected),
				strings.TrimSpace(string(b))) {
				t.FailNow()
			}
		})
	}
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
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			local: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			original: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
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
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			local: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			original: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
- localDir: bar
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
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
`,
			local: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			original: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
- localDir: bar
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
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			local: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
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
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			local: `
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			expected: `
- localDir: foo
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
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			local: `[]`,
			original: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
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
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
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
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
`,
			local: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: v1.0
  updateStrategy: resource-merge
`,
			original: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
  git:
      repo: github.com/GoogleContainerTools/kpt
      directory: /
      ref: master
  updateStrategy: resource-merge
`,
			expected: `
- localDir: bar
  git:
      repo: k8s.io/kubernetes
      directory: /pkg
      ref: master
  updateStrategy: fast-forward
- localDir: foo
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
			var updated []Subpackage
			if !assert.NoError(t, yaml.Unmarshal([]byte(tc.updated), &updated)) {
				t.FailNow()
			}

			var local []Subpackage
			if !assert.NoError(t, yaml.Unmarshal([]byte(tc.local), &local)) {
				t.FailNow()
			}

			var original []Subpackage
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

			if !assert.Equal(t,
				strings.TrimSpace(tc.expected),
				strings.TrimSpace(string(b))) {
				t.FailNow()
			}
		})
	}
}
