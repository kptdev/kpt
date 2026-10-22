// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsource

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestSourceCommand(t *testing.T) {
	d, err := os.MkdirTemp("", "source-test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)

	err = os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: ghcr.io/example/reconciler:v1
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
          image: ghcr.io/example/reconciler:v1
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
	err = os.WriteFile(filepath.Join(d, "f2.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: ghcr.io/example/reconciler:v1
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
          image: ghcr.io/example/reconciler:v1
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
	err = os.WriteFile(filepath.Join(d, "f2.json"), []byte(`
{
  "apiVersion": "v1",
  "kind": "Abstraction",
  "metadata": {
    "name": "foo",
    "annotations": {
      "config.kubernetes.io/function": "container:\n  image: ghcr.io/example/reconciler:v1\n",
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
- {"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {"labels": {"app": "nginx2"}, "name": "foo", "annotations": {"app": "nginx2", config.kubernetes.io/index: '0', config.kubernetes.io/path: 'f1.json', internal.config.kubernetes.io/index: '0', internal.config.kubernetes.io/path: 'f1.json', internal.config.kubernetes.io/seqindent: 'compact'}}, "spec": {"replicas": 1}}
- {"apiVersion": "v1", "kind": "Abstraction", "metadata": {"name": "foo", "annotations": {"config.kubernetes.io/function": "container:\n  image: ghcr.io/example/reconciler:v1\n", "config.kubernetes.io/local-config": "true", config.kubernetes.io/index: '0', config.kubernetes.io/path: 'f2.json', internal.config.kubernetes.io/index: '0', internal.config.kubernetes.io/path: 'f2.json', internal.config.kubernetes.io/seqindent: 'compact'}}, "spec": {"replicas": 3}}
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
	err = os.WriteFile(filepath.Join(d, "foo", "f2.yaml"), []byte(`
apiVersion: v1
kind: Abstraction
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: ghcr.io/example/reconciler:v1
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
      internal.config.kubernetes.io/index: '0'
      internal.config.kubernetes.io/path: 'f1.yaml'
      internal.config.kubernetes.io/seqindent: 'compact'
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
          image: ghcr.io/example/reconciler:v1
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

// TestSourceCommand_NonKRM covers the non-KRM pre-scan in SourceRunner.runE.
func TestSourceCommand_NonKRM(t *testing.T) {
	t.Run("non-KRM file with no annotation blocks source with a consolidated, file-naming error", func(t *testing.T) {
		d := t.TempDir()
		err := os.WriteFile(filepath.Join(d, "deployment.yaml"), []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 1
`), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "values.yaml"), []byte("replicaCount: 3\n"), 0600)
		assert.NoError(t, err)

		b := &bytes.Buffer{}
		r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
		r.Command.SetArgs([]string{d})
		err = r.Command.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "values.yaml")
		assert.Contains(t, err.Error(), "1 file(s)")
	})

	t.Run("annotated package skips the non-KRM file in memory, leaves it untouched on disk", func(t *testing.T) {
		d := t.TempDir()
		err := os.WriteFile(filepath.Join(d, "Kptfile"), []byte(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pkg
  annotations:
    kpt.dev/auto-ignore-non-krm: "true"
`), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "deployment.yaml"), []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 1
`), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "values.yaml"), []byte("replicaCount: 3\n"), 0600)
		assert.NoError(t, err)

		b := &bytes.Buffer{}
		r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
		r.Command.SetArgs([]string{d})
		err = r.Command.Execute()
		assert.NoError(t, err)

		assert.NotContains(t, b.String(), "replicaCount", "non-KRM file must not appear in the ResourceList output")
		assert.Contains(t, b.String(), "kind: Deployment")

		content, readErr := os.ReadFile(filepath.Join(d, "values.yaml"))
		assert.NoError(t, readErr)
		assert.Equal(t, "replicaCount: 3\n", string(content), "non-KRM file must be left completely untouched on disk")
		assert.NoFileExists(t, filepath.Join(d, ".krmignore"), ".krmignore must never be written by the annotation path")
	})

	t.Run(".krmignore still works independently of the annotation, no non-KRM-file error at all", func(t *testing.T) {
		d := t.TempDir()
		err := os.WriteFile(filepath.Join(d, "deployment.yaml"), []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
spec:
  replicas: 1
`), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, "values.yaml"), []byte("replicaCount: 3\n"), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(d, ".krmignore"), []byte("values.yaml\n"), 0600)
		assert.NoError(t, err)

		b := &bytes.Buffer{}
		r := GetSourceRunner(fake.CtxWithPrinter(b, nil), "")
		r.Command.SetArgs([]string{d})
		err = r.Command.Execute()
		assert.NoError(t, err)

		assert.NotContains(t, b.String(), "replicaCount")
		assert.Contains(t, b.String(), "kind: Deployment")
	})
}
