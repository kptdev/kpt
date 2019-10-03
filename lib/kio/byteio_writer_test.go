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

package kio

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/yaml"
)

// TestByteWriter_Write_withoutAnnotations tests:
// - Resource Config ordering is preserved if no annotations are present
func TestByteWriter_Write_wrapped(t *testing.T) {
	node1, err := yaml.Parse(`a: b #first
`)
	if !assert.NoError(t, err) {
		return
	}
	node2, err := yaml.Parse(`c: d # second
`)
	if !assert.NoError(t, err) {
		return
	}
	node3, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
`)
	if !assert.NoError(t, err) {
		return
	}

	buff := &bytes.Buffer{}
	err = ByteWriter{
		Writer:             buff,
		FunctionConfig:     node3,
		WrappingKind:       InputOutputListKind,
		WrappingApiVersion: InputOutputListApiVersion}.
		Write([]*yaml.RNode{node2, node1})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: kpt.dev/v1alpha1
kind: InputOutputList
items:
- c: d # second
- a: b #first
functionConfig:
  e: f
  g:
    h:
    - i # has a list
    - j
`, buff.String())
}

// TestByteWriter_Write_withoutAnnotations tests:
// - Resource Config ordering is preserved if no annotations are present
func TestByteWriter_Write_withoutAnnotations(t *testing.T) {
	node1, err := yaml.Parse(`a: b #first
`)
	if !assert.NoError(t, err) {
		return
	}
	node2, err := yaml.Parse(`c: d # second
`)
	if !assert.NoError(t, err) {
		return
	}
	node3, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
`)
	if !assert.NoError(t, err) {
		return
	}

	buff := &bytes.Buffer{}
	err = ByteWriter{Writer: buff}.
		Write([]*yaml.RNode{node2, node3, node1})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `c: d # second
---
e: f
g:
  h:
  - i # has a list
  - j
---
a: b #first
`, buff.String())
}

// TestByteWriter_Write_withAnnotationsKeepAnnotations tests:
// - Resource Config is sorted by annotations if present
// - IndexAnnotations are retained
func TestByteWriter_Write_withAnnotationsKeepAnnotations(t *testing.T) {
	node1, err := yaml.Parse(`a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/a_test.yaml"
`)
	if !assert.NoError(t, err) {
		return
	}
	node2, err := yaml.Parse(`c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/path: "a/b/a_test.yaml"
`)
	if !assert.NoError(t, err) {
		return
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
`)
	if !assert.NoError(t, err) {
		return
	}

	buff := &bytes.Buffer{}
	err = ByteWriter{Writer: buff, KeepReaderAnnotations: true}.
		Write([]*yaml.RNode{node2, node3, node1})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/a_test.yaml"
---
c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/path: "a/b/a_test.yaml"
---
e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/b_test.yaml"
`, buff.String())
}

// TestByteWriter_Write_withAnnotations tests:
// - Resource Config is sorted by annotations if present
// - IndexAnnotations are pruned
func TestByteWriter_Write_withAnnotations(t *testing.T) {
	node1, err := yaml.Parse(`a: b #first
metadata:
  annotations:
    kpt.dev/kio/index: 0
    kpt.dev/kio/path: "a/b/a_test.yaml"
`)
	if !assert.NoError(t, err) {
		return
	}
	node2, err := yaml.Parse(`c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/path: "a/b/a_test.yaml"
`)
	if !assert.NoError(t, err) {
		return
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
`)
	if !assert.NoError(t, err) {
		return
	}

	buff := &bytes.Buffer{}
	err = ByteWriter{Writer: buff}.
		Write([]*yaml.RNode{node2, node3, node1})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `a: b #first
metadata:
  annotations:
    kpt.dev/kio/path: "a/b/a_test.yaml"
---
c: d # second
metadata:
  annotations:
    kpt.dev/kio/path: "a/b/a_test.yaml"
---
e: f
g:
  h:
  - i # has a list
  - j
metadata:
  annotations:
    kpt.dev/kio/path: "a/b/b_test.yaml"
`, buff.String())
}

// TestByteWriter_Write_partialValues tests:
// - Resource Config is sorted when annotations are present on some but not all ResourceNodes
func TestByteWriter_Write_partialAnnotations(t *testing.T) {
	node1, err := yaml.Parse(`a: b #first
metadata:
  annotations:
    kpt.dev/kio/path: "a/b/a_test.yaml"
`)
	if !assert.NoError(t, err) {
		return
	}
	node2, err := yaml.Parse(`c: d # second
metadata:
  annotations:
    kpt.dev/kio/index: 1
    kpt.dev/kio/path: "a/b/a_test.yaml"
`)
	if !assert.NoError(t, err) {
		return
	}
	node3, err := yaml.Parse(`e: f
g:
  h:
  - i # has a list
  - j
`)
	if !assert.NoError(t, err) {
		return
	}

	buff := &bytes.Buffer{}
	rw := ByteWriter{Writer: buff}
	err = rw.Write([]*yaml.RNode{node2, node3, node1})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `e: f
g:
  h:
  - i # has a list
  - j
---
a: b #first
metadata:
  annotations:
    kpt.dev/kio/path: "a/b/a_test.yaml"
---
c: d # second
metadata:
  annotations:
    kpt.dev/kio/path: "a/b/a_test.yaml"
`, buff.String())
}
