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
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/cmdupdate"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestCmd_execute verifies that update is correctly invoked.
func TestCmd_execute(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()
	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// clone the repo
	getCmd := cmdget.NewRunner("kpt")
	getCmd.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git", w.WorkspaceDirectory})
	err := getCmd.Command.Execute()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest) {
		return
	}
	gitRunner := gitutil.NewLocalGitRunner(w.WorkspaceDirectory)
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
	updateCmd := cmdupdate.NewRunner("kpt")
	if !assert.NoError(t, os.Chdir(w.WorkspaceDirectory)) {
		return
	}
	updateCmd.Command.SetArgs([]string{g.RepoName, "--strategy", "fast-forward"})
	if !assert.NoError(t, updateCmd.Command.Execute()) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), dest) {
		return
	}

	commit, err := g.GetCommit()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertKptfile(t, dest, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfile.TypeMeta.APIVersion,
				Kind:       kptfile.TypeMeta.Kind},
		},
		PackageMeta: kptfile.PackageMeta{},
		Upstream: kptfile.Upstream{
			Type: "git",
			Git: kptfile.Git{
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()
	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// clone the repo
	getCmd := cmdget.NewRunner("kpt")
	getCmd.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git", w.WorkspaceDirectory})
	err := getCmd.Command.Execute()
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
	updateCmd := cmdupdate.NewRunner("kpt")
	if !assert.NoError(t, os.Chdir(w.WorkspaceDirectory)) {
		return
	}
	updateCmd.Command.SetArgs([]string{g.RepoName})
	err = updateCmd.Command.Execute()
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
	r := cmdupdate.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{})
	err := r.Command.Execute()
	assert.EqualError(t, err, "accepts 1 arg(s), received 0")
	assert.Equal(t, "", r.Update.Ref)
	assert.Equal(t, update.Default, r.Update.Strategy)

	// verify an error is thrown if multiple paths are specified
	r = cmdupdate.NewRunner("kpt")
	r.Command.SilenceErrors = true
	r.Command.RunE = failRun
	r.Command.SetArgs([]string{"foo", "bar"})
	err = r.Command.Execute()
	assert.EqualError(t, err, "accepts 1 arg(s), received 2")
	assert.Equal(t, "", r.Update.Ref)
	assert.Equal(t, update.Default, r.Update.Strategy)

	// verify the branch ref is set to the correct value
	r = cmdupdate.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo@refs/heads/foo"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "foo", r.Update.Path)
	assert.Equal(t, "refs/heads/foo", r.Update.Ref)
	assert.Equal(t, update.FastForward, r.Update.Strategy)

	// verify the branch ref is set to the correct value
	r = cmdupdate.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo", "--strategy", "force-delete-replace"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "foo", r.Update.Path)
	assert.Equal(t, update.ForceDeleteReplace, r.Update.Strategy)
	assert.Equal(t, "", r.Update.Ref)

	r = cmdupdate.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo", "--strategy", "resource-merge"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "foo", r.Update.Path)
	assert.Equal(t, update.KResourceMerge, r.Update.Strategy)
	assert.Equal(t, "", r.Update.Ref)
}

// TestCmd_fail verifies that that command returns an error when it fails rather than exiting the process
func TestCmd_fail(t *testing.T) {
	r := cmdupdate.NewRunner("kpt")
	r.Command.SilenceErrors = true
	r.Command.SilenceUsage = true
	r.Command.SetArgs([]string{filepath.Join("not", "real", "dir")})
	err := r.Command.Execute()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "no such file or directory")
	}
}

func TestCmd_path(t *testing.T) {
	var pathPrefix string
	if runtime.GOOS == "darwin" {
		pathPrefix = "/private"
	}

	dir, err := ioutil.TempDir("", "")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	testCases := []struct {
		name                    string
		currentWD               string
		path                    string
		expectedPath            string
		expectedFullPackagePath string
		expectedErrMsg          string
	}{
		{
			name:                    "update package in current directory",
			currentWD:               dir,
			path:                    ".",
			expectedPath:            ".",
			expectedFullPackagePath: filepath.Join(pathPrefix, dir),
		},
		{
			name:                    "update package in subfolder of current directory",
			currentWD:               filepath.Dir(dir),
			path:                    filepath.Base(dir),
			expectedPath:            filepath.Base(dir),
			expectedFullPackagePath: filepath.Join(pathPrefix, dir),
		},
		{
			name:                    "update package with full absolute path",
			currentWD:               filepath.Dir(dir),
			path:                    filepath.Join(pathPrefix, dir),
			expectedPath:            filepath.Base(dir),
			expectedFullPackagePath: filepath.Join(pathPrefix, dir),
		},
		{
			name:           "package must exist as a subdirectory of cwd",
			currentWD:      filepath.Dir(dir),
			path:           "/var/user/temp",
			expectedErrMsg: "package path must be under current working directory",
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			currentWD, err := os.Getwd()
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer func() {
				_ = os.Chdir(currentWD)
			}()
			err = os.Chdir(test.currentWD)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			r := cmdupdate.NewRunner("kpt")
			r.Command.RunE = func(cmd *cobra.Command, args []string) error {
				if !assert.Equal(t, test.expectedPath, r.Update.Path) {
					t.FailNow()
				}
				if !assert.Equal(t, test.expectedFullPackagePath, r.Update.FullPackagePath) {
					t.FailNow()
				}
				return nil
			}
			r.Command.SetArgs([]string{test.path})
			err = r.Command.Execute()

			if test.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), test.expectedErrMsg)
				return
			}

			if !assert.NoError(t, err) {
				t.FailNow()
			}
		})
	}
}
