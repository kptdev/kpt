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

package command_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/command"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/testutil"
)

type testCase struct {
	name        string   // name of the test case for messaging
	command     []string // the kpt sub-command to run
	updatedFile string   // the file that is updated or read from the dataset
	replaceNew  string   // the new expected value in the file -- if empty, don't replace
	replaceOld  string   // the old value in the file -- ignore if replaceNew is empty
	output      string   // the command output -- if empty, don't compare
	err         string   // error message -- if empty, don't expect and error
}

var setTestCases = []testCase{
	{
		name:        "set-replicas-mysql",
		command:     []string{"set", "replicas", "mysql", "--value", "7"},
		updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
		replaceOld:  "replicas: 3",
		replaceNew:  "replicas: 7",
	},
	{
		name:    "set-replicas-wordpress",
		command: []string{"set", "replicas", "wordpress", "--value", "8"},
		err:     "no matching resources",
	},
	{
		name:        "set-image-mysql",
		command:     []string{"set", "image", "mysql", "--value", "mysql:5.9"},
		updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
		replaceOld:  "- name: mysql\n        image: mysql:5.7",
		replaceNew:  "- name: mysql\n        image: mysql:5.9",
	},
	{
		name:        "set-env-mysql",
		command:     []string{"set", "env", "mysql", "--name", "A", "--value", "B"},
		updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
		replaceOld:  "env:\n        - name: MYSQL_ALLOW_EMPTY_PASSWORD\n          value: \"1\"",
		replaceNew:  "env:\n        - name: MYSQL_ALLOW_EMPTY_PASSWORD\n          value: \"1\"\n        - name: A\n          value: B",
	},

	// Cpu
	{
		name:        "set-cpu-limits-mysql",
		command:     []string{"set", "cpu-limits", "mysql", "--value", "700m"},
		updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
		replaceOld: `        resources:
          requests:
            cpu: 500m
            memory: 1Gi`,
		replaceNew: `        resources:
          limits:
            cpu: 700m
          requests:
            cpu: 500m
            memory: 1Gi`,
	},
	{
		name:        "set-cpu-requests-mysql",
		command:     []string{"set", "cpu-requests", "mysql", "--value", "700m"},
		updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
		replaceOld: `        resources:
          requests:
            cpu: 500m
            memory: 1Gi`,
		replaceNew: `        resources:
          requests:
            cpu: 700m
            memory: 1Gi`,
	},

	// Memory
	{
		name:        "set-memory-limits-mysql",
		command:     []string{"set", "memory-limits", "mysql", "--value", ".7Gi"},
		updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
		replaceOld: `        resources:
          requests:
            cpu: 500m
            memory: 1Gi`,
		replaceNew: `        resources:
          limits:
            memory: .7Gi
          requests:
            cpu: 500m
            memory: 1Gi`,
	},
	{
		name:        "set-memory-requests-mysql",
		command:     []string{"set", "memory-requests", "mysql", "--value", ".5Gi"},
		updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
		replaceOld: `        resources:
          requests:
            cpu: 500m
            memory: 1Gi`,
		replaceNew: `        resources:
          requests:
            cpu: 500m
            memory: .5Gi`,
	},
}

var getTestCases = []testCase{
	{
		name:    "get-replicas-mysql",
		command: []string{"get", "replicas", "mysql"},
		output:  "3\n",
	},
	{
		name:    "get-replicas-wordpress",
		command: []string{"get", "replicas", "wordpress"},
		err:     "no matching resources",
	},
	{
		name:    "get-image-mysql",
		command: []string{"get", "image", "mysql"},
		output:  "mysql:5.7\n",
	},
	{
		name:    "get-env-mysql",
		command: []string{"get", "env", "mysql", "--name", "MYSQL_ALLOW_EMPTY_PASSWORD"},
		output:  "\"1\"\n",
	},

	{
		name:    "get-cpu-limits-mysql",
		command: []string{"get", "cpu-limits", "mysql"},
		err:     "no matching resources",
	},
	{
		name:    "get-cpu-requests-mysql",
		command: []string{"get", "cpu-requests", "mysql"},
		output:  "500m\n",
	},
	{
		name:    "get-memory-limits-mysql",
		command: []string{"get", "memory-limits", "mysql"},
		err:     "no matching resources",
	},
	{
		name:    "get-memory-requests-mysql",
		command: []string{"get", "memory-requests", "mysql"},
		output:  "1Gi\n",
	},
}

