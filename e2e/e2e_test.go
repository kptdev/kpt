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

package e2e_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/e2e"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/run"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/cmd/config/ext"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

func TestKptGetSet(t *testing.T) {
	type testCase struct {
		name         string
		subdir       string
		tag          string
		branch       string
		setBy        string
		dataset      string
		replacements map[string][]string

		// the upstream doesn't have a kptfile
		noKptfile bool
	}

	tests := []testCase{
		{name: "subdir", subdir: "helloworld-set",
			dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`          setBy: package-default
          value: "5"`,
					`          value: "7"
          isSet: true`,
				},
			},
		},
		{name: "tag-subdir", tag: "v0.1.0", subdir: "helloworld-set",
			dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`          setBy: package-default
          value: "5"`,
					`          value: "7"
          isSet: true`,
				},
			},
		},
		{name: "tag", tag: "v0.1.0", dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`          setBy: package-default
          value: "5"`,
					`          value: "7"
          isSet: true`,
				},
			},
		},
		{name: "branch", branch: "master",
			dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`          setBy: package-default
          value: "5"`,
					`          value: "7"
          isSet: true`,
				},
			},
		},
		{name: "setBy", setBy: "foo",
			dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`          setBy: package-default
          value: "5"`,
					`          setBy: foo
          value: "7"
          isSet: true`,
				},
			},
		},
		{name: "fn",
			dataset: testutil.HelloWorldFn,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7",
					`    app: hello`,
					`    app: hello
    foo: bar`},
				"service.yaml": {
					`    app: hello`,
					`    app: hello
    foo: bar`},
				"Kptfile": {
					`          setBy: package-default
          value: "5"`,
					`          value: "7"
          isSet: true`,
				},
			},
		},

		// verify things work if there is no kptfile
		{name: "no_kptfile", dataset: testutil.HelloWorldSet, noKptfile: true},

		// verify things work if there is no kptfile
		{name: "fn_no_kptfile", dataset: testutil.HelloWorldFnNoKptfile, noKptfile: true},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			upstreamGit, upstream, cleanActual := e2e.SetupGitRepo(t)
			defer cleanActual()
			upstream += ".git"

			expectedGit, expected, cleanExpected := e2e.SetupGitRepo(t)
			defer cleanExpected()

			testutil.CopyData(t, upstreamGit, test.dataset, test.subdir)
			testutil.Commit(t, upstreamGit, "set")

			// get from a version if one is specified
			var version string
			if test.tag != "" {
				version = "@" + test.tag
				testutil.Tag(t, upstreamGit, test.tag)
			}
			if test.branch != "" {
				version = "@" + test.branch
			}

			// local directory we are fetching to
			d, err := ioutil.TempDir("", "kpt")
			defer os.RemoveAll(d)
			testutil.AssertNoError(t, err)
			testutil.AssertNoError(t, os.Chdir(d))

			// Run Get
			cmd := run.GetMain()
			localDir := "helloworld"
			args := []string{
				"pkg", "get",
				"file://" + filepath.Join(upstream, test.subdir) + version,
				localDir,
			}
			cmd.SetArgs(args)
			e2e.Exec(t, cmd)

			// Validate Get Results
			testutil.CopyData(t, expectedGit, test.dataset, test.subdir)
			testutil.CopyKptfile(t, localDir,
				filepath.Join(expected, test.subdir))

			// Kptfile is missing from upstream -- make sure it was copied correctly and nothing else
			if test.noKptfile {
				// diff requires a kptfile exists
				testutil.CopyKptfile(t, localDir, upstreamGit.RepoDirectory)

				testutil.AssertPkgEqual(t, upstreamGit,
					filepath.Join(expected, test.subdir), localDir)
				return
			}

			testutil.AssertPkgEqual(t, upstreamGit,
				filepath.Join(expected, test.subdir), localDir)

			// Run Set
			cmd = run.GetMain()
			args = []string{"cfg", "set", localDir, "replicas", "7"}
			if test.setBy != "" {
				args = append(args, "--set-by", test.setBy)
			}
			cmd.SetArgs(args)
			e2e.Exec(t, cmd)

			// Validate Set Results
			for k, v := range test.replacements {
				for i := range v {
					if i%2 != 0 {
						continue
					}
					testutil.Replace(t, filepath.Join(expected, test.subdir, k),
						v[i], v[i+1])
				}
			}
			testutil.Compare(t,
				filepath.Join(expected, test.subdir, "Kptfile"),
				filepath.Join(localDir, "Kptfile"))
			testutil.AssertPkgEqual(t, upstreamGit,
				filepath.Join(expected, test.subdir),
				localDir)
		})
	}
}

