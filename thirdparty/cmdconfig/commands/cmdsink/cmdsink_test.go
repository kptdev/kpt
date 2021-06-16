// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsink

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestSinkCommand(t *testing.T) {
	d, err := ioutil.TempDir("", "sink-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	r := GetSinkRunner(fake.CtxWithDefaultPrinter(), "")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      app: nginx2
    name: foo
    annotations:
      app: nginx2
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'f1.yaml'
  spec:
    replicas: 1
- apiVersion: v1
  kind: Service
  metadata:
    name: foo
    annotations:
      app: nginx
      config.kubernetes.io/index: '1'
      config.kubernetes.io/path: 'f1.yaml'
  spec:
    selector:
      app: nginx
- apiVersion: v1
  kind: Abstraction
  metadata:
    name: foo
    annotations:
      config.kubernetes.io/function: |
        container:
          image: gcr.io/example/reconciler:v1
      config.kubernetes.io/local-config: "true"
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'f2.yaml'
  spec:
    replicas: 3
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      app: nginx
    name: bar
    annotations:
      app: nginx
      config.kubernetes.io/index: '1'
      config.kubernetes.io/path: 'f2.yaml'
  spec:
    replicas: 3
`))
	r.Command.SetArgs([]string{d})
	if !assert.NoError(t, r.Command.Execute()) {
		t.FailNow()
	}

	actual, err := ioutil.ReadFile(filepath.Join(d, "f1.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expected := `apiVersion: apps/v1
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
apiVersion: v1
kind: Service
metadata:
  name: foo
  annotations:
    app: nginx
spec:
  selector:
    app: nginx
`
	if !assert.Equal(t, expected, string(actual)) {
		t.FailNow()
	}

	actual, err = ioutil.ReadFile(filepath.Join(d, "f2.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expected = `apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: gcr.io/example/reconciler:v1
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
`
	if !assert.Equal(t, expected, string(actual)) {
		t.FailNow()
	}
}

func TestSinkCommandJSON(t *testing.T) {
	d, err := ioutil.TempDir("", "sink-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	r := GetSinkRunner(fake.CtxWithDefaultPrinter(), "")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- {"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo",
    "annotations": {"app": "nginx2", config.kubernetes.io/index: '0',
      config.kubernetes.io/path: 'f1.json'}}, "spec": {"replicas": 1}}
`))
	r.Command.SetArgs([]string{d})
	if !assert.NoError(t, r.Command.Execute()) {
		t.FailNow()
	}

	actual, err := ioutil.ReadFile(filepath.Join(d, "f1.json"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expected := `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "annotations": {
      "app": "nginx2"
    },
    "labels": {
      "app": "nginx2"
    },
    "name": "foo"
  },
  "spec": {
    "replicas": 1
  }
}
`
	if !assert.Equal(t, expected, string(actual)) {
		t.FailNow()
	}
}
func TestSinkCommandNonKrm(t *testing.T) {
	d, err := ioutil.TempDir("", "sink-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	r := GetSinkRunner(fake.CtxWithDefaultPrinter(), "")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- kind: Deployment
  metadata:
    labels:
      app: nginx2
    annotations:
      app: nginx2
      config.kubernetes.io/index: '0'
      config.kubernetes.io/path: 'f1.yaml'
  spec:
    replicas: 1
`))
	r.Command.SetArgs([]string{d})
	err = r.Command.Execute()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "f1.yaml: resource must have `apiVersion`", err.Error())
}
