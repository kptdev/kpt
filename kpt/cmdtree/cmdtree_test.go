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

package cmdtree_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"kpt.dev/kpt/cmdtree"
)

// TestCmd_files verifies fmt reads the files and filters them
func TestCmd_files(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}

	err = ioutil.WriteFile(filepath.Join(d, "f1.yaml"), []byte(`
apiVersion: gcr.io/example/reconciler:v1
kind: Abstraction
metadata:
  name: foo
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
	r := cmdtree.Cmd()
	r.C.SetArgs([]string{d})
	r.C.SetOut(b)
	if !assert.NoError(t, r.C.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
└── [f2.yaml]  Deployment bar
`, d), b.String()) {
		return
	}
}

func TestCmd_stdin(t *testing.T) {
	// fmt the files
	b := &bytes.Buffer{}
	r := cmdtree.Cmd()
	r.C.SetArgs([]string{})
	r.C.SetIn(bytes.NewBufferString(`apiVersion: extensions/v1
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo3
  namespace: default
  annotations:
    app: nginx2
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
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
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
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
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
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
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
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
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx
  annotations:
    app: nginx
    kpt.dev/kio/package: bar-package
    kpt.dev/kio/path: f2.yaml
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
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f1.yaml
spec:
  selector:
    app: nginx
`))
	r.C.SetOut(b)
	if !assert.NoError(t, r.C.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`.
├── [f1.yaml]  Deployment default/foo
├── [f1.yaml]  Service default/foo
├── [f1.yaml]  Deployment default/foo3
├── [f1.yaml]  Deployment default/foo3
├── [f1.yaml]  Deployment default/foo3
├── [f1.yaml]  Deployment default2/foo2
└── bar-package
    └── [f2.yaml]  Deployment bar
`), b.String()) {
		return
	}
}

func TestCmd_includeReconcilers(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}

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
apiVersion: gcr.io/example/reconciler:v1
kind: Abstraction
metadata:
  name: foo
spec:
  replicas: 1
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
	r := cmdtree.Cmd()
	r.C.SetArgs([]string{d, "--include-reconcilers"})
	r.C.SetOut(b)
	if !assert.NoError(t, r.C.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
├── [f1.yaml]  Deployment foo
├── [f1.yaml]  Service foo
├── [f2.yaml]  Deployment bar
└── [f2.yaml]  Abstraction foo
`, d), b.String()) {
		return
	}
}

func TestCmd_excludeNonReconcilers(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt-test")
	defer os.RemoveAll(d)
	if !assert.NoError(t, err) {
		return
	}

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
apiVersion: gcr.io/example/reconciler:v1
kind: Abstraction
metadata:
  name: foo
spec:
  replicas: 1
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
	r := cmdtree.Cmd()
	r.C.SetArgs([]string{d, "--include-reconcilers", "--exclude-non-reconcilers"})
	r.C.SetOut(b)
	if !assert.NoError(t, r.C.Execute()) {
		return
	}

	if !assert.Equal(t, fmt.Sprintf(`%s
└── [f2.yaml]  Abstraction foo
`, d), b.String()) {
		return
	}
}
