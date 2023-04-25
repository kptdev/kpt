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

package update_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleContainerTools/kpt/commands/pkg/get"
	"github.com/GoogleContainerTools/kpt/commands/pkg/update"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
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
	getCmd := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
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
	_, err = gitRunner.Run(fake.CtxWithDefaultPrinter(), "add", ".")
	if !assert.NoError(t, err) {
		return
	}
	_, err = gitRunner.Run(fake.CtxWithDefaultPrinter(), "commit", "-m", "commit local package -- ds1")
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
	updateCmd := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
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
	if !g.AssertKptfile(t, dest, kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.TypeMeta.APIVersion,
				Kind:       kptfilev1.TypeMeta.Kind},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Directory: "/",
			},
			UpdateStrategy: kptfilev1.FastForward,
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
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

func TestCmd_successUnCommitted(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// clone the repo
	getCmd := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
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

	// commit the upstream but not the local
	_, err = g.Commit("new dataset")
	if !assert.NoError(t, err) {
		return
	}

	// update the cloned package
	updateCmd := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	updateCmd.Command.SetArgs([]string{g.RepoName})
	err = updateCmd.Command.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), dest, true) {
		return
	}
}

func TestCmd_successNoGit(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	err := os.RemoveAll(".git")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// clone the repo
	getCmd := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	getCmd.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git", w.WorkspaceDirectory})
	err = getCmd.Command.Execute()
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

	// commit the upstream but not the local
	_, err = g.Commit("new dataset")
	if !assert.NoError(t, err) {
		return
	}

	// update the cloned package
	updateCmd := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	updateCmd.Command.SetArgs([]string{g.RepoName})
	err = updateCmd.Command.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), dest, true) {
		return
	}
}

func TestCmd_onlyVersionAsInput(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	err := os.RemoveAll(".git")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// clone the repo
	getCmd := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	getCmd.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git", w.WorkspaceDirectory})
	err = getCmd.Command.Execute()
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

	// commit the upstream but not the local
	_, err = g.Commit("new dataset")
	if !assert.NoError(t, err) {
		return
	}

	// update the cloned package
	updateCmd := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	defer testutil.Chdir(t, dest)()
	updateCmd.Command.SetArgs([]string{"@master"})
	err = updateCmd.Command.Execute()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), dest, true) {
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

func (t NoOpFailRunE) runE(_ *cobra.Command, _ []string) error {
	assert.Fail(t.t, "run should not be called")
	return nil
}

// TestCmd_Execute_flagAndArgParsing verifies that the flags and args are parsed into the correct Command fields
func TestCmd_Execute_flagAndArgParsing(t *testing.T) {
	failRun := NoOpFailRunE{t: t}.runE

	dir := t.TempDir()
	defer testutil.Chdir(t, filepath.Dir(dir))()

	// verify the current working directory is used if no path is specified
	r := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{})
	err := r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "", r.Update.Ref)
	assert.Equal(t, kptfilev1.ResourceMerge, r.Update.Strategy)

	// verify an error is thrown if multiple paths are specified
	r = update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.SilenceErrors = true
	r.Command.RunE = failRun
	r.Command.SetArgs([]string{"foo", "bar"})
	err = r.Command.Execute()
	assert.EqualError(t, err, "accepts at most 1 arg(s), received 2")
	assert.Equal(t, "", r.Update.Ref)
	assert.Equal(t, kptfilev1.UpdateStrategyType(""), r.Update.Strategy)

	// verify the branch ref is set to the correct value
	r = update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{dir + "@refs/heads/foo"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "refs/heads/foo", r.Update.Ref)
	assert.Equal(t, kptfilev1.ResourceMerge, r.Update.Strategy)

	// verify the branch ref is set to the correct value
	r = update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{dir, "--strategy", "force-delete-replace"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, kptfilev1.ForceDeleteReplace, r.Update.Strategy)
	assert.Equal(t, "", r.Update.Ref)

	r = update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{dir, "--strategy", "resource-merge"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, kptfilev1.ResourceMerge, r.Update.Strategy)
	assert.Equal(t, "", r.Update.Ref)
}

