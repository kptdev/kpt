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

package cmdgrep_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"kpt.dev/cmdgrep"

	"github.com/stretchr/testify/assert"
)

// TestCmd_files verifies grep reads the files and filters them
func TestCmd_files(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
kind: Deployment
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
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = ioutil.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := cmdgrep.Cmd()
	r.C.SetArgs([]string{"metadata.name=foo", d})
	r.C.SetOut(b)
	if !assert.NoError(t, r.C.Execute()) {
		return
	}

	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
    kpt.dev/kio/index: 0
    kpt.dev/kio/mode: 384
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
spec:
  replicas: 1
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
    kpt.dev/kio/index: 1
    kpt.dev/kio/mode: 384
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
spec:
  selector:
    app: nginx
`, b.String()) {
		return
	}
}

// TestCmd_stdin verifies the grep command reads stdin if no files are provided
func TestCmd_stdin(t *testing.T) {
	// fmt the files
	b := &bytes.Buffer{}
	r := cmdgrep.Cmd()
	r.C.SetArgs([]string{"metadata.name=foo"})
	r.C.SetOut(b)
	r.C.SetIn(bytes.NewBufferString(`
kind: Deployment
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
---
kind: Deployment
metadata:
  labels:
    app: nginx
  name: bar
  annotations:
    app: nginx
spec:
  replicas: 3
`))
	if !assert.NoError(t, r.C.Execute()) {
		return
	}

	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
    kpt.dev/kio/index: 0
spec:
  replicas: 1
---
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
    kpt.dev/kio/index: 1
spec:
  selector:
    app: nginx
`, b.String()) {
		return
	}
}

// TestCmd_errInputs verifies the grep command errors on invalid matches
func TestCmd_errInputs(t *testing.T) {
	b := &bytes.Buffer{}
	r := cmdgrep.Cmd()
	r.C.SetArgs([]string{"metadata.name=foo=bar"})
	r.C.SetOut(b)
	r.C.SetIn(bytes.NewBufferString(`
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`))
	err := r.C.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "multiple '='")

	// fmt the files
	b = &bytes.Buffer{}
	r = cmdgrep.Cmd()
	r.C.SetArgs([]string{"spec.template.spec.containers[a[b=c].image=foo"})
	r.C.SetOut(b)
	r.C.SetIn(bytes.NewBufferString(`
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`))
	err = r.C.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "unrecognized path element:")
}

// TestCmd_escapeDots verifies the grep command correctly escapes '\.' in inputs
func TestCmd_escapeDots(t *testing.T) {
	// fmt the files
	b := &bytes.Buffer{}
	r := cmdgrep.Cmd()
	r.C.SetArgs([]string{"spec.template.spec.containers[name=nginx].image=nginx:1\\.7\\.9",
		"--annotate=false"})
	r.C.SetOut(b)
	r.C.SetIn(bytes.NewBufferString(`
kind: Deployment
metadata:
  labels:
    app: nginx1.8
  name: nginx1.8
  annotations:
    app: nginx1.8
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.8.1
---
kind: Deployment
metadata:
  labels:
    app: nginx1.7
  name: nginx1.7
  annotations:
    app: nginx1.7
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
`))
	err := r.C.Execute()
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx1.7
  name: nginx1.7
  annotations:
    app: nginx1.7
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
`, b.String()) {
		return
	}
}
