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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/kio"
)

// getByteReaderTestInput returns test input
func getByteReaderTestInput(t *testing.T) *bytes.Buffer {
	b := &bytes.Buffer{}
	_, err := b.WriteString(`
---
a: b # first resource
c: d
---
# second resource
e: f
g:
- h
---
---
i: j
`)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "")
	}
	return b
}

// TestByteReader_Read tests the default Read behavior
// - Resources are read into a slice
// - ReaderAnnotations are set on the ResourceNodes
func TestByteReader_Read(t *testing.T) {
	nodes, err := ByteReader{Reader: getByteReaderTestInput(t)}.Read()
	if !assert.NoError(t, err) {
		return
	}

	if !assert.Len(t, nodes, 3) {
		return
	}
	expected := []string{
		`a: b # first resource
c: d
metadata:
  annotations:
    kpt.dev/kio/index: 0
`,
		`# second resource
e: f
g:
- h
metadata:
  annotations:
    kpt.dev/kio/index: 1
`,
		`i: j
metadata:
  annotations:
    kpt.dev/kio/index: 2
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

// TestByteReader_Read_omitReaderAnnotations tests
// - Resources are read into a slice
// - ReaderAnnotations are not set on the ResourceNodes
func TestByteReader_Read_omitReaderAnnotations(t *testing.T) {
	nodes, err := ByteReader{Reader: getByteReaderTestInput(t), OmitReaderAnnotations: true}.Read()
	if !assert.NoError(t, err) {
		return
	}

	// should have parsed 3 resources
	if !assert.Len(t, nodes, 3) {
		return
	}
	expected := []string{
		"a: b # first resource\nc: d\n",
		"# second resource\ne: f\ng:\n- h\n",
		"i: j\n",
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

// TestByteReader_Read_omitReaderAnnotations tests
// - Resources are read into a slice
// - ReaderAnnotations are NOT set on the ResourceNodes
// - Additional annotations ARE set on the ResourceNodes
func TestByteReader_Read_setAnnotationsOmitReaderAnnotations(t *testing.T) {
	nodes, err := ByteReader{
		Reader:                getByteReaderTestInput(t),
		SetAnnotations:        map[string]string{"foo": "bar"},
		OmitReaderAnnotations: true,
	}.Read()
	if !assert.NoError(t, err) {
		return
	}

	if !assert.Len(t, nodes, 3) {
		return
	}
	expected := []string{
		`a: b # first resource
c: d
metadata:
  annotations:
    foo: bar
`,
		`# second resource
e: f
g:
- h
metadata:
  annotations:
    foo: bar
`,
		`i: j
metadata:
  annotations:
    foo: bar
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

// TestByteReader_Read_omitReaderAnnotations tests
// - Resources are read into a slice
// - ReaderAnnotations ARE set on the ResourceNodes
// - Additional annotations ARE set on the ResourceNodes
func TestByteReader_Read_setAnnotations(t *testing.T) {
	nodes, err := ByteReader{
		Reader:         getByteReaderTestInput(t),
		SetAnnotations: map[string]string{"foo": "bar"},
	}.Read()
	if !assert.NoError(t, err) {
		return
	}

	if !assert.Len(t, nodes, 3) {
		return
	}
	expected := []string{
		`a: b # first resource
c: d
metadata:
  annotations:
    foo: bar
    kpt.dev/kio/index: 0
`,
		`# second resource
e: f
g:
- h
metadata:
  annotations:
    foo: bar
    kpt.dev/kio/index: 1
`,
		`i: j
metadata:
  annotations:
    foo: bar
    kpt.dev/kio/index: 2
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
