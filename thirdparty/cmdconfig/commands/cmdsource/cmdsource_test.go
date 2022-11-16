// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsource

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestSourceCommand(t *testing.T) {
	d, err := os.MkdirTemp("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`
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

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 1
- kind: Service
  metadata:
    name: foo
    annotations:
      app: nginx
      config.kubernetes.io/index: '1'
      config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/index: '1'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f2.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
      internal.config.kubernetes.io/index: '1'
      internal.config.kubernetes.io/path: 'f2.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 3
`, b.String()) {
		return
	}
}

func TestSourceCommand_Unwrap(t *testing.T) {
	d, err := os.MkdirTemp("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`
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
	d, err := os.MkdirTemp("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	d, err := os.MkdirTemp("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`
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
	revert := testutil.Chdir(t, d)
	defer revert()

	// fmt the files
	b := &bytes.Buffer{}
	r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
	r.Command.SetArgs([]string{})

	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 1
- kind: Service
  metadata:
    name: foo
    annotations:
      app: nginx
      config.kubernetes.io/index: '1'
      config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/index: '1'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f2.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
      internal.config.kubernetes.io/index: '1'
      internal.config.kubernetes.io/path: 'f2.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 3
`, b.String()) {
		return
	}
}

func TestSourceCommandJSON(t *testing.T) {
	d, err := os.MkdirTemp("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = os.WriteFile(filepath.Join(d, "f1.json"), []byte(`
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
	err = os.WriteFile(filepath.Join(d, "f2.json"), []byte(`
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
	expected := `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- {"kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo", "annotations": {"app": "nginx2", config.kubernetes.io/index: '0', config.kubernetes.io/path: 'f1.json', internal.config.kubernetes.io/index: '0', internal.config.kubernetes.io/path: 'f1.json', internal.config.kubernetes.io/seqindent: 'compact'}}, "spec": {"replicas": 1}}
- {"apiVersion": "v1", "kind": "Abstraction", "metadata": {"name": "foo", "annotations": {"config.kubernetes.io/function": "container:\n  image: gcr.io/example/reconciler:v1\n", "config.kubernetes.io/local-config": "true", config.kubernetes.io/index: '0', config.kubernetes.io/path: 'f2.json', internal.config.kubernetes.io/index: '0', internal.config.kubernetes.io/path: 'f2.json', internal.config.kubernetes.io/seqindent: 'compact'}}, "spec": {"replicas": 3}}
`

	if !assert.Equal(t, expected, b.String()) {
		return
	}
}

func TestSourceCommand_Symlink(t *testing.T) {
	d, err := os.MkdirTemp("", "source-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}

	defer testutil.Chdir(t, d)()
	err = os.MkdirAll(filepath.Join(d, "foo"), 0700)
	assert.NoError(t, err)
	err = os.Symlink("foo", "foo-link")
	if !assert.NoError(t, err) {
		return
	}
	err = os.WriteFile(filepath.Join(d, "foo", "f1.yaml"), []byte(`
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
	err = os.WriteFile(filepath.Join(d, "foo", "f2.yaml"), []byte(`
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
	stderr := &bytes.Buffer{}
	r := GetSourceRunner(fake.CtxWithPrinter(b, stderr), "")
	r.Command.SetArgs([]string{"foo-link"})
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `apiVersion: config.kubernetes.io/v1
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 1
- kind: Service
  metadata:
    name: foo
    annotations:
      app: nginx
      config.kubernetes.io/index: '1'
      config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/index: '1'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f2.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
      internal.config.kubernetes.io/index: '1'
      internal.config.kubernetes.io/path: 'f2.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
  spec:
    replicas: 3
`, b.String()) {
		return
	}
	assert.Contains(t, stderr.String(), "please note that the symlinks within the package are ignored")
}
