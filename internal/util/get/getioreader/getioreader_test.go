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

package getioreader_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/GoogleContainerTools/kpt/internal/util/get/getioreader"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)
	b := bytes.NewBufferString(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo2
---
apiVersion: v1
kind: Service
metadata:
  name: foo2
  namespace: bar
---
apiVersion: v1
kind: Service
metadata:
  name: foo1
`)
	if !assert.NoError(t, Get(d, "", b)) {
		return
	}
	actual, err := ioutil.ReadFile(filepath.Join(d, "foo1_deployment.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
`, string(actual))

	actual, err = ioutil.ReadFile(filepath.Join(d, "foo2_deployment.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo2
`, string(actual))

	actual, err = ioutil.ReadFile(filepath.Join(d, "foo2_service.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: v1
kind: Service
metadata:
  name: foo2
  namespace: bar
`, string(actual))

	actual, err = ioutil.ReadFile(filepath.Join(d, "foo1_service.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: v1
kind: Service
metadata:
  name: foo1
`, string(actual))
}

func TestGet_pattern(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)
	b := bytes.NewBufferString(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
`)
	if !assert.NoError(t, Get(d, "%k.yaml", b)) {
		return
	}
	actual, err := ioutil.ReadFile(filepath.Join(d, "deployment.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
`, string(actual))
}
