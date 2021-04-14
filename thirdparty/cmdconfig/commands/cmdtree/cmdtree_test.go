// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdtree

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTreeCommandDefaultCurDir_files(t *testing.T) {
	d, err := ioutil.TempDir("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}
	err = os.Chdir(d)
	if !assert.NoError(t, err) {
		return
	}

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	r := GetTreeRunner("")
	r.Command.SetArgs([]string{})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `.
├── [f1.yaml]  Abstraction foo
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── [f2.yaml]  Deployment bar
`, b.String()) {
		return
	}
}

func TestTreeCommand_files(t *testing.T) {
	d, err := ioutil.TempDir("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	r := GetTreeRunner("")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
├── [f1.yaml]  Abstraction foo
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── [f2.yaml]  Deployment bar
`, d), b.String()) {
		return
	}
}

func TestTreeCommand_Kustomization(t *testing.T) {
	d, err := ioutil.TempDir("", "tree-test")
	defer os.RemoveAll(d)
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

	err = ioutil.WriteFile(filepath.Join(d, "Kustomization"), []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- f2.yaml
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner("")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
└── [f2.yaml]  Deployment bar
`, d), b.String()) {
		return
	}
}

func TestTreeCommand_subpkgs(t *testing.T) {
	d, err := ioutil.TempDir("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = os.MkdirAll(filepath.Join(d, "subpkg"), 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
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
	err = ioutil.WriteFile(filepath.Join(d, "subpkg", "f2.yaml"), []byte(`kind: Deployment
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

	err = ioutil.WriteFile(filepath.Join(d, "Kptfile"), []byte(`apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: mainpkg
openAPI:
  definitions:
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = ioutil.WriteFile(filepath.Join(d, "subpkg", "Kptfile"), []byte(`apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: subpkg
openAPI:
  definitions:
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner("")
	r.Command.SetArgs([]string{d})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`PKG: %s
├── [Kptfile]  Kptfile mainpkg
├── [f1.yaml]  Abstraction foo
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── PKG: subpkg
    ├── [Kptfile]  Kptfile subpkg
    └── [f2.yaml]  Deployment bar
`, d), b.String()) {
		return
	}
}

func TestTreeCommand_CurDirInput(t *testing.T) {
	d, err := ioutil.TempDir("", "tree-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = os.MkdirAll(filepath.Join(d, "Mainpkg", "Subpkg"), 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = os.Chdir(filepath.Join(d, "Mainpkg"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = ioutil.WriteFile(filepath.Join(d, "Mainpkg", "f1.yaml"), []byte(`
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
	err = ioutil.WriteFile(filepath.Join(d, "Mainpkg", "Subpkg", "f2.yaml"), []byte(`kind: Deployment
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

	err = ioutil.WriteFile(filepath.Join(d, "Mainpkg", "Kptfile"), []byte(`apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: Mainpkg
`), 0600)
	if !assert.NoError(t, err) {
		return
	}
	err = ioutil.WriteFile(filepath.Join(d, "Mainpkg", "Subpkg", "Kptfile"), []byte(`apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: Subpkg
`), 0600)
	if !assert.NoError(t, err) {
		return
	}

	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner("")
	r.Command.SetArgs([]string{})
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `PKG: Mainpkg
├── [Kptfile]  Kptfile Mainpkg
├── [f1.yaml]  Deployment foo
└── PKG: Subpkg
    ├── [Kptfile]  Kptfile Subpkg
    └── [f2.yaml]  Deployment bar
`, b.String()) {
		return
	}
}

func TestTreeCommand_stdin(t *testing.T) {
	// fmt the files
	b := &bytes.Buffer{}
	r := GetTreeRunner("")
	r.Command.SetArgs([]string{"-"})
	r.Command.SetIn(bytes.NewBufferString(`apiVersion: extensions/v1
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo3
  namespace: default
  annotations:
    app: nginx2
    config.kubernetes.io/path: f1.yaml
spec:
  replicas: 1
---
apiVersion: extensions/v1
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo3
  namespace: default
  annotations:
    app: nginx2
    config.kubernetes.io/path: f1.yaml
spec:
  replicas: 1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo3
  namespace: default
  annotations:
    app: nginx2
    config.kubernetes.io/path: f1.yaml
spec:
  replicas: 1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo2
  namespace: default2
  annotations:
    app: nginx2
    config.kubernetes.io/path: f1.yaml
spec:
  replicas: 1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx3
  name: foo
  namespace: default
  annotations:
    app: nginx3
    config.kubernetes.io/path: f1.yaml
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx
  annotations:
    app: nginx
    config.kubernetes.io/path: bar-package/f2.yaml
  name: bar
spec:
  replicas: 3
---
kind: Service
metadata:
  name: foo
  namespace: default
  annotations:
    app: nginx
    config.kubernetes.io/path: f1.yaml
spec:
  selector:
    app: nginx
`))
	r.Command.SetOut(b)
	if !assert.NoError(t, r.Command.Execute()) {
		return
	}

	if !assert.Equal(t, `.
├── [f1.yaml]  Deployment default/foo
├── [f1.yaml]  Service default/foo
├── [f1.yaml]  Deployment default/foo3
├── [f1.yaml]  Deployment default/foo3
├── [f1.yaml]  Deployment default/foo3
├── [f1.yaml]  Deployment default2/foo2
└── bar-package
    └── [f2.yaml]  Deployment bar
`, b.String()) {
		return
	}
}
