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
		"--read-only",
		"--security-opt=no-new-privileges",
		"-e", "API_CONFIG", // the api config
	}
	for _, e := range os.Environ() {
		// the process env
		expected = append(expected, "-e", strings.Split(e, "=")[0])
	}
	expected = append(expected, "example.com:version")
	assert.Equal(t, expected, cmd.Args)

	foundApi := false
	foundKpt := false
	for _, e := range cmd.Env {
		// verify the command has the right environment variables to pass to the container
		split := strings.Split(e, "=")
		if split[0] == "API_CONFIG" {
			assert.Equal(t,
				"{apiversion: apps/v1, kind: Deployment, metadata: {name: foo}}\n", split[1])
			foundApi = true
		}
		if split[0] == "KPT_TEST" {
			assert.Equal(t, "FOO", split[1])
			foundKpt = true
		}
	}
	assert.True(t, foundApi)
	assert.True(t, foundKpt)
}

func TestFilter_Filter(t *testing.T) {
	cfg, err := yaml.Parse(`apiversion: apps/v1
kind: Deployment
metadata:
  name: foo
`)
	if !assert.NoError(t, err) {
		return
	}

	input, err := kio.ByteReader{Reader: bytes.NewBufferString(`
apiversion: apps/v1
kind: Deployment
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`)}.Read()
	if !assert.NoError(t, err) {
		return
	}

	result, err := (&ContainerFilter{
		Image:  "example.com:version",
		Config: cfg,
		args:   []string{"sed", "s/Deployment/StatefulSet/g"},
	}).Filter(input)
	if !assert.NoError(t, err) {
		return
	}

	b := &bytes.Buffer{}
	err = kio.ByteWriter{Writer: b}.Write(result)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, `apiversion: apps/v1
kind: StatefulSet
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`, b.String())
}

func TestFilter_Filter_config(t *testing.T) {
	cfg, err := yaml.Parse(`apiversion: apps/v1
kind: Deployment
metadata:
  name: foo
`)
	if !assert.NoError(t, err) {
		return
	}

	input, err := kio.ByteReader{Reader: bytes.NewBufferString(`
apiversion: apps/v1
kind: Deployment
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`)}.Read()
	if !assert.NoError(t, err) {
		return
	}

	result, err := (&ContainerFilter{
		Image:  "example.com:version",
		Config: cfg,
		args:   []string{"sh", "-c", "echo ${API_CONFIG}"},
	}).Filter(input)
	if !assert.NoError(t, err) {
		return
	}

	b := &bytes.Buffer{}
	err = kio.ByteWriter{Writer: b}.Write(result)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, `apiversion: apps/v1
kind: Deployment
metadata:
  name: foo
`, b.String())
}