func TestCmd_flagAndArgParsing_Symlink(t *testing.T) {
	dir := t.TempDir()
	defer testutil.Chdir(t, dir)()

	err := os.MkdirAll(filepath.Join(dir, "path", "to", "pkg", "dir"), 0700)
	assert.NoError(t, err)
	err = os.Symlink(filepath.Join("path", "to", "pkg", "dir"), "foo")
	assert.NoError(t, err)

	// verify the branch ref is set to the correct value
	r := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo" + "@refs/heads/foo"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "refs/heads/foo", r.Update.Ref)
	assert.Equal(t, kptfilev1.ResourceMerge, r.Update.Strategy)
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(cwd, "path", "to", "pkg", "dir"), r.Update.Pkg.UniquePath.String())
}

// TestCmd_fail verifies that that command returns an error when it fails rather than exiting the process
func TestCmd_fail(t *testing.T) {
	r := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
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

	dir := t.TempDir()

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
			path:           filepath.Dir(filepath.Dir(dir)),
			expectedErrMsg: "package path must be under current working directory",
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			defer testutil.Chdir(t, test.currentWD)()

			r := update.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
			r.Command.RunE = func(cmd *cobra.Command, args []string) error {
				if !assert.Equal(t, test.expectedFullPackagePath, r.Update.Pkg.UniquePath.String()) {
					t.FailNow()
				}
				return nil
			}

			r.Command.SetArgs([]string{test.path})
			err := r.Command.Execute()

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

func TestCmd_output(t *testing.T) {
	testCases := map[string]struct {
		reposChanges   map[string][]testutil.Content
		updatedLocal   testutil.Content
		expectedLocal  *pkgbuilder.RootPkg
		expectedOutput string
	}{
		"basic package": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "master",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.SecretResource),
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", "master", 1),
				).
				WithResource(pkgbuilder.SecretResource),
			expectedOutput: `
Package "{{ .PKG_NAME }}":
Fetching upstream from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Fetching origin from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Updating package "{{ .PKG_NAME }}" with strategy "resource-merge".

Updated 1 package(s).
`,
		},
		"nested packages": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "master", "fast-forward").
											WithUpstreamLockRef("foo", "/", "master", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: "master",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.SecretResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "master", "fast-forward").
											WithUpstreamLockRef("foo", "/", "master", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "master",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", "master", 1),
				).
				WithResource(pkgbuilder.SecretResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "master", "fast-forward").
								WithUpstreamLockRef("foo", "/", "master", 1),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
			expectedOutput: `
Package "{{ .PKG_NAME }}":
Fetching upstream from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Fetching origin from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Updating package "{{ .PKG_NAME }}" with strategy "resource-merge".
Updating package "subpkg" with strategy "fast-forward".

Package "{{ .PKG_NAME }}/subpkg":
Fetching upstream from {{ (index .REPOS "foo").RepoDirectory }}@master
<git_output>
Fetching origin from {{ (index .REPOS "foo").RepoDirectory }}@master
<git_output>
Updating package "subpkg" with strategy "fast-forward".

Updated 2 package(s).
`,
		},
		"subpackage deleted from upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg1").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "master", "resource-merge").
											WithUpstreamLockRef("foo", "/", "master", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
								pkgbuilder.NewSubPkg("subpkg2").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "master", "resource-merge").
											WithUpstreamLockRef("foo", "/", "master", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: "master",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.SecretResource),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "master",
					},

					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", "master", 0),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg1").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "master", "resource-merge").
									WithUpstreamLockRef("foo", "/", "master", 0),
							).
							WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("5", "spec", "replicas")),
						pkgbuilder.NewSubPkg("subpkg2").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "master", "resource-merge").
									WithUpstreamLockRef("foo", "/", "master", 0),
							).
							WithResource(pkgbuilder.DeploymentResource),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", "master", 1),
				).
				WithResource(pkgbuilder.SecretResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg1").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "master", "resource-merge").
								WithUpstreamLockRef("foo", "/", "master", 1),
						).
						WithResource(pkgbuilder.ConfigMapResource).
						WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("5", "spec", "replicas")),
				),
			expectedOutput: `
Package "{{ .PKG_NAME }}":
Fetching upstream from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Fetching origin from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Updating package "{{ .PKG_NAME }}" with strategy "resource-merge".
Deleting package "subpkg2" from local since it is removed in upstream.
Package "subpkg1" deleted from upstream, but keeping local since it has changes.

Package "{{ .PKG_NAME }}/subpkg1":
Fetching upstream from {{ (index .REPOS "foo").RepoDirectory }}@master
<git_output>
Fetching origin from {{ (index .REPOS "foo").RepoDirectory }}@master
<git_output>
Updating package "subpkg1" with strategy "resource-merge".

Updated 2 package(s).
`,
		},
		"Adding package in upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "master",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.SecretResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1", "force-delete-replace"),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "master",
						Tag:    "v1",
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", "master", 1),
				).
				WithResource(pkgbuilder.SecretResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v1", "force-delete-replace").
								WithUpstreamLockRef("foo", "/", "v1", 0),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			expectedOutput: `
Package "{{ .PKG_NAME }}":
Fetching upstream from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Fetching origin from {{ (index .REPOS "upstream").RepoDirectory }}@master
<git_output>
Updating package "{{ .PKG_NAME }}" with strategy "resource-merge".
Adding package "subpkg" from upstream.

Package "{{ .PKG_NAME }}/subpkg":
Fetching upstream from {{ (index .REPOS "foo").RepoDirectory }}@v1
<git_output>
Updating package "subpkg" with strategy "force-delete-replace".

Updated 2 package(s).
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T:            t,
				ReposChanges: tc.reposChanges,
			}
			defer g.Clean()
			if tc.updatedLocal.Pkg != nil {
				g.LocalChanges = []testutil.Content{
					tc.updatedLocal,
				}
			}
			if !g.Init() {
				return
			}

			clean := testutil.Chdir(t, g.LocalWorkspace.FullPackagePath())
			defer clean()

			var outBuf bytes.Buffer
			var errBuf bytes.Buffer

			ctx := fake.CtxWithPrinter(&outBuf, &errBuf)
			r := update.NewRunner(ctx, "kpt")
			r.Command.SetArgs([]string{})
			err := r.Command.Execute()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.Empty(t, outBuf.String())

			tmpl := template.Must(template.New("test").Parse(tc.expectedOutput))
			var expected bytes.Buffer
			err = tmpl.Execute(&expected, map[string]interface{}{
				"PKG_PATH": g.LocalWorkspace.FullPackagePath(),
				"PKG_NAME": g.LocalWorkspace.PackageDir,
				"REPOS":    g.Repos,
			})
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			actual := scrubGitOutput(errBuf.String())

			assert.Equal(t, strings.TrimSpace(expected.String()), strings.TrimSpace(actual))

			expectedPath := tc.expectedLocal.ExpandPkgWithName(t, g.LocalWorkspace.PackageDir, testutil.ToReposInfo(g.Repos))
			testutil.KptfileAwarePkgEqual(t, expectedPath, g.LocalWorkspace.FullPackagePath(), true)
		})
	}
}

const (
	gitOutputPattern = `From \/.*(\r\n|\r|\n)( * .*(\r\n|\r|\n))+`
)

func scrubGitOutput(output string) string {
	re := regexp.MustCompile(gitOutputPattern)
	return re.ReplaceAllString(output, "<git_output>\n")
}
