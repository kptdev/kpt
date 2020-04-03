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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/e2e"
	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/run"
)

func TestKptGetSet(t *testing.T) {

	type testCase struct {
		name   string
		subdir string
		tag    string
		branch string
		setBy  string
	}

	tests := []testCase{
		{name: "subdir", subdir: "helloworld-set"},
		{name: "tag-subdir", tag: "v0.1.0", subdir: "helloworld-set"},
		{name: "tag", tag: "v0.1.0"},
		{name: "branch", branch: "master"},
		ß{name: "setBy", setBy: "foo"},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			upstreamGit, upstream, cleanActual := e2e.SetupGitRepo(t)
			defer cleanActual()
			upstream += ".git"

			expectedGit, expected, cleanExpected := e2e.SetupGitRepo(t)
			defer cleanExpected()

			testutil.CopyData(t, upstreamGit,
				testutil.HelloWorldSet, test.subdir)
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
			testutil.CopyData(t, expectedGit,
				testutil.HelloWorldSet, test.subdir)
			testutil.CopyKptfile(t,
				localDir,
				filepath.Join(expected, test.subdir))
			testutil.AssertEqual(t, upstreamGit,
				filepath.Join(expected, test.subdir),
				localDir)

			// Run Set
			cmd = run.GetMain()
			args = []string{"cfg", "set", localDir, "replicas", "7"}
			if test.setBy != "" {
				args = append(args, "--set-by", test.setBy)
			}
			cmd.SetArgs(args)
			e2e.Exec(t, cmd)

			// Validate Set Results
			testutil.Replace(t,
				filepath.Join(expected, test.subdir, "deploy.yaml"),
				"replicas: 5",
				"replicas: 7")
			old := `                    setBy: package-default
                    value: "5"`
			new := `                    value: "7"`
			if test.setBy != "" {
				new = fmt.Sprintf(`                    setBy: %s
%s`, test.setBy, new)
			}
			testutil.Replace(t,
				filepath.Join(expected, test.subdir, kptfile.KptFileName),
				old, new,
			)
			testutil.Compare(t,
				filepath.Join(expected, test.subdir, "Kptfile"),
				filepath.Join(localDir, "Kptfile"))
			testutil.AssertEqual(t, upstreamGit,
				filepath.Join(expected, test.subdir),
				localDir)
		})
	}
}
