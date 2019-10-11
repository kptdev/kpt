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

package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lib.kpt.dev/testutil"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	var tests = []struct {
		name        string   // name of the test case for messaging
		command     []string // the kpt sub-command to run
		updatedFile string   // the file that is updated or read from the dataset
		replaceNew  string   // the new expected value in the file -- if empty, don't replace
		replaceOld  string   // the old value in the file -- ignore if replaceNew is empty
		output      string   // the command output -- if empty, don't compare
		err         string   // error message -- if empty, don't expect and error
	}{
		{
			name:        "set-replicas-mysql",
			command:     []string{"set", "replicas", "mysql", "--value", "7"},
			updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
			replaceOld:  "replicas: 3",
			replaceNew:  "replicas: 7",
		},
		{
			name:    "get-replicas-mysql",
			command: []string{"get", "replicas", "mysql"},
			output:  "3\n",
		},
		{
			name:    "set-replicas-wordpress",
			command: []string{"set", "replicas", "wordpress", "--value", "8"},
			err:     "no matching resources",
		},
		{
			name:    "get-replicas-wordpress",
			command: []string{"get", "replicas", "wordpress"},
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
			name:    "get-image-mysql",
			command: []string{"get", "image", "mysql"},
			output:  "mysql:5.7\n",
		},
		{
			name:        "set-env-mysql",
			command:     []string{"set", "env", "mysql", "--name", "A", "--value", "B"},
			updatedFile: filepath.Join("mysql", "mysql-statefulset.resource.yaml"),
			replaceOld:  "env:\n        - name: MYSQL_ALLOW_EMPTY_PASSWORD\n          value: \"1\"",
			replaceNew:  "env:\n        - name: MYSQL_ALLOW_EMPTY_PASSWORD\n          value: \"1\"\n        - name: A\n          value: B",
		},
		{
			name:    "get-env-mysql",
			command: []string{"get", "env", "mysql", "--name", "MYSQL_ALLOW_EMPTY_PASSWORD"},
			output:  "\"1\"\n",
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
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 700m`,
		},
		{
			name:    "get-cpu-limits-mysql",
			command: []string{"get", "cpu-limits", "mysql"},
			err:     "no matching resources",
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
		{
			name:    "get-cpu-requests-mysql",
			command: []string{"get", "cpu-requests", "mysql"},
			output:  "500m\n",
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
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            memory: .7Gi`,
		},
		{
			name:    "get-memory-limits-mysql",
			command: []string{"get", "memory-limits", "mysql"},
			err:     "no matching resources",
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
		{
			name:    "get-memory-requests-mysql",
			command: []string{"get", "memory-requests", "mysql"},
			output:  "1Gi\n",
		},
	}

	for _, test := range tests {
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
