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

package get_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoogleContainerTools/kpt/commands/pkg/get"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

// TestCmd_execute tests that get is correctly invoked.
func TestCmd_execute(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	r := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	// defaults LOCAL_DEST_DIR to current working directory
	r.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git/"})
	err := r.Command.Execute()

	assert.NoError(t, err)

	// verify the cloned contents matches the repository with merge comment added
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest, true)

	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, dest, kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.KptFileGVK().GroupVersion().String(),
				Kind:       kptfilev1.KptFileGVK().Kind,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCmdMainBranch_execute tests that get is correctly invoked if default branch
// is main and master branch doesn't exist
func TestCmdMainBranch_execute(t *testing.T) {
	// set up git repository with master and main branches
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data: testutil.Dataset1,
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := g.CheckoutBranch("main", false)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	r := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git/", "./"})
	err = r.Command.Execute()

	assert.NoError(t, err)

	// verify the cloned contents matches the repository with merge comment added
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest, true)

	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, dest, kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.KptFileGVK().GroupVersion().String(),
				Kind:       kptfilev1.KptFileGVK().Kind,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "main",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "main",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCmd_fail verifies that that command returns an error rather than exiting the process
func TestCmd_fail(t *testing.T) {
	r := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.SilenceErrors = true
	r.Command.SilenceUsage = true
	r.Command.SetArgs([]string{"file://" + filepath.Join("not", "real", "dir") + ".git/@master", "./"})

	defer os.RemoveAll("dir")

	err := r.Command.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "'/real/dir' does not appear to be a git repository")
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
	var pathPrefix string
	if runtime.GOOS == "darwin" {
		pathPrefix = "/private"
	}

	_, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	failRun := NoOpFailRunE{t: t}.runE

	testCases := map[string]struct {
		argsFunc    func(repo, dir string) []string
		runE        func(*cobra.Command, []string) error
		validations func(repo, dir string, r *get.Runner, err error)
	}{
		"must have at least 1 arg": {
			argsFunc: func(repo, _ string) []string {
				return []string{}
			},
			runE: failRun,
			validations: func(_, _ string, r *get.Runner, err error) {
				assert.EqualError(t, err, "requires at least 1 arg(s), only received 0")
			},
		},
		"must provide unambiguous repo, dir and version": {
			argsFunc: func(repo, _ string) []string {
				return []string{"foo", "bar", "baz"}
			},
			runE: failRun,
			validations: func(_, _ string, r *get.Runner, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "ambiguous repo/dir@version specify '.git' in argument")
			},
		},
		"repo arg is split up correctly into ref and repo": {
			argsFunc: func(repo, _ string) []string {
				return []string{"something://foo.git/@master", "./"}
			},
			runE: NoOpRunE,
			validations: func(_, _ string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, "something://foo", r.Get.Git.Repo)
				assert.Equal(t, filepath.Join(pathPrefix, w.WorkspaceDirectory, "foo"), r.Get.Destination)
			},
		},
		"repo arg is split up correctly into ref, directory and repo": {
			argsFunc: func(repo, _ string) []string {
				return []string{fmt.Sprintf("file://%s.git/blueprints/java", repo), "."}
			},
			runE: NoOpRunE,
			validations: func(repo, _ string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, "/blueprints/java", r.Get.Git.Directory)
				assert.Equal(t, filepath.Join(pathPrefix, w.WorkspaceDirectory, "java"), r.Get.Destination)
			},
		},
		"current working dir -- should use package name": {
			argsFunc: func(repo, _ string) []string {
				return []string{fmt.Sprintf("file://%s.git/blueprints/java", repo), "foo/../bar/../"}
			},
			runE: NoOpRunE,
			validations: func(repo, _ string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, "/blueprints/java", r.Get.Git.Directory)
				assert.Equal(t, filepath.Join(pathPrefix, w.WorkspaceDirectory, "java"), r.Get.Destination)
			},
		},
		"clean relative path": {
			argsFunc: func(repo, _ string) []string {
				return []string{fmt.Sprintf("file://%s.git/blueprints/java", repo), "./foo/../bar/../baz"}
			},
			runE: NoOpRunE,
			validations: func(repo, _ string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, "/blueprints/java", r.Get.Git.Directory)
				assert.Equal(t, filepath.Join(pathPrefix, w.WorkspaceDirectory, "baz"), r.Get.Destination)
			},
		},
		"clean absolute path": {
			argsFunc: func(repo, _ string) []string {
				return []string{fmt.Sprintf("file://%s.git/blueprints/java", repo), "/foo/../bar/../baz"}
			},
			runE: NoOpRunE,
			validations: func(repo, _ string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, "/blueprints/java", r.Get.Git.Directory)
				assert.Equal(t, "/baz", r.Get.Destination)
			},
		},
		"provide an absolute destination directory": {
			argsFunc: func(repo, dir string) []string {
				return []string{fmt.Sprintf("file://%s.git", repo), filepath.Join(dir, "my-app")}
			},
			runE: NoOpRunE,
			validations: func(repo, dir string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, filepath.Join(dir, "my-app"), r.Get.Destination)
			},
		},
		"package in a subdirectory": {
			argsFunc: func(repo, dir string) []string {
				return []string{fmt.Sprintf("file://%s.git/baz", repo), filepath.Join(dir, "my-app")}
			},
			runE: NoOpRunE,
			validations: func(repo, dir string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "/baz", r.Get.Git.Directory)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, filepath.Join(dir, "my-app"), r.Get.Destination)
			},
		},
		"package in a subdirectory at a specific ref": {
			argsFunc: func(repo, dir string) []string {
				return []string{fmt.Sprintf("file://%s.git/baz@v1", repo), filepath.Join(dir, "my-app")}
			},
			runE: NoOpRunE,
			validations: func(repo, dir string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "/baz", r.Get.Git.Directory)
				assert.Equal(t, "v1", r.Get.Git.Ref)
				assert.Equal(t, filepath.Join(dir, "my-app"), r.Get.Destination)
			},
		},
		"provided directory already exists": {
			argsFunc: func(repo, dir string) []string {
				return []string{fmt.Sprintf("file://%s.git", repo), filepath.Join(dir, "package")}
			},
			runE: NoOpRunE,
			validations: func(repo, dir string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, filepath.Join(dir, "package"), r.Get.Destination)
			},
		},
		"invalid repo": {
			argsFunc: func(repo, dir string) []string {
				return []string{"/", filepath.Join(dir, "package", "my-app")}
			},
			runE: failRun,
			validations: func(repo, dir string, r *get.Runner, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "specify '.git'")
			},
		},
		"valid strategy provided": {
			argsFunc: func(repo, dir string) []string {
				return []string{fmt.Sprintf("file://%s.git", repo), filepath.Join(dir, "package"), "--strategy=fast-forward"}
			},
			runE: NoOpRunE,
			validations: func(repo, dir string, r *get.Runner, err error) {
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("file://%s", repo), r.Get.Git.Repo)
				assert.Equal(t, "master", r.Get.Git.Ref)
				assert.Equal(t, filepath.Join(dir, "package"), r.Get.Destination)
				assert.Equal(t, kptfilev1.FastForward, r.Get.UpdateStrategy)
			},
		},
		"invalid strategy provided": {
			argsFunc: func(repo, dir string) []string {
				return []string{fmt.Sprintf("file://%s.git", repo), filepath.Join(dir, "package"), "--strategy=does-not-exist"}
			},
			runE: failRun,
			validations: func(repo, dir string, r *get.Runner, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown update strategy \"does-not-exist\"")
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			})
			defer clean()

			r := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
			r.Command.SilenceErrors = true
			r.Command.SilenceUsage = true
			r.Command.RunE = tc.runE
			r.Command.SetArgs(tc.argsFunc(g.RepoDirectory, w.WorkspaceDirectory))
			err := r.Command.Execute()
			tc.validations(g.RepoDirectory, w.WorkspaceDirectory, r, err)
		})
	}
}

func TestCmd_flagAndArgParsing_Symlink(t *testing.T) {
	dir := t.TempDir()
	defer testutil.Chdir(t, dir)()

	err := os.MkdirAll(filepath.Join(dir, "path", "to", "pkg", "dir"), 0700)
	assert.NoError(t, err)
	err = os.Symlink(filepath.Join("path", "to", "pkg", "dir"), "link")
	assert.NoError(t, err)

	r := get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"file://foo.git" + "@refs/heads/foo", "link"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(cwd, "path", "to", "pkg", "dir", "foo"), r.Get.Destination)

	// make the link broken by deleting the dir
	err = os.RemoveAll(filepath.Join("path", "to", "pkg", "dir"))
	assert.NoError(t, err)
	r = get.NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"file://foo.git" + "@refs/heads/foo", "link"})
	err = r.Command.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}
