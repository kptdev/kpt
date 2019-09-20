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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

// TestLocalPackageWriter_Write tests:
// - ReaderAnnotations are cleared when writing the Resources
func TestLocalPackageWriter_Write(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	w := LocalPackageWriter{PackagePath: d}
	err := w.Write([]*yaml.RNode{node2, node1, node3})
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	b, err := ioutil.ReadFile(filepath.Join(d, "a", "b", "a_test.yaml"))
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	assert.Equal(t, `a: b #first
---
c: d # second
`, string(b))

	b, err = ioutil.ReadFile(filepath.Join(d, "a", "b", "b_test.yaml"))
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	assert.Equal(t, `e: f
g:
  h:
  - i # has a list
  - j
`, string(b))
}

// TestLocalPackageWriter_Write_keepReaderAnnotations tests:
// - ReaderAnnotations are kept when writing the Resources
func TestLocalPackageWriter_Write_keepReaderAnnotations(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	w := LocalPackageWriter{PackagePath: d, KeepReaderAnnotations: true}
	err := w.Write([]*yaml.RNode{node2, node1, node3})
	if !assert.NoError(t, err) {
		return
	}

	b, err := ioutil.ReadFile(filepath.Join(d, "a", "b", "a_test.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: a/b/a_test.yaml
    kpt.dev/kio/mode: 384
---
c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/path: a/b/a_test.yaml
    kpt.dev/kio/mode: 384
`, string(b))

	b, err = ioutil.ReadFile(filepath.Join(d, "a", "b", "b_test.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: a/b/b_test.yaml
    kpt.dev/kio/mode: 384
`, string(b))
}

// TestLocalPackageWriter_Write_clearAnnotations tests:
// - ClearAnnotations are removed from Resources
func TestLocalPackageWriter_Write_clearAnnotations(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	w := LocalPackageWriter{PackagePath: d, ClearAnnotations: []string{"kpt.dev/kio/mode"}}
	err := w.Write([]*yaml.RNode{node2, node1, node3})
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	b, err := ioutil.ReadFile(filepath.Join(d, "a", "b", "a_test.yaml"))
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	assert.Equal(t, `a: b #first
---
c: d # second
`, string(b))

	b, err = ioutil.ReadFile(filepath.Join(d, "a", "b", "b_test.yaml"))
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	assert.Equal(t, `e: f
g:
  h:
  - i # has a list
  - j
`, string(b))
}

// TestLocalPackageWriter_Write_failRelativePath tests:
// - If a relative path above the package is defined, write fails
func TestLocalPackageWriter_Write_failRelativePath(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/../../../b_test.yaml"
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "resource must be written under package")
	}
}

// TestLocalPackageWriter_Write_multipleModes tests:
// - If multiple file perm modes are specified for the same file, fail
func TestLocalPackageWriter_Write_multipleModes(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/c/../b_test.yaml" # use a different path, should still collide
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	node5, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/b_test.yaml"
    kpt.dev/kio/mode: 448
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node5, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "conflicting file modes")
	}
}

// TestLocalPackageWriter_Write_invalidMode tests:
// - If a non-int mode is given, fail
func TestLocalPackageWriter_Write_invalidMode(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/b_test.yaml" # use a different path, should still collide
    kpt.dev/kio/mode: a
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unable to parse kpt.dev/kio/mode")
	}
}

// TestLocalPackageWriter_Write_invalidIndex tests:
// - If a non-int index is given, fail
func TestLocalPackageWriter_Write_invalidIndex(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: a
    kpt.dev/kio/path: "a/b/b_test.yaml" # use a different path, should still collide
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unable to parse kpt.dev/kio/index")
	}
}

// TestLocalPackageWriter_Write_absPath tests:
// - If kpt.dev/kio/path is absolute, fail
func TestLocalPackageWriter_Write_absPath(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(fmt.Sprintf(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: a
    kpt.dev/kio/path: "%s/a/b/b_test.yaml" # use a different path, should still collide
    kpt.dev/kio/mode: 384
`, d))
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "package paths may not be absolute paths")
	}
}

// TestLocalPackageWriter_Write_missingIndex tests:
// - If kpt.dev/kio/path is missing, fail
func TestLocalPackageWriter_Write_missingPath(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: a
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "kpt.dev/kio/path")
	}
}

// TestLocalPackageWriter_Write_missingIndex tests:
// - If kpt.dev/kio/index is missing, fail
func TestLocalPackageWriter_Write_missingIndex(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/path: a/a.yaml
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "kpt.dev/kio/index")
	}
}

// TestLocalPackageWriter_Write_missingMode tests:
// - If kpt.dev/kio/mode is missing, fail
func TestLocalPackageWriter_Write_missingMode(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/path: a/a.yaml
    kpt.dev/kio/index: 0
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "kpt.dev/kio/mode")
	}
}

// TestLocalPackageWriter_Write_pathIsDir tests:
// - If  kpt.dev/kio/path is a directory, fail
func TestLocalPackageWriter_Write_pathIsDir(t *testing.T) {
	d, node1, node2, node3 := getWriterInputs(t)
	defer os.RemoveAll(d)

	node4, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/mode: 384
    kpt.dev/kio/path: a/
    kpt.dev/kio/index: 0
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}

	w := LocalPackageWriter{PackagePath: d}
	err = w.Write([]*yaml.RNode{node2, node1, node3, node4})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "kpt.dev/kio/path cannot be a directory")
	}
}

func getWriterInputs(t *testing.T) (string, *yaml.RNode, *yaml.RNode, *yaml.RNode) {
	node1, err := yaml.Parse(`a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/a_test.yaml"
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	node2, err := yaml.Parse(`c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/path: "a/b/a_test.yaml"
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	node3, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/b_test.yaml"
    kpt.dev/kio/mode: 384
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	d, err := ioutil.TempDir("", "kpt-test")
	if !assert.NoError(t, err) {
		assert.FailNow(t, err.Error())
	}
	if !assert.NoError(t, os.MkdirAll(filepath.Join(d, "a"), 0700)) {
		assert.FailNow(t, "")
	}
	return d, node1, node2, node3
}
