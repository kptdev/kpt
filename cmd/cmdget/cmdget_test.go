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

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"kpt.dev/cmdget"
	"lib.kpt.dev/kptfile"
	"lib.kpt.dev/testutil"
	"lib.kpt.dev/yaml"
)

// TestCmd_execute tests that get is correctly invoked.
func TestCmd_execute(t *testing.T) {
	g, dir, clean := testutil.SetupDefaultRepoAndWorkspace(t)
	defer clean()
	dest := filepath.Join(dir, g.RepoName)

	r := cmdget.Cmd()
	r.C.SetArgs([]string{"file://" + g.RepoDirectory + ".git/", "./"})
	err := r.C.Execute()

	assert.NoError(t, err)
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), dest)

	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, dest, kptfile.KptFile{
		ResourceMeta: yaml.NewResourceMeta(g.RepoName, kptfile.TypeMeta),
		PackageMeta:  kptfile.PackageMeta{},
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

	r := cmdget.Cmd()
	r.C.SetIn(b)
	r.C.SetArgs([]string{"-", d, "--pattern", "%k.yaml"})
	err = r.C.Execute()
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
	r := cmdget.Cmd()
	r.C.SilenceErrors = true
	r.C.SilenceUsage = true
	r.C.SetArgs([]string{"file://" + filepath.Join("not", "real", "dir") + ".git/@master", "./"})
	err := r.C.Execute()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "failed to clone git repo: trouble fetching")
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

	r := cmdget.Cmd()
	r.C.SilenceErrors = true
	r.C.SilenceUsage = true
	r.C.RunE = failRun
	r.C.SetArgs([]string{})
	err := r.C.Execute()
	assert.EqualError(t, err, "accepts 2 arg(s), received 0")

	r = cmdget.Cmd()
	r.C.SilenceErrors = true
	r.C.SilenceUsage = true
	r.C.RunE = failRun
	r.C.SetArgs([]string{"foo", "bar", "baz"})
	err = r.C.Execute()
	assert.EqualError(t, err, "accepts 2 arg(s), received 3")

	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"something://foo.git/@master", "./"})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, "something://foo", r.Repo)
	assert.Equal(t, "foo", r.Destination)

	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"file://foo.git/blueprints/java", "."})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "file://foo", r.Repo)
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, "blueprints/java", r.Directory)
	assert.Equal(t, "java", r.Destination)

	// current working dir -- should use package name
	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"https://foo.git/blueprints/java", "foo/../bar/../"})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "https://foo", r.Repo)
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, "blueprints/java", r.Directory)
	assert.Equal(t, "java", r.Destination)

	// current working dir -- should use package name
	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"https://foo.git/blueprints/java", "./foo/../bar/../"})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "https://foo", r.Repo)
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, "blueprints/java", r.Directory)
	assert.Equal(t, "java", r.Destination)

	// clean relative path
	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"https://foo.git/blueprints/java", "./foo/../bar/../baz"})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "https://foo", r.Repo)
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, "blueprints/java", r.Directory)
	assert.Equal(t, "baz", r.Destination)

	// clean absolute path
	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"https://foo.git/blueprints/java", "/foo/../bar/../baz"})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "https://foo", r.Repo)
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, "blueprints/java", r.Directory)
	assert.Equal(t, "/baz", r.Destination)

	d, err := ioutil.TempDir("", "ktp")
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	err = os.Mkdir(filepath.Join(d, "package"), 0700)
	assert.NoError(t, err)

	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"https://foo.git", filepath.Join(d, "package", "my-app")})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "https://foo", r.Repo)
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, filepath.Join(d, "package", "my-app"), r.Destination)

	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"https://foo.git", filepath.Join(d, "package")})
	assert.NoError(t, r.C.Execute())
	assert.Equal(t, "https://foo", r.Repo)
	assert.Equal(t, "master", r.Ref)
	assert.Equal(t, filepath.Join(d, "package", "foo"), r.Destination)

	r = cmdget.Cmd()
	r.C.RunE = NoOpRunE
	r.C.SetArgs([]string{"/", filepath.Join(d, "package", "my-app")})
	err = r.C.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify the repository schema ")
}
