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

package cmdupdate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"kpt.dev/cmdget"
	"kpt.dev/cmdupdate"
	"kpt.dev/internal/gitutil"
	"kpt.dev/internal/pkgfile"
	"kpt.dev/internal/testutil"
	"kpt.dev/internal/update"
	"lib.kpt.dev/yaml"
)

// TestCmd_execute verifies that update is correctly invoked.
func TestCmd_execute(t *testing.T) {
	g, dir, clean := testutil.SetupDefaultRepoAndWorkspace(t)
	defer clean()
	dest := filepath.Join(dir, g.RepoName)

	// clone the repo
	getCmd := cmdget.Cmd()
	getCmd.C.SetArgs([]string{"file://" + g.RepoDirectory + ".git", dir})
	err := getCmd.C.Execute()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest) {
		return
	}
	gitRunner := gitutil.NewLocalGitRunner(dir)
	if !assert.NoError(t, gitRunner.Run("add", ".")) {
		return
	}
	if !assert.NoError(t, gitRunner.Run("commit", "-m", "commit local package -- ds1")) {
		return
	}

	// update the master branch
	if !assert.NoError(t, g.ReplaceData(testutil.Dataset2)) {
		return
	}
	if !assert.NoError(t, g.Commit("modify upstream package -- ds2")) {
		return
	}

	// update the cloned package
	updateCmd := cmdupdate.Cmd()
	if !assert.NoError(t, os.Chdir(dir)) {
		return
	}
	updateCmd.C.SetArgs([]string{g.RepoName})
	if !assert.NoError(t, updateCmd.C.Execute()) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), dest) {
		return
	}

	commit, err := g.GetCommit()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertKptfile(t, dest, pkgfile.KptFile{
		ResourceMeta: yaml.NewResourceMeta(g.RepoName, pkgfile.TypeMeta),
		PackageMeta:  pkgfile.PackageMeta{},
		Upstream: pkgfile.Upstream{
			Type: "git",
			Git: pkgfile.Git{
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Directory: "/",
				Commit:    commit,
			},
		},
	}) {
		return
	}
}

func TestCmd_failUnCommitted(t *testing.T) {
	g, dir, clean := testutil.SetupDefaultRepoAndWorkspace(t)
	defer clean()
	dest := filepath.Join(dir, g.RepoName)

	// clone the repo
	getCmd := cmdget.Cmd()
	getCmd.C.SetArgs([]string{"file://" + g.RepoDirectory + ".git", dir})
	err := getCmd.C.Execute()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest) {
		return
	}

	// update the master branch
	if !assert.NoError(t, g.ReplaceData(testutil.Dataset2)) {
		return
	}

	if !assert.NoError(t, g.Commit("new dataset")) {
		return
	}

	// update the cloned package
	updateCmd := cmdupdate.Cmd()
	if !assert.NoError(t, os.Chdir(dir)) {
		return
	}
	updateCmd.C.SetArgs([]string{g.RepoName})
	err = updateCmd.C.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "must commit package")

	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest) {
		return
	}
}

// NoOpRunE is a noop function to replace the run function of a command.  Useful for testing argument parsing.
var NoOpRunE = func(cmd *cobra.Command, args []string) error { return nil }

// NoOpFailRunE causes the test to fail if run is called.  Useful for validating run isn't called for
// errors.
type NoOpFailRunE struct {
	t *testing.T
}

func (t NoOpFailRunE) runE(cmd *cobra.Command, args []string) error {
	assert.Fail(t.t, "run should not be called")
	return nil
}

// TestCmd_Execute_flagAndArgParsing verifies that the flags and args are parsed into the correct Command fields
func TestCmd_Execute_flagAndArgParsing(t *testing.T) {
	failRun := NoOpFailRunE{t: t}.runE

	// verify the current working directory is used if no path is specified
	r := cmdupdate.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{})
	err := r.C.Execute()
	assert.EqualError(t, err, "accepts 1 arg(s), received 0")
	assert.Equal(t, "", r.Command.Ref)
	assert.Equal(t, update.Default, r.Command.Strategy)

	// verify an error is thrown if multiple paths are specified
	r = cmdupdate.Cmd()
	r.C.SilenceErrors = true
	r.C.RunE = failRun
	r.C.SetArgs([]string{"foo", "bar"})
	err = r.C.Execute()
	assert.EqualError(t, err, "accepts 1 arg(s), received 2")
	assert.Equal(t, "", r.Command.Ref)
	assert.Equal(t, update.Default, r.Command.Strategy)

	// verify the branch ref is set to the correct value
	r = cmdupdate.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"foo@refs/heads/foo"})
	err = r.C.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "foo", r.Command.Path)
	assert.Equal(t, "refs/heads/foo", r.Command.Ref)
	assert.Equal(t, update.FastForward, r.Command.Strategy)

	// verify the branch ref is set to the correct value
	r = cmdupdate.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"foo", "--strategy", "force-delete-replace"})
	err = r.C.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "foo", r.Command.Path)
	assert.Equal(t, update.ForceDeleteReplace, r.Command.Strategy)
	assert.Equal(t, "", r.Command.Ref)
}

// TestCmd_fail verifies that that command returns an error when it fails rather than exiting the process
func TestCmd_fail(t *testing.T) {
	r := cmdupdate.Cmd()
	r.C.SilenceErrors = true
	r.C.SilenceUsage = true
	r.C.SetArgs([]string{filepath.Join("not", "real", "dir")})
	err := r.C.Execute()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "no such file or directory")
	}
}
