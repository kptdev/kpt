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
	"testing"

	"github.com/GoogleContainerTools/kpt/e2e"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/run"
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
					`                    setBy: package-default
                    value: "5"`,
					`                    value: "7"`,
				},
			},
		},
		{name: "tag-subdir", tag: "v0.1.0", subdir: "helloworld-set",
			dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`                    setBy: package-default
                    value: "5"`,
					`                    value: "7"`,
				},
			},
		},
		{name: "tag", tag: "v0.1.0", dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`                    setBy: package-default
                    value: "5"`,
					`                    value: "7"`,
				},
			},
		},
		{name: "branch", branch: "master",
			dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`                    setBy: package-default
                    value: "5"`,
					`                    value: "7"`,
				},
			},
		},
		{name: "setBy", setBy: "foo",
			dataset: testutil.HelloWorldSet,
			replacements: map[string][]string{
				"deploy.yaml": {"replicas: 5", "replicas: 7"},
				"Kptfile": {
					`                    setBy: package-default
                    value: "5"`,
					`                    setBy: foo
                    value: "7"`,
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
					`                    setBy: package-default
                    value: "5"`,
					`                    value: "7"`,
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

				testutil.AssertEqual(t, upstreamGit,
					filepath.Join(expected, test.subdir), localDir)
				return
			}

			testutil.AssertEqual(t, upstreamGit,
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
			testutil.AssertEqual(t, upstreamGit,
				filepath.Join(expected, test.subdir),
				localDir)
		})
	}
}