func Test(t *testing.T) {

	for _, test := range append(getTestCases, setTestCases...) {
		func() {
			g, _, clean := testutil.SetupDefaultRepoAndWorkspace(t)
			defer clean()
			if !assert.NoError(t, os.Chdir(filepath.Dir(g.RepoDirectory)), test.name) {
				return
			}
			c := filepath.Base(g.RepoDirectory)

			cmd := &cobra.Command{Use: "kpt"}
			if !assert.NoError(t, AddCommands(c, cmd), test.name) {
				return
			}
			b := &bytes.Buffer{}
			cmd.SetOut(b)
			cmd.SetArgs(append([]string{c}, test.command...))
			err := cmd.Execute()

			if test.err != "" {
				if !assert.Error(t, err, test.name) {
					return
				}
				if !assert.Contains(t, err.Error(), test.err, test.name) {
					return
				}
			} else {
				if !assert.NoError(t, err, test.name) {
					return
				}
			}

			if test.output != "" {
				assert.Equal(t, test.output, b.String(), test.name)
				// don't return, verify the output hasn't changed
			}

			if test.replaceNew != "" {
				// verify the replicas were updated
				original, err := ioutil.ReadFile(filepath.Join(
					g.DatasetDirectory, testutil.Dataset1, test.updatedFile))
				if !assert.NoError(t, err, test.name) {
					return
				}
				expected := strings.Replace(string(original), test.replaceOld, test.replaceNew, 1)
				actual, err := ioutil.ReadFile(filepath.Join(c, test.updatedFile))
				if !assert.NoError(t, err, test.name) {
					return
				}
				if !assert.Equal(t, expected, string(actual), test.name) {
					return
				}

				if !assert.NoError(t, ioutil.WriteFile(filepath.Join(c, test.updatedFile), original, 0600)) {
					return
				}
			}

			// assert no other files changed
			g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), c)
		}()
	}
}

func Test_stdin(t *testing.T) {
	for _, test := range setTestCases {
		func() {
			g, _, clean := testutil.SetupDefaultRepoAndWorkspace(t)
			defer clean()
			if !assert.NoError(t, os.Chdir(filepath.Dir(g.RepoDirectory)), test.name) {
				return
			}
			c := filepath.Base(g.RepoDirectory)

			cmd := &cobra.Command{Use: "kpt"}
			if !assert.NoError(t, AddCommands(Duck, cmd), test.name) {
				return
			}

			// setup the input for the command
			bin := &bytes.Buffer{}
			nodes, err := kio.LocalPackageReader{PackagePath: c}.Read()
			if !assert.NoError(t, err) {
				return
			}
			err = kio.ByteWriter{Writer: bin, KeepReaderAnnotations: true}.Write(nodes)
			if !assert.NoError(t, err) {
				return
			}
			bout := &bytes.Buffer{}
			cmd.SetIn(bin)
			cmd.SetOut(bout)

			// setup the command args
			cmd.SetArgs(append([]string{Duck}, test.command...))
			err = cmd.Execute()

			// check if there should be an error
			if test.err != "" {
				if !assert.Error(t, err, test.name) {
					return
				}
				if !assert.Contains(t, err.Error(), test.err, test.name) {
					return
				}
				return
			} else {
				if !assert.NoError(t, err, test.name) {
					return
				}
			}

			// for get commands
			if test.output != "" {
				assert.Equal(t, test.output, bout.String(), test.name)
				return
			}

			// for set commands
			if test.replaceNew != "" {
				// read the golden file
				original, err := ioutil.ReadFile(filepath.Join(
					g.DatasetDirectory, testutil.Dataset1, test.updatedFile))
				if !assert.NoError(t, err, test.name) {
					return
				}
				// set the expected updated value
				expected := strings.Replace(string(original), test.replaceOld, test.replaceNew, 1)

				// update the file so we can compare the directories
				if !assert.NoError(t, ioutil.WriteFile(filepath.Join(c, test.updatedFile), []byte(expected), 0600)) {
					return
				}
			}

			nodes, err = kio.LocalPackageReader{PackagePath: c}.Read()
			if !assert.NoError(t, err, test.name) {
				return
			}
			expected := &bytes.Buffer{}
			err = kio.ByteWriter{Writer: expected, KeepReaderAnnotations: true}.Write(nodes)
			if !assert.NoError(t, err, test.name) {
				return
			}
			expected, err = filters.FormatInput(expected)
			if !assert.NoError(t, err) {
				return
			}
			bout, err = filters.FormatInput(bout)
			if !assert.NoError(t, err, test.name) {
				return
			}

			if !assert.Equal(t, expected.String(), bout.String(), test.name) {
				return
			}
		}()
	}
}

func TestCommandDuckRegister(t *testing.T) {
	dir := copyTestData(t, ".")
	if !assert.NoError(t, os.Chdir(dir)) {
		t.FailNow()
	}
	cmd := &cobra.Command{Use: "kpt"}
	if !assert.NoError(t, AddCommands(dir, cmd)) {
		return
	}
	b := &bytes.Buffer{}
	cmd.SetOut(b)
	cmd.SetArgs([]string{dir, "get", "service-name", "foo"})
	err := cmd.Execute()
	if !assert.NoError(t, err) {
		return
	}
	if !assert.Equal(t, "foo-service\n", b.String()) {
		return
	}

	cmd.SetArgs([]string{dir, "set", "service-name", "foo", "--value=foo2-service"})
	err = cmd.Execute()
	if !assert.NoError(t, err) {
		return
	}

	result, err := ioutil.ReadFile(filepath.Join(dir, "resource.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  labels:
    app: nginx
  annotations:
    name: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: foo
  labels:
    app: foo
  annotations:
    name: foo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: foo
  template:
    metadata:
      labels:
        app: foo
    spec:
      containers:
      - name: foo
        image: foo:v1.0.0
        ports:
        - containerPort: 80
  serviceName: foo2-service
---
apiVersion: v1
kind: Service
metadata:
  name: foo-service
spec:
  selector:
    app: foo
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
`, string(result))
}
