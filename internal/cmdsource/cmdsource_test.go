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

package cmdsource

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceCommand(t *testing.T) {
	d, err := ioutil.TempDir("", "kustomize-source-test")
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
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetSourceRunner("")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1alpha1
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
`, b.String()) {
		return
	}
}

func TestSourceCommandJSON(t *testing.T) {
	d, err := ioutil.TempDir("", "kustomize-source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = ioutil.WriteFile(filepath.Join(d, "f1.json"), []byte(`
{
  "kind": "Deployment",
  "metadata": {
    "labels": {
      "app": "nginx2"
    },
    "name": "foo",
    "annotations": {
      "app": "nginx2"
    }
  },
  "spec": {
    "replicas": 1
  }
}
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = ioutil.WriteFile(filepath.Join(d, "f2.json"), []byte(`
{
  "apiVersion": "v1",
  "kind": "Abstraction",
  "metadata": {
    "name": "foo",
    "annotations": {
      "config.kubernetes.io/function": "container:\n  image: gcr.io/example/reconciler:v1\n",
      "config.kubernetes.io/local-config": "true"
    }
  },
  "spec": {
    "replicas": 3
  }
}
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetSourceRunner("")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- {"kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo",
    "annotations": {"app": "nginx2", config.kubernetes.io/index: '0', config.kubernetes.io/path: 'f1.json'}},
  "spec": {"replicas": 1}}
- {"apiVersion": "v1", "kind": "Abstraction", "metadata": {"name": "foo", "annotations": {
      "config.kubernetes.io/function": "container:\n  image: gcr.io/example/reconciler:v1\n",
      "config.kubernetes.io/local-config": "true", config.kubernetes.io/index: '0',
      config.kubernetes.io/path: 'f2.json'}}, "spec": {"replicas": 3}}
`, b.String()) {
		return
	}
}

func TestSourceCommand_Stdin(t *testing.T) {
	d, err := ioutil.TempDir("", "kustomize-source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	in := bytes.NewBufferString(`
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
`)

	out := &bytes.Buffer{}
	r := GetSourceRunner("")
	r.Command.SetArgs([]string{})
	r.Command.SetIn(in)
	r.Command.SetOut(out)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1alpha1
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
  spec:
    replicas: 1
- kind: Service
  metadata:
    name: foo
    annotations:
      app: nginx
      config.kubernetes.io/index: '1'
  spec:
    selector:
      app: nginx
`, out.String()) {
		return
	}
}

func TestSourceCommandJSON_Stdin(t *testing.T) {
	d, err := ioutil.TempDir("", "kustomize-source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	in := bytes.NewBufferString(`
{
  "kind": "Deployment",
  "metadata": {
    "labels": {
      "app": "nginx2"
    },
    "name": "foo",
    "annotations": {
      "app": "nginx2"
    }
  },
  "spec": {
    "replicas": 1
  }
}
`)

	out := &bytes.Buffer{}
	r := GetSourceRunner("")
	r.Command.SetArgs([]string{})
	r.Command.SetIn(in)
	r.Command.SetOut(out)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
- {"kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo",
    "annotations": {"app": "nginx2", config.kubernetes.io/index: '0'}}, "spec": {
    "replicas": 1}}
`, out.String()) {
		return
	}
}
