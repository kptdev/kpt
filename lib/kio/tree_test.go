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
package kio_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

func TestPrinter_Write(t *testing.T) {
	in := `kind: Deployment
metadata:
  labels:
    app: nginx3
  name: foo
  namespace: default
  annotations:
    app: nginx3
    kpt.dev/kio/package: foo-package/3
    kpt.dev/kio/path: f3.yaml
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  namespace: default
  annotations:
    app: nginx2
    kpt.dev/kio/package: foo-package
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
    kpt.dev/kio/package: foo-package
    kpt.dev/kio/path: f1.yaml
spec:
  selector:
    app: nginx
`
	out := &bytes.Buffer{}
	err := Pipeline{
		Inputs:  []Reader{ByteReader{Reader: bytes.NewBufferString(in)}},
		Outputs: []Writer{TreeWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, `
├── bar-package
│   └── [f2.yaml]  .Deployment bar
└── foo-package
    ├── [f1.yaml]  .Deployment default/foo
    ├── [f1.yaml]  .Service default/foo
    └── foo-package/3
        └── [f3.yaml]  .Deployment default/foo
`, out.String()) {
		t.FailNow()
	}
}

func TestPrinter_Write_base(t *testing.T) {
	in := `kind: Deployment
metadata:
  labels:
    app: nginx3
  name: foo
  namespace: default
  annotations:
    app: nginx3
    kpt.dev/kio/package: .
    kpt.dev/kio/path: f3.yaml
spec:
  replicas: 1
---
kind: Deployment
metadata:
  labels:
    app: nginx2
  name: foo
  namespace: default
  annotations:
    app: nginx2
    kpt.dev/kio/package: foo-package
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
`
	out := &bytes.Buffer{}
	err := Pipeline{
		Inputs:  []Reader{ByteReader{Reader: bytes.NewBufferString(in)}},
		Outputs: []Writer{TreeWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, `
├── [f1.yaml]  .Service default/foo
├── [f3.yaml]  .Deployment default/foo
├── bar-package
│   └── [f2.yaml]  .Deployment bar
└── foo-package
    └── [f1.yaml]  .Deployment default/foo
`, out.String()) {
		t.FailNow()
	}
}

func TestPrinter_Write_sort(t *testing.T) {
	in := `apiVersion: extensions/v1
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
`
	out := &bytes.Buffer{}
	err := Pipeline{
		Inputs:  []Reader{ByteReader{Reader: bytes.NewBufferString(in)}},
		Outputs: []Writer{TreeWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Equal(t, `
├── [f1.yaml]  apps/v1.Deployment default/foo
├── [f1.yaml]  .Service default/foo
├── [f1.yaml]  apps/v1.Deployment default/foo3
├── [f1.yaml]  extensions/v1.Deployment default/foo3
├── [f1.yaml]  extensions/v1.Deployment default/foo3
├── [f1.yaml]  apps/v1.Deployment default2/foo2
└── bar-package
    └── [f2.yaml]  .Deployment bar
`, out.String()) {
		t.FailNow()
	}
}

func TestPrinter_metaError(t *testing.T) {
	out := &bytes.Buffer{}
	err := TreeWriter{Writer: out}.Write([]*yaml.RNode{{}})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, `
`, out.String()) {
		t.FailNow()
	}
}
