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

package filters

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

func TestFilter_command(t *testing.T) {
	cfg, err := yaml.Parse(`apiversion: apps/v1
kind: Deployment
metadata:
  name: foo
`)
	if !assert.NoError(t, err) {
		return
	}
	instance := &ContainerFilter{
		Image:  "example.com:version",
		Config: cfg,
	}
	os.Setenv("KPT_TEST", "FOO")
	cmd, err := instance.getCommand()
	if !assert.NoError(t, err) {
		return
	}

	expected := []string{
		"docker", "run",
		"--rm",
		"-i", "-a", "STDIN", "-a", "STDOUT", "-a", "STDERR",
		"--network", "none",
		"--user", "nobody",
		"--security-opt=no-new-privileges",
	}
	for _, e := range os.Environ() {
		// the process env
		expected = append(expected, "-e", strings.Split(e, "=")[0])
	}
	expected = append(expected, "example.com:version")
	assert.Equal(t, expected, cmd.Args)

	foundKpt := false
	for _, e := range cmd.Env {
		// verify the command has the right environment variables to pass to the container
		split := strings.Split(e, "=")
		if split[0] == "KPT_TEST" {
			assert.Equal(t, "FOO", split[1])
			foundKpt = true
		}
	}
	assert.True(t, foundKpt)
}

func TestFilter_Filter(t *testing.T) {
	cfg, err := yaml.Parse(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`)
	if !assert.NoError(t, err) {
		return
	}

	input, err := (&kio.ByteReader{Reader: bytes.NewBufferString(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`)}).Read()
	if !assert.NoError(t, err) {
		return
	}

	called := false
	result, err := (&ContainerFilter{
		Image:  "example.com:version",
		Config: cfg,
		args:   []string{"sed", "s/Deployment/StatefulSet/g"},
		checkInput: func(s string) {
			called = true
			if !assert.Equal(t, `apiVersion: kpt.dev/v1alpha1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: deployment-foo
    annotations:
      kpt.dev/kio/index: 0
- apiVersion: v1
  kind: Service
  metadata:
    name: service-foo
    annotations:
      kpt.dev/kio/index: 1
functionConfig: {apiVersion: apps/v1, kind: Deployment, metadata: {name: foo}}
`, s) {
				t.FailNow()
			}
		},
	}).Filter(input)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.True(t, called) {
		return
	}

	b := &bytes.Buffer{}
	err = kio.ByteWriter{Writer: b, KeepReaderAnnotations: true}.Write(result)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: deployment-foo
  annotations:
    kpt.dev/kio/index: 0
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
  annotations:
    kpt.dev/kio/index: 1
`, b.String())
}

func TestFilter_Filter_noChange(t *testing.T) {
	cfg, err := yaml.Parse(`apiversion: apps/v1
kind: Deployment
metadata:
  name: foo
`)
	if !assert.NoError(t, err) {
		return
	}

	input, err := (&kio.ByteReader{Reader: bytes.NewBufferString(`
apiversion: apps/v1
kind: Deployment
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`)}).Read()
	if !assert.NoError(t, err) {
		return
	}

	called := false
	result, err := (&ContainerFilter{
		Image:  "example.com:version",
		Config: cfg,
		args:   []string{"sh", "-c", "cat <&0"},
		checkInput: func(s string) {
			called = true
			if !assert.Equal(t, `apiVersion: kpt.dev/v1alpha1
kind: ResourceList
items:
- apiversion: apps/v1
  kind: Deployment
  metadata:
    name: deployment-foo
    annotations:
      kpt.dev/kio/index: 0
- apiVersion: v1
  kind: Service
  metadata:
    name: service-foo
    annotations:
      kpt.dev/kio/index: 1
functionConfig: {apiversion: apps/v1, kind: Deployment, metadata: {name: foo}}
`, s) {
				t.FailNow()
			}
		},
	}).Filter(input)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.True(t, called) {
		return
	}

	b := &bytes.Buffer{}
	err = kio.ByteWriter{Writer: b, KeepReaderAnnotations: true}.Write(result)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, `apiversion: apps/v1
kind: Deployment
metadata:
  name: deployment-foo
  annotations:
    kpt.dev/kio/index: 0
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
  annotations:
    kpt.dev/kio/index: 1
`, b.String())
}

func Test_GetContainerName(t *testing.T) {
	// make sure gcr.io works
	n, err := yaml.Parse(`apiVersion: gcr.io/foo/bar:something
kind: MyThing
`)
	if !assert.NoError(t, err) {
		return
	}
	c := GetContainerName(n)
	assert.Equal(t, "gcr.io/foo/bar:something", c)

	// make sure regional gcr.io works
	n, err = yaml.Parse(`apiVersion: us.gcr.io/foo/bar:something
kind: MyThing
`)
	if !assert.NoError(t, err) {
		return
	}
	c = GetContainerName(n)
	assert.Equal(t, "us.gcr.io/foo/bar:something", c)

	// container from annotation
	n, err = yaml.Parse(`apiVersion: v1
kind: MyThing
metadata:
  annotations:
    kpt.dev/container: gcr.io/foo/bar:something
`)
	if !assert.NoError(t, err) {
		return
	}
	c = GetContainerName(n)
	assert.Equal(t, "gcr.io/foo/bar:something", c)

	// doesn't have a container
	n, err = yaml.Parse(`apiVersion: v1
kind: MyThing
metadata:
`)
	if !assert.NoError(t, err) {
		return
	}
	c = GetContainerName(n)
	assert.Equal(t, "", c)

	// make sure docker.io works
	n, err = yaml.Parse(`apiVersion: docker.io/foo/bar:something
kind: MyThing
`)
	if !assert.NoError(t, err) {
		return
	}
	c = GetContainerName(n)
	assert.Equal(t, "docker.io/foo/bar:something", c)
}
