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

package kio_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/kio"
)

// setup creates directories and files for testing
type setup struct {
	// root is the tmp directory
	root string
}

// setupDirectories creates directories for reading test configuration from
func setupDirectories(t *testing.T, dirs ...string) setup {
	d, err := ioutil.TempDir("", "kpt-test")
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	err = os.Chdir(d)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	for _, s := range dirs {
		err = os.MkdirAll(s, 0700)
		if !assert.NoError(t, err) {
			assert.FailNow(t, err.Error())
		}
	}
	return setup{root: d}
}

// writeFile writes a file under the test directory
func (s setup) writeFile(t *testing.T, path string, value []byte) {
	err := os.MkdirAll(filepath.Dir(filepath.Join(s.root, path)), 0700)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	err = ioutil.WriteFile(filepath.Join(s.root, path), value, 0600)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
}

// clean deletes the test config
func (s setup) clean() {
	os.RemoveAll(s.root)
}

var readFileA = []byte(`---
a: b #first
---
c: d # second
`)

var readFileB = []byte(`# second thing
e: f
g:
  h:
  - i # has a list
  - j
`)

var kptfile = []byte(``)

func TestLocalPackageReader_Read_empty(t *testing.T) {
	var r LocalPackageReader
	nodes, err := r.Read()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "must specify package path")
	}
	assert.Nil(t, nodes)

}

func TestLocalPackageReader_Read_pkg(t *testing.T) {
	s := setupDirectories(t, filepath.Join("a", "b"), filepath.Join("a", "c"))
	defer s.clean()
	s.writeFile(t, filepath.Join("a_test.yaml"), readFileA)
	s.writeFile(t, filepath.Join("b_test.yaml"), readFileB)

	paths := []struct {
		path string
	}{
		{path: "./"},
		{path: s.root},
	}
	for _, p := range paths {
		rfr := LocalPackageReader{PackagePath: p.path}
		nodes, err := rfr.Read()
		if !assert.NoError(t, err) {
			return
		}

		if !assert.Len(t, nodes, 3) {
			return
		}
		expected := []string{
			`a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: .
    kpt.dev/kio/path: a_test.yaml
`,
			`c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/package: .
    kpt.dev/kio/path: a_test.yaml
`,
			`# second thing
e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: .
    kpt.dev/kio/path: b_test.yaml
`,
		}
		for i := range nodes {
			val, err := nodes[i].String()
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Equal(t, expected[i], val) {
				return
			}
		}
	}
}

func TestLocalPackageReader_Read_file(t *testing.T) {
	s := setupDirectories(t, filepath.Join("a", "b"), filepath.Join("a", "c"))
	defer s.clean()
	s.writeFile(t, filepath.Join("a_test.yaml"), readFileA)
	s.writeFile(t, filepath.Join("b_test.yaml"), readFileB)

	paths := []struct {
		path string
	}{
		{path: "./"},
		{path: s.root},
	}
	for _, p := range paths {
		rfr := LocalPackageReader{PackagePath: filepath.Join(p.path, "a_test.yaml")}
		nodes, err := rfr.Read()
		if !assert.NoError(t, err) {
			return
		}

		if !assert.Len(t, nodes, 2) {
			return
		}
		expected := []string{
			`a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: .
    kpt.dev/kio/path: a_test.yaml
`,
			`c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/package: .
    kpt.dev/kio/path: a_test.yaml
`,
		}
		for i := range nodes {
			val, err := nodes[i].String()
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Equal(t, expected[i], val) {
				return
			}
		}
	}
}

func TestLocalPackageReader_Read_pkgOmitAnnotations(t *testing.T) {
	s := setupDirectories(t, filepath.Join("a", "b"), filepath.Join("a", "c"))
	defer s.clean()
	s.writeFile(t, filepath.Join("a_test.yaml"), readFileA)
	s.writeFile(t, filepath.Join("b_test.yaml"), readFileB)

	paths := []struct {
		path string
	}{
		{path: "./"},
		{path: s.root},
	}
	for _, p := range paths {

		// empty path
		rfr := LocalPackageReader{PackagePath: p.path, OmitReaderAnnotations: true}
		nodes, err := rfr.Read()
		if !assert.NoError(t, err) {
			return
		}

		if !assert.Len(t, nodes, 3) {
			return
		}
		expected := []string{
			`a: b #first
`,
			`c: d # second
`,
			`# second thing
e: f
g:
  h:
  - i # has a list
  - j
`,
		}
		for i := range nodes {
			val, err := nodes[i].String()
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Equal(t, expected[i], val) {
				return
			}
		}
	}
}

