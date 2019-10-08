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

package cmdxargs_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"kpt.dev/cmdreconcile/cmdxargs"
)

const (
	flagsInput = `kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  spec:
    template:
      spec:
        containers:
        - name: nginx
          image: nginx
- apiVersion: apps/v1
  kind: Service
  spec: {}
functionConfig:
  kind: Foo
  spec:
    a: b
    c: d
    e: f
  items:
  - 1
  - 3
  - 2
  - 4
`

	resourceInput = `apiVersion: kpt.dev/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  spec:
    template:
      spec:
        containers:
        - name: nginx
          image: nginx
- apiVersion: apps/v1
  kind: Service
  spec: {}
functionConfig:
  kind: Foo
`

	resourceOutput = `apiVersion: v1
kind: List
items:
- apiVersion: apps/v1
  kind: Deployment
  spec:
    template:
      spec:
        containers:
        - name: nginx
          image: nginx
- apiVersion: apps/v1
  kind: Service
  spec: {}
`
)

func TestCmd_flags(t *testing.T) {
	c := cmdxargs.Cmd()
	c.C.SetIn(bytes.NewBufferString(flagsInput))
	out := &bytes.Buffer{}
	c.C.SetOut(out)
	c.C.SetArgs([]string{"--", "echo"})

	c.Args = []string{"--", "echo"}
	if !assert.NoError(t, c.C.Execute()) {
		t.FailNow()
	}
	assert.Equal(t, `--a=b --c=d --e=f 1 3 2 4
`, out.String())
}

func TestCmd_input(t *testing.T) {
	c := cmdxargs.Cmd()
	c.C.SetIn(bytes.NewBufferString(resourceInput))
	out := &bytes.Buffer{}
	c.C.SetOut(out)
	c.C.SetArgs([]string{"--", "cat"})

	c.Args = []string{"--", "cat"}
	if !assert.NoError(t, c.C.Execute()) {
		t.FailNow()
	}
	assert.Equal(t, resourceOutput, out.String())
}

func TestCmd_env(t *testing.T) {
	c := cmdxargs.Cmd()
	c.C.SetIn(bytes.NewBufferString(flagsInput))
	out := &bytes.Buffer{}
	c.C.SetOut(out)
	c.C.SetArgs([]string{"--env-only", "--", "env"})

	c.Args = []string{"--", "env"}
	if !assert.NoError(t, c.C.Execute()) {
		t.FailNow()
	}
	assert.Contains(t, out.String(), "\nA=b\nC=d\nE=f\n")
}
