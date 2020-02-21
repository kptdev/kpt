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

	. "github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/kptfile/kptfileutil"

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
			ObjectMeta: yaml.ObjectMeta{Name: "cockroachdb"},
			APIVersion: TypeMeta.APIVersion,
			Kind:       TypeMeta.Kind},
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
		to       string
		from     string
		expected string
	}{
		{
			name: "add value",
			to: `
openAPI:
  definitions:
    io.k8s.cli.setters.image:
      x-k8s-cli:
        setter:
          name: "image"
          value: "nginx"
`,
			from: `
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
                    value: 1.7.9
`,
		},
		{
			name: "copy value",
			to: `
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
			from: `
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
                    value: 1.8.0
`,
		},
		{
			name: "replace values",
			to: `
`,
			from: `
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
        io.k8s.cli.setters.tag:
            x-k8s-cli:
                setter:
                    name: tag
                    value: 1.8.0
`,
		},
		{
			name: "keep values",
			to: `
openAPI:
  definitions:
    io.k8s.cli.setters.tag:
      x-k8s-cli:
        setter:
          name: "tag"
          value: "1.8.0"
`,
			from: `
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
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			kTo := KptFile{}
			if !assert.NoError(t, yaml.Unmarshal([]byte(test.to), &kTo)) {
				t.FailNow()
			}

			kFrom := KptFile{}
			if !assert.NoError(t, yaml.Unmarshal([]byte(test.from), &kFrom)) {
				t.FailNow()
			}

			err := kTo.MergeOpenAPI(kFrom)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			b, err := yaml.Marshal(kTo)
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
