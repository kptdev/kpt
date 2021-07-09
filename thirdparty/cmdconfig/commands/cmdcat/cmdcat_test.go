// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdcat

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd_DIR(t *testing.T) {
	d, err := ioutil.TempDir("", "cat-test")
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
	err = ioutil.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  configFn:
    container:
      image: gcr.io/example/reconciler:v1
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
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
	r := GetCatRunner("")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
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
---
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  annotations:
    config.kubernetes.io/local-config: "true"
  configFn:
    container:
      image: gcr.io/example/reconciler:v1
spec:
  replicas: 3
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bar
  labels:
    app: nginx
  annotations:
    app: nginx
spec:
  replicas: 3
`, b.String()) {
		return
	}
}

func TestCmd_File(t *testing.T) {
	d, err := ioutil.TempDir("", "cat-test")
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
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetCatRunner("")
	r.Command.SetArgs([]string{filepath.Join(d, "f1.yaml")})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
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
`, b.String()) {
		return
	}
}

func TestCmd_Annotate(t *testing.T) {
	d, err := ioutil.TempDir("", "cat-test")
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
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetCatRunner("")
	r.Command.SetArgs([]string{filepath.Join(d, "f1.yaml"), "--annotate"})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
    config.kubernetes.io/index: '0'
    config.kubernetes.io/path: 'f1.yaml'
    internal.config.kubernetes.io/seqindent: 'compact'
spec:
  replicas: 1
`, b.String()) {
		return
	}
}

func TestCmd_Subpkgs(t *testing.T) {
	d, err := ioutil.TempDir("", "cat-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)
	err = os.MkdirAll(filepath.Join(d, "subpkg"), 0700)
	if !assert.NoError(t, err) {
		return
	}

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
kind: Deployment
metadata:
  labels:
    app: nginx1
  name: foo
  annotations:
    app: nginx1
spec:
  replicas: 1
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	err = ioutil.WriteFile(filepath.Join(d, "subpkg", "f2.yaml"), []byte(`
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	b := &bytes.Buffer{}
	r := GetCatRunner("")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `kind: Deployment
metadata:
  labels:
    app: nginx1
  name: foo
  annotations:
    app: nginx1
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  annotations:
    app: nginx2
spec:
  replicas: 1
`, b.String()) {
		return
	}
}
