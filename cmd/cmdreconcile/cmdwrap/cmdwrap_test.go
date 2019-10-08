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

package cmdwrap

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	input = `apiVersion: kpt.dev/v1
kind: ResourceList
functionConfig:
  metadata:
    name: test
  spec:
    replicas: 11
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: test
    labels:
      app: nginx
      name: test
  spec:
    replicas: 5
    selector:
      matchLabels:
        app: nginx
        name: test
    template:
      metadata:
        labels:
          app: nginx
          name: test
      spec:
        containers:
        - name: test
          image: nginx:v1.7
          ports:
          - containerPort: 8080
            name: http
          resources:
            limits:
              cpu: 500m
- apiVersion: v1
  kind: Service
  metadata:
    name: test
    labels:
      app: nginx
      name: test
  spec:
    ports:
    # This i the port.
    - port: 8080
      targetPort: 8080
      name: http
    selector:
      app: nginx
      name: test
`

	output = `apiVersion: kpt.dev/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: test
    labels:
      name: test
      app: nginx
    annotations:
      kpt.dev/kio/index: 0
      kpt.dev/kio/path: config/test_deployment.yaml
  spec:
    replicas: 11
    selector:
      matchLabels:
        name: test
        app: nginx
    template:
      metadata:
        labels:
          name: test
          app: nginx
      spec:
        containers:
        - name: test
          image: nginx:v1.7
          ports:
          - name: http
            containerPort: 8080
          resources:
            limits:
              cpu: 500m
- apiVersion: v1
  kind: Service
  metadata:
    name: test
    labels:
      name: test
      app: nginx
    annotations:
      kpt.dev/kio/index: 0
      kpt.dev/kio/path: config/test_service.yaml
  spec:
    selector:
      name: test
      app: nginx
    ports:
    - name: http
      # This i the port.
      port: 8080
      targetPort: 8080
`

	outputNoMerge = `apiVersion: kpt.dev/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: test
    labels:
      name: test
      app: nginx
    annotations:
      kpt.dev/kio/index: 0
      kpt.dev/kio/path: config/test_deployment.yaml
  spec:
    replicas: 11
    selector:
      matchLabels:
        name: test
        app: nginx
    template:
      metadata:
        labels:
          name: test
          app: nginx
      spec:
        containers:
        - name: test
          image: nginx:v1.7
          ports:
          - name: http
            containerPort: 8080
- apiVersion: v1
  kind: Service
  metadata:
    name: test
    labels:
      name: test
      app: nginx
    annotations:
      kpt.dev/kio/index: 0
      kpt.dev/kio/path: config/test_service.yaml
  spec:
    selector:
      name: test
      app: nginx
    ports:
    - name: http
      # This i the port.
      port: 8080
      targetPort: 8080
`

	outputOverride = `apiVersion: kpt.dev/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: test
    labels:
      name: test
      app: nginx
    annotations:
      kpt.dev/kio/index: 0
      kpt.dev/kio/path: config/test_deployment.yaml
  spec:
    replicas: 11
    selector:
      matchLabels:
        name: test
        app: nginx
    template:
      metadata:
        labels:
          name: test
          app: nginx
      spec:
        containers:
        - name: test
          image: nginx:v1.9
          ports:
          - name: http
            containerPort: 8080
          resources:
            limits:
              cpu: 500m
- apiVersion: v1
  kind: Service
  metadata:
    name: test
    labels:
      name: test
      app: nginx
    annotations:
      kpt.dev/kio/index: 0
      kpt.dev/kio/path: config/test_service.yaml
  spec:
    selector:
      name: test
      app: nginx
    ports:
    - name: http
      # This i the port.
      port: 8080
      targetPort: 8080
`
)

func TestCmd_wrap(t *testing.T) {
	_, dir, _, ok := runtime.Caller(0)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	dir = filepath.Dir(dir)

	c := Cmd()
	c.C.SetIn(bytes.NewBufferString(input))
	out := &bytes.Buffer{}
	c.C.SetOut(out)
	args := []string{"--", filepath.Join(dir, "test", "test.sh")}
	c.C.SetArgs(args)
	c.Xargs.Args = args

	if !assert.NoError(t, c.C.Execute()) {
		t.FailNow()
	}

	assert.Equal(t, output, out.String())
}

func TestCmd_wrapNoMerge(t *testing.T) {
	_, dir, _, ok := runtime.Caller(0)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	dir = filepath.Dir(dir)

	c := Cmd()
	c.getEnv = func(key string) string {
		if key == KptMerge {
			return "false"
		}
		return ""
	}
	c.C.SetIn(bytes.NewBufferString(input))
	out := &bytes.Buffer{}
	c.C.SetOut(out)
	args := []string{"--", filepath.Join(dir, "test", "test.sh")}
	c.C.SetArgs(args)
	c.Xargs.Args = args
	if !assert.NoError(t, c.C.Execute()) {
		t.FailNow()
	}

	assert.Equal(t, outputNoMerge, out.String())
}

func TestCmd_wrapOverride(t *testing.T) {
	_, dir, _, ok := runtime.Caller(0)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	dir = filepath.Dir(dir)

	c := Cmd()
	c.getEnv = func(key string) string {
		if key == KptOverridePkg {
			return filepath.Join(dir, "test")
		}
		return ""
	}
	c.C.SetIn(bytes.NewBufferString(input))
	out := &bytes.Buffer{}
	c.C.SetOut(out)
	args := []string{"--", filepath.Join(dir, "test", "test.sh")}
	c.C.SetArgs(args)
	c.Xargs.Args = args
	if !assert.NoError(t, c.C.Execute()) {
		t.FailNow()
	}

	assert.Equal(t, outputOverride, out.String())
}
