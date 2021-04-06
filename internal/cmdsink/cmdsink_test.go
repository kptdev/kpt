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

package cmdsink

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSinkCommand(t *testing.T) {
	d, err := ioutil.TempDir("", "kustomize-source-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	r := GetSinkRunner("")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1alpha1
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

	actual, err := ioutil.ReadFile(filepath.Join(d, "f1.yaml"))
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
	d, err := ioutil.TempDir("", "kustomize-source-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	r := GetSinkRunner("")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1alpha1
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

	actual, err := ioutil.ReadFile(filepath.Join(d, "f1.json"))
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

func TestSinkCommand_Stdout(t *testing.T) {
	d, err := ioutil.TempDir("", "kustomize-source-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	// fmt the files
	out := &bytes.Buffer{}
	r := GetSinkRunner("")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1alpha1
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

	r.Command.SetOut(out)
	r.Command.SetArgs([]string{})
	if !assert.NoError(t, r.Command.Execute()) {
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
---
apiVersion: v1
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
	if !assert.Equal(t, expected, out.String()) {
		t.FailNow()
	}
}

func TestSinkCommandJSON_Stdout(t *testing.T) {
	d, err := ioutil.TempDir("", "kustomize-source-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	// fmt the files
	out := &bytes.Buffer{}
	r := GetSinkRunner("")
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- {"kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo",
    "annotations": {"app": "nginx2", config.kubernetes.io/index: '0',
      config.kubernetes.io/path: 'f1.json'}}, "spec": {"replicas": 1}}
`))

	r.Command.SetOut(out)
	r.Command.SetArgs([]string{})
	if !assert.NoError(t, r.Command.Execute()) {
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
	if !assert.Equal(t, expected, out.String()) {
		println(out.String())
		t.FailNow()
	}
}
