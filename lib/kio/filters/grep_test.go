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

package filters_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/kio"
	. "lib.kpt.dev/kio/filters"
	"lib.kpt.dev/yaml"
)

func TestGrepFilter_Filter(t *testing.T) {
	in := `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx
  annotations:
    app: nginx
  name: bar
spec:
  replicas: 3
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
spec:
  selector:
    app: nginx
`
	out := &bytes.Buffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBufferString(in)}},
		Filters: []kio.Filter{GrepFilter{Path: []string{"metadata", "name"}, Value: "foo"}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
spec:
  selector:
    app: nginx
`, out.String()) {
		t.FailNow()
	}

	out = &bytes.Buffer{}
	err = kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBufferString(in)}},
		Filters: []kio.Filter{GrepFilter{Path: []string{"kind"}, Value: "Deployment"}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx
  annotations:
    app: nginx
  name: bar
spec:
  replicas: 3
`, out.String()) {
		t.FailNow()
	}

	out = &bytes.Buffer{}
	err = kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBufferString(in)}},
		Filters: []kio.Filter{GrepFilter{Path: []string{"spec", "replicas"}, Value: "3"}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx
  annotations:
    app: nginx
  name: bar
spec:
  replicas: 3
`, out.String()) {
		t.FailNow()
	}

	out = &bytes.Buffer{}
	err = kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBufferString(in)}},
		Filters: []kio.Filter{GrepFilter{Path: []string{"spec", "not-present"}, Value: "3"}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, ``, out.String()) {
		t.FailNow()
	}
}

func TestGrepFilter_init(t *testing.T) {
	assert.Equal(t, GrepFilter{}, Filters["GrepFilter"]())
}

func TestGrepFilter_error(t *testing.T) {
	_, err := GrepFilter{Path: []string{"metadata", "name"},
		Value: "foo"}.Filter([]*yaml.RNode{{}})
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "missing value")
}