func TestSetters(t *testing.T) {
	var tests = []struct {
		name              string
		inputOpenAPI      string
		command           string
		input             string
		args              []string
		out               string
		expectedOpenAPI   string
		expectedResources string
		errMsg            string
	}{
		{
			name:    "add replicas",
			command: "create-setter",
			args:    []string{"replicas", "3", "--description", "hello world", "--set-by", "me"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
 `,
			inputOpenAPI: `
apiVersion: v1alpha1
kind: Example
`,
			expectedOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
 `,
		},
		{
			name:    "substitution replicas",
			command: "create-subst",
			args: []string{
				"my-image-subst", "--field-value", "nginx:1.7.9", "--pattern", "${my-image-setter}:${my-tag-setter}"},
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
      - name: sidecar
        image: sidecar:1.7.9
 `,
			inputOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.my-image-setter:
      x-k8s-cli:
        setter:
          name: my-image-setter
          value: "nginx"
    io.k8s.cli.setters.my-tag-setter:
      x-k8s-cli:
        setter:
          name: my-tag-setter
          value: "1.7.9"
 `,
			expectedOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.my-image-setter:
      x-k8s-cli:
        setter:
          name: my-image-setter
          value: "nginx"
    io.k8s.cli.setters.my-tag-setter:
      x-k8s-cli:
        setter:
          name: my-tag-setter
          value: "1.7.9"
    io.k8s.cli.substitutions.my-image-subst:
      x-k8s-cli:
        substitution:
          name: my-image-subst
          pattern: ${my-image-setter}:${my-tag-setter}
          values:
          - marker: ${my-image-setter}
            ref: '#/definitions/io.k8s.cli.setters.my-image-setter'
          - marker: ${my-tag-setter}
            ref: '#/definitions/io.k8s.cli.setters.my-tag-setter'
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9 # {"$kpt-set":"my-image-subst"}
      - name: sidecar
        image: sidecar:1.7.9
 `,
		},
		{
			name:    "set replicas",
			command: "set",
			args:    []string{"replicas", "4", "--description", "hi there", "--set-by", "pw"},
			out:     "set 1 fields\n",
			inputOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
 `,
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
 `,
			expectedOpenAPI: `
apiVersion: v1alpha1
kind: Example
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hi there
      x-k8s-cli:
        setter:
          name: replicas
          value: "4"
          setBy: pw
          isSet: true
 `,
			expectedResources: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 4 # {"$kpt-set":"replicas"}
 `,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			// reset the openAPI afterward
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()

			dir, err := ioutil.TempDir("", "")
			if err != nil {
				t.FailNow()
			}
			defer os.RemoveAll(dir)

			err = ioutil.WriteFile(dir+"/Kptfile", []byte(test.inputOpenAPI), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			old := ext.KRMFileName
			defer func() { ext.KRMFileName = old }()
			ext.KRMFileName = func() string {
				return kptfile.KptFileName
			}

			r, err := ioutil.TempFile(dir, "k8s-cli-*.yaml")
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = ioutil.WriteFile(r.Name(), []byte(test.input), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			cmd := run.GetMain()
			args := []string{"cfg", test.command, dir}
			args = append(args, test.args...)
			cmd.SetArgs(args)
			e2e.Exec(t, cmd)

			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}

			actualResources, err := ioutil.ReadFile(r.Name())
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				strings.TrimSpace(test.expectedResources),
				strings.TrimSpace(string(actualResources))) {
				t.FailNow()
			}

			actualOpenAPI, err := ioutil.ReadFile(dir + "/Kptfile")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t,
				strings.TrimSpace(test.expectedOpenAPI),
				strings.TrimSpace(string(actualOpenAPI))) {
				t.FailNow()
			}
		})
	}
}

func TestLiveCommands(t *testing.T) {
	var tests = []struct {
		name         string
		inputOpenAPI string
		command      string
		args         []string
		out          string
		errMsg       string
	}{
		{
			name:    "test preview command setters pre-check",
			command: "preview",
			inputOpenAPI: `
apiVersion: v1alpha1
kind: OpenAPIfile
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: true
          isSet: false
 `,
			errMsg: `setter replicas is required but not set, please set it to new value and try again`,
		},
		{
			name:    "test apply command setters pre-check",
			command: "apply",
			inputOpenAPI: `
apiVersion: v1alpha1
kind: OpenAPIfile
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: true
          isSet: false
 `,
			errMsg: `setter replicas is required but not set, please set it to new value and try again`,
		},
		{
			name:    "preview command setters pre-check pass",
			command: "preview",
			inputOpenAPI: `
apiVersion: v1alpha1
kind: OpenAPIfile
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: true
          isSet: true
 `,
		},
		{
			name:    "apply command setters pre-check pass",
			command: "apply",
			inputOpenAPI: `
apiVersion: v1alpha1
kind: OpenAPIfile
openAPI:
  definitions:
    io.k8s.cli.setters.replicas:
      description: hello world
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"
          setBy: me
          required: true
          isSet: true
 `,
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			// reset the openAPI afterward
			openapi.ResetOpenAPI()
			defer openapi.ResetOpenAPI()

			dir, err := ioutil.TempDir("", "")
			if err != nil {
				t.FailNow()
			}
			defer os.RemoveAll(dir)

			err = ioutil.WriteFile(dir+"/Kptfile", []byte(test.inputOpenAPI), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			cmd := run.GetMain()

			subCommand, _, err := cmd.Find([]string{"live", test.command})
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			subCommand.RunE = func(_ *cobra.Command, _ []string) error {
				return nil
			}

			args := []string{"live", test.command, dir}
			args = append(args, test.args...)
			cmd.SetArgs(args)
			err = cmd.Execute()

			if test.errMsg != "" {
				if !assert.NotNil(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.errMsg) {
					t.FailNow()
				}
			}

			if test.errMsg == "" && !assert.NoError(t, err) {
				t.FailNow()
			}
		})
	}
}
