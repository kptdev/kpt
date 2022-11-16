// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsink

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestSinkCommand(t *testing.T) {
	d, err := os.MkdirTemp("", "sink-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// delete the dir as we just want the temp dir path
	// directory should be created by the command
	os.RemoveAll(d)

	r := GetSinkRunner(fake.CtxWithDefaultPrinter(), "")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- kind: Deployment
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
- kind: Service
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
	defer os.RemoveAll(d)

	actual, err := os.ReadFile(filepath.Join(d, "f1.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expected := `kind: Deployment
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
`
	if !assert.Equal(t, expected, string(actual)) {
		t.FailNow()
	}

	actual, err = os.ReadFile(filepath.Join(d, "f2.yaml"))
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

func TestSinkCommand_Error(t *testing.T) {
	d, err := os.MkdirTemp("", "sink-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	r := GetSinkRunner(fake.CtxWithDefaultPrinter(), "")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- kind: Deployment
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
`))
	r.Command.SetArgs([]string{d})
	logs := &bytes.Buffer{}
	r.Ctx = printer.WithContext(r.Ctx, printer.New(nil, logs))
	err = r.Command.Execute()
	if !assert.Error(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, fmt.Sprintf(`directory %q already exists, please delete the directory and retry`, d), err.Error()) {
		t.FailNow()
	}
}

func TestSinkCommandJSON(t *testing.T) {
	d, err := os.MkdirTemp("", "sink-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// delete the dir as we just want the temp dir path
	// directory should be created by the command
	os.RemoveAll(d)

	r := GetSinkRunner(fake.CtxWithDefaultPrinter(), "")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- {"kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo",
    "annotations": {"app": "nginx2", config.kubernetes.io/index: '0',
      config.kubernetes.io/path: 'f1.json'}}, "spec": {"replicas": 1}}
`))
	r.Command.SetArgs([]string{d})
	if !assert.NoError(t, r.Command.Execute()) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	actual, err := os.ReadFile(filepath.Join(d, "f1.json"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	expected := `{
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
