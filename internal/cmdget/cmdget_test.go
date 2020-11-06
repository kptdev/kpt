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

package cmdget_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestCmd_execute tests that get is correctly invoked.
func TestCmd_execute(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()
	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	r := cmdget.NewRunner("kpt")
	r.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git/", "./"})
	err := r.Command.Execute()

	assert.NoError(t, err)
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest)

	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, dest, kptfile.KptFile{
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()
	dest := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := g.CheckoutBranch("main", false)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = g.DeleteBranch("master")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	r := cmdget.NewRunner("kpt")
	r.Command.SetArgs([]string{"file://" + g.RepoDirectory + ".git/", "./"})
	err = r.Command.Execute()

	assert.NoError(t, err)
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest)

	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, dest, kptfile.KptFile{
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
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "main",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})

}

func TestCmd_stdin(t *testing.T) {
	d, err := ioutil.TempDir("", "kpt")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(d)
	b := bytes.NewBufferString(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
`)

	r := cmdget.NewRunner("kpt")
	r.Command.SetIn(b)
	r.Command.SetArgs([]string{"-", d, "--pattern", "%k.yaml"})
	err = r.Command.Execute()
	if !assert.NoError(t, err) {
		return
	}
	actual, err := ioutil.ReadFile(filepath.Join(d, "deployment.yaml"))
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
`, string(actual))
}

// TestCmd_fail verifies that that command returns an error rather than exiting the process
func TestCmd_fail(t *testing.T) {
	r := cmdget.NewRunner("kpt")
	r.Command.SilenceErrors = true
	r.Command.SilenceUsage = true
	r.Command.SetArgs([]string{"file://" + filepath.Join("not", "real", "dir") + ".git/@master", "./"})
	err := r.Command.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "failed to lookup master(or main) branch")
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
	gitutil.DefaultRef = func(repo string) (string, error) {
		return "master", nil
	}

	r := cmdget.NewRunner("kpt")
	r.Command.SilenceErrors = true
	r.Command.SilenceUsage = true
	r.Command.RunE = failRun
	r.Command.SetArgs([]string{})
	err := r.Command.Execute()
	assert.EqualError(t, err, "accepts 2 arg(s), received 0")

	r = cmdget.NewRunner("kpt")
	r.Command.SilenceErrors = true
	r.Command.SilenceUsage = true
	r.Command.RunE = failRun
	r.Command.SetArgs([]string{"foo", "bar", "baz"})
	err = r.Command.Execute()
	assert.EqualError(t, err, "accepts 2 arg(s), received 3")

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"something://foo.git/@master", "./"})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, "something://foo", r.Get.Repo)
	assert.Equal(t, "foo", r.Get.Destination)

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"file://foo.git/blueprints/java", "."})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "file://foo", r.Get.Repo)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, "/blueprints/java", r.Get.Directory)
	assert.Equal(t, "java", r.Get.Destination)

	// current working dir -- should use package name
	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://foo.git/blueprints/java", "foo/../bar/../"})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://foo", r.Get.Repo)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, "/blueprints/java", r.Get.Directory)
	assert.Equal(t, "java", r.Get.Destination)

	// current working dir -- should use package name
	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://foo.git/blueprints/java", "./foo/../bar/../"})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://foo", r.Get.Repo)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, "/blueprints/java", r.Get.Directory)
	assert.Equal(t, "java", r.Get.Destination)

	// clean relative path
	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://foo.git/blueprints/java", "./foo/../bar/../baz"})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://foo", r.Get.Repo)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, "/blueprints/java", r.Get.Directory)
	assert.Equal(t, "baz", r.Get.Destination)

	// clean absolute path
	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://foo.git/blueprints/java", "/foo/../bar/../baz"})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://foo", r.Get.Repo)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, "/blueprints/java", r.Get.Directory)
	assert.Equal(t, "/baz", r.Get.Destination)

	d, err := ioutil.TempDir("", "kpt")
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	err = os.Mkdir(filepath.Join(d, "package"), 0700)
	assert.NoError(t, err)

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://foo.git", filepath.Join(d, "package", "my-app")})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://foo", r.Get.Repo)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, filepath.Join(d, "package", "my-app"), r.Get.Destination)

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://github.com/foo/bar.git/baz", filepath.Join(d, "package", "my-app")})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://github.com/foo/bar", r.Get.Repo)
	assert.Equal(t, "/baz", r.Get.Directory)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, filepath.Join(d, "package", "my-app"), r.Get.Destination)

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://github.com/foo/bar/.git/baz", filepath.Join(d, "package", "my-app")})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://github.com/foo/bar", r.Get.Repo)
	assert.Equal(t, "/baz", r.Get.Directory)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, filepath.Join(d, "package", "my-app"), r.Get.Destination)

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://github.com/foo/bar.git/baz@v1", filepath.Join(d, "package", "my-app")})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://github.com/foo/bar", r.Get.Repo)
	assert.Equal(t, "/baz", r.Get.Directory)
	assert.Equal(t, "v1", r.Get.Ref)
	assert.Equal(t, filepath.Join(d, "package", "my-app"), r.Get.Destination)

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"https://foo.git", filepath.Join(d, "package")})
	assert.NoError(t, r.Command.Execute())
	assert.Equal(t, "https://foo", r.Get.Repo)
	assert.Equal(t, "master", r.Get.Ref)
	assert.Equal(t, filepath.Join(d, "package", "foo"), r.Get.Destination)

	r = cmdget.NewRunner("kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"/", filepath.Join(d, "package", "my-app")})
	err = r.Command.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "specify '.git'")
}
