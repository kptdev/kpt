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
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/cmdupdate"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

// TestCmd_execute verifies that update is correctly invoked.
func TestCmd_execute(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// clone the repo
	getCmd := cmdget.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	getCmd.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git", w.WorkspaceDirectory})
	err := getCmd.Command.Execute()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest, true) {
		return
	}
	gitRunner, err := gitutil.NewLocalGitRunner(w.WorkspaceDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = gitRunner.Run(context.Background(), "add", ".")
	if !assert.NoError(t, err) {
		return
	}
	_, err = gitRunner.Run(context.Background(), "commit", "-m", "commit local package -- ds1")
	if !assert.NoError(t, err) {
		return
	}

	// update the master branch
	if !assert.NoError(t, g.ReplaceData(testutil.Dataset2)) {
		return
	}
	_, err = g.Commit("modify upstream package -- ds2")
	if !assert.NoError(t, err) {
		return
	}

	// update the cloned package
	updateCmd := cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	updateCmd.Command.SetArgs([]string{g.RepoName, "--strategy", "fast-forward"})
	if !assert.NoError(t, updateCmd.Command.Execute()) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), dest, true) {
		return
	}

	commit, err := g.GetCommit()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertKptfile(t, dest, kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.TypeMeta.APIVersion,
				Kind:       kptfilev1alpha2.TypeMeta.Kind},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Directory: "/",
			},
			UpdateStrategy: kptfilev1alpha2.FastForward,
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.GitLock{
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
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// clone the repo
	getCmd := cmdget.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	getCmd.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git", w.WorkspaceDirectory})
	err := getCmd.Command.Execute()
	if !assert.NoError(t, err) {
		return
	}
	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest, true) {
		return
	}

	// update the master branch
	if !assert.NoError(t, g.ReplaceData(testutil.Dataset2)) {
		return
	}

	_, err = g.Commit("new dataset")
	if !assert.NoError(t, err) {
		return
	}

	// update the cloned package
	updateCmd := cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	updateCmd.Command.SetArgs([]string{g.RepoName})
	err = updateCmd.Command.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "package must be committed to git before attempting to update")

	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest, true) {
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
	r := cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{})
	err := r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", r.Update.Ref)
	assert.Equal(t, kptfilev1alpha2.ResourceMerge, r.Update.Strategy)

	// verify an error is thrown if multiple paths are specified
	r = cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	r.Command.SilenceErrors = true
	r.Command.RunE = failRun
	r.Command.SetArgs([]string{"foo", "bar"})
	err = r.Command.Execute()
	assert.EqualError(t, err, "accepts at most 1 arg(s), received 2")
	assert.Equal(t, "", r.Update.Ref)
	assert.Equal(t, kptfilev1alpha2.UpdateStrategyType(""), r.Update.Strategy)

	// verify the branch ref is set to the correct value
	r = cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo@refs/heads/foo"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "refs/heads/foo", r.Update.Ref)
	assert.Equal(t, kptfilev1alpha2.ResourceMerge, r.Update.Strategy)

	// verify the branch ref is set to the correct value
	r = cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo", "--strategy", "force-delete-replace"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, kptfilev1alpha2.ForceDeleteReplace, r.Update.Strategy)
	assert.Equal(t, "", r.Update.Ref)

	r = cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo", "--strategy", "resource-merge"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, kptfilev1alpha2.ResourceMerge, r.Update.Strategy)
	assert.Equal(t, "", r.Update.Ref)
}

// TestCmd_fail verifies that that command returns an error when it fails rather than exiting the process
func TestCmd_fail(t *testing.T) {
	r := cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
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
			defer testutil.Chdir(t, test.currentWD)()

			r := cmdupdate.NewRunner(fake.CtxWithNilPrinter(), "kpt")
			r.Command.RunE = func(cmd *cobra.Command, args []string) error {
				if !assert.Equal(t, test.expectedFullPackagePath, r.Update.Pkg.UniquePath.String()) {
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
