// Copyright 2019 The kpt Authors
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

package diff_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/commands/pkg/diff"
	"github.com/GoogleContainerTools/kpt/commands/pkg/get"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

func TestCmdInvalidDiffType(t *testing.T) {
	runner := diff.NewRunner(fake.CtxWithDefaultPrinter(), "")
	runner.C.SetArgs([]string{"--diff-type", "invalid"})
	err := runner.C.Execute()
	assert.EqualError(t,
		err,
		"invalid diff-type 'invalid': supported diff-types are: local, remote, combined, 3way")
}

func TestCmdInvalidDiffTool(t *testing.T) {
	runner := diff.NewRunner(fake.CtxWithDefaultPrinter(), "")
	runner.C.SetArgs([]string{"--diff-tool", "nodiff"})
	err := runner.C.Execute()
	assert.EqualError(t,
		err,
		"diff-tool 'nodiff' not found in the PATH")
}

func TestCmdExecute(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	getRunner := get.NewRunner(fake.CtxWithDefaultPrinter(), "")
	getRunner.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git/", "./"})
	err := getRunner.Command.Execute()
	assert.NoError(t, err)

	runner := diff.NewRunner(fake.CtxWithDefaultPrinter(), "")
	runner.C.SetArgs([]string{dest, "--diff-type", "local"})
	err = runner.C.Execute()
	assert.NoError(t, err)
}

func TestCmd_flagAndArgParsing_Symlink(t *testing.T) {
	dir := t.TempDir()
	defer testutil.Chdir(t, dir)()

	err := os.MkdirAll(filepath.Join(dir, "path", "to", "pkg", "dir"), 0700)
	assert.NoError(t, err)
	err = os.Symlink(filepath.Join("path", "to", "pkg", "dir"), "foo")
	assert.NoError(t, err)

	// verify the branch ref is set to the correct value
	r := diff.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"foo" + "@refs/heads/foo"})
	err = r.C.Execute()
	assert.NoError(t, err)
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(cwd, "path", "to", "pkg", "dir"), r.Path)
}

var NoOpRunE = func(cmd *cobra.Command, args []string) error { return nil }