func TestLocalPackageReader_Read_nestedDirs(t *testing.T) {
	s := setupDirectories(t, filepath.Join("a", "b"), filepath.Join("a", "c"))
	defer s.clean()
	s.writeFile(t, filepath.Join("a", "b", "a_test.yaml"), readFileA)
	s.writeFile(t, filepath.Join("a", "b", "b_test.yaml"), readFileB)

	paths := []struct {
		path string
	}{
		{path: "./"},
		{path: s.root},
	}
	for _, p := range paths {
		// empty path
		rfr := LocalPackageReader{PackagePath: p.path}
		nodes, err := rfr.Read()
		if !assert.NoError(t, err) {
			assert.FailNow(t, err.Error())
		}

		if !assert.Len(t, nodes, 3) {
			return
		}
		expected := []string{
			`a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`,
			`c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`,
			`# second thing
e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/b_test.yaml
`,
		}
		for i := range nodes {
			val, err := nodes[i].String()
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Equal(t, expected[i], val) {
				return
			}
		}
	}
}

func TestLocalPackageReader_Read_matchRegex(t *testing.T) {
	s := setupDirectories(t, filepath.Join("a", "b"), filepath.Join("a", "c"))
	defer s.clean()
	s.writeFile(t, filepath.Join("a", "b", "a_test.yaml"), readFileA)
	s.writeFile(t, filepath.Join("a", "b", "b_test.yaml"), readFileB)

	// empty path
	rfr := LocalPackageReader{PackagePath: s.root, MatchFilesGlob: []string{`a*.yaml`}}
	nodes, err := rfr.Read()
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	if !assert.Len(t, nodes, 2) {
		assert.FailNow(t, "wrong number items")
	}

	val, err := nodes[0].String()
	assert.NoError(t, err)
	assert.Equal(t, `a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`, val)

	val, err = nodes[1].String()
	assert.NoError(t, err)
	assert.Equal(t, `c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`, val)
}

func TestLocalPackageReader_Read_skipSubpackage(t *testing.T) {
	s := setupDirectories(t, filepath.Join("a", "b"), filepath.Join("a", "c"))
	defer s.clean()
	s.writeFile(t, filepath.Join("a", "b", "a_test.yaml"), readFileA)
	s.writeFile(t, filepath.Join("a", "c", "c_test.yaml"), readFileB)
	s.writeFile(t, filepath.Join("a", "c", "Kptfile"), kptfile)

	// empty path
	rfr := LocalPackageReader{PackagePath: s.root}
	nodes, err := rfr.Read()
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	if !assert.Len(t, nodes, 2) {
		assert.FailNow(t, "wrong number items")
	}

	val, err := nodes[0].String()
	assert.NoError(t, err)
	assert.Equal(t, `a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`, val)

	val, err = nodes[1].String()
	assert.NoError(t, err)
	assert.Equal(t, `c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`, val)
}

func TestLocalPackageReader_Read_includeSubpackage(t *testing.T) {
	s := setupDirectories(t, filepath.Join("a", "b"), filepath.Join("a", "c"))
	defer s.clean()
	s.writeFile(t, filepath.Join("a", "b", "a_test.yaml"), readFileA)
	s.writeFile(t, filepath.Join("a", "c", "c_test.yaml"), readFileB)
	s.writeFile(t, filepath.Join("a", "c", "Kptfile"), kptfile)

	// empty path
	rfr := LocalPackageReader{PackagePath: s.root, IncludeSubpackages: true}
	nodes, err := rfr.Read()
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	if !assert.Len(t, nodes, 3) {
		assert.FailNow(t, "wrong number items")
	}
	val, err := nodes[0].String()
	assert.NoError(t, err)
	assert.Equal(t, `a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`, val)

	val, err = nodes[1].String()
	assert.NoError(t, err)
	assert.Equal(t, `c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/package: a/b
    kpt.dev/kio/path: a/b/a_test.yaml
`, val)

	val, err = nodes[2].String()
	assert.NoError(t, err)
	assert.Equal(t, `# second thing
e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/package: a/c
    kpt.dev/kio/path: a/c/c_test.yaml
`, val)
}
