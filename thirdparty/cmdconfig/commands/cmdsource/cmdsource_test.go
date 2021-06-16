// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsource

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestSourceCommand(t *testing.T) {
	d, err := ioutil.TempDir("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: apps/v1
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
	r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1alpha1
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
`, b.String()) {
		return
	}
}

func TestSourceCommand_Unwrap(t *testing.T) {
	d, err := ioutil.TempDir("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: apps/v1
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
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = ioutil.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
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
	r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d, "-o", "unwrap"})
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: apps/v1
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
---
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
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
`, b.String()) {
		return
	}
}

func TestSourceCommand_InvalidInput(t *testing.T) {
	d, err := ioutil.TempDir("", "source-test")
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

	r := GetSourceRunner(fake.CtxWithDefaultPrinter(), "")
	r.Command.SetArgs([]string{d, "-o", "foo/bar"})
	err = r.Command.Execute()
	if !assert.Error(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, `invalid input for --output flag "foo/bar", must be "stdout" or "unwrap"`, err.Error()) {
		t.FailNow()
	}

}

func TestSourceCommand_DefaultDir(t *testing.T) {
	d, err := ioutil.TempDir("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: apps/v1
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
	err = os.Chdir(d)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{})

	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1alpha1
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
`, b.String()) {
		return
	}
}

func TestSourceCommandJSON(t *testing.T) {
	d, err := ioutil.TempDir("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = ioutil.WriteFile(filepath.Join(d, "f1.json"), []byte(`
{
  "apiVersion": "apps/v1",
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
	r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})

	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	//nolint:lll
	expected := `apiVersion: config.kubernetes.io/v1alpha1
kind: ResourceList
items:
  - {"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo", "annotations": {"app": "nginx2", config.kubernetes.io/index: '0', config.kubernetes.io/path: 'f1.json'}}, "spec": {"replicas": 1}}
  - {"apiVersion": "v1", "kind": "Abstraction", "metadata": {"name": "foo", "annotations": {"config.kubernetes.io/function": "container:\n  image: gcr.io/example/reconciler:v1\n", "config.kubernetes.io/local-config": "true", config.kubernetes.io/index: '0', config.kubernetes.io/path: 'f2.json'}}, "spec": {"replicas": 3}}
`

	if !assert.Equal(t, expected, b.String()) {
		return
	}
}

func TestSourceCommandNonKrm(t *testing.T) {
	d, err := ioutil.TempDir("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: v1
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
kind: Custom
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{d})
	err = r.Command.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Equal(t, "f1.yaml: resource must have `metadata.name`", err.Error())
}
