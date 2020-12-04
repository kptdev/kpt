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

package testutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	assertnow "gotest.tools/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const TmpDirPrefix = "test-kpt"

const (
	Dataset1              = "dataset1"
	Dataset2              = "dataset2"
	Dataset3              = "dataset3"
	Dataset4              = "dataset4" // Dataset4 is replica of Dataset2 with different setter values
	Dataset5              = "dataset5" // Dataset5 is replica of Dataset2 with additional non KRM files
	DatasetMerged         = "datasetmerged"
	DiffOutput            = "diff_output"
	UpdateMergeConflict   = "updateMergeConflict"
	HelloWorldSet         = "helloworld-set"
	HelloWorldFn          = "helloworld-fn"
	HelloWorldFnNoKptfile = "helloworld-fn-no-kptfile"
)

// TestGitRepo manages a local git repository for testing
type TestGitRepo struct {
	// RepoDirectory is the temp directory of the git repo
	RepoDirectory string

	// DatasetDirectory is the directory of the testdata files
	DatasetDirectory string

	// RepoName is the name of the repository
	RepoName string
}

var AssertNoError = assertnow.NilError

var KptfileSet = func() sets.String {
	s := sets.String{}
	s.Insert(kptfile.KptFileName)
	return s
}()

// AssertEqual verifies the contents of a source package matches the contents of the
// destination package it was fetched to.
// Excludes comparing the .git directory in the source package.
//
// While the sourceDir can be the TestGitRepo, because tests change the TestGitRepo
// may have been changed after the destDir was copied, it is often better to explicitly
// use a set of golden files as the sourceDir rather than the original TestGitRepo
// that was copied.
func (g *TestGitRepo) AssertEqual(t *testing.T, sourceDir, destDir string) bool {
	diff, err := Diff(sourceDir, destDir)
	if !assert.NoError(t, err) {
		return false
	}
	diff = diff.Difference(KptfileSet)
	return assert.Empty(t, diff.List())
}

func AssertPkgEqual(t *testing.T, g *TestGitRepo, sourceDir, destDir string) {
	diff, err := Diff(sourceDir, destDir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	diff = diff.Difference(KptfileSet)
	if !assert.Empty(t, diff.List()) {
		t.FailNow()
	}
}

// Diff returns a list of files that differ between the source and destination.
//
// Diff is guaranteed to return a non-empty set if any files differ, but
// this set is not guaranteed to contain all differing files.
func Diff(sourceDir, destDir string) (sets.String, error) {
	// get set of filenames in the package source
	upstreamFiles := sets.String{}
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip git repo if it exists
		if strings.Contains(path, ".git") {
			return nil
		}

		upstreamFiles.Insert(strings.TrimPrefix(strings.TrimPrefix(path, sourceDir), string(filepath.Separator)))
		return nil
	})
	if err != nil {
		return sets.String{}, err
	}

	// get set of filenames in the cloned package
	localFiles := sets.String{}
	err = filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip git repo if it exists
		if strings.Contains(path, ".git") {
			return nil
		}

		localFiles.Insert(strings.TrimPrefix(strings.TrimPrefix(path, destDir), string(filepath.Separator)))
		return nil
	})
	if err != nil {
		return sets.String{}, err
	}

	// verify the source and cloned packages have the same set of filenames
	diff := upstreamFiles.SymmetricDifference(localFiles)

	// verify file contents match
	for _, f := range upstreamFiles.Intersection(localFiles).List() {
		fi, err := os.Stat(filepath.Join(destDir, f))
		if err != nil {
			return diff, err
		}
		if fi.Mode().IsDir() {
			// already checked that this directory exists in the local files
			continue
		}

		// compare upstreamFiles
		b1, err := ioutil.ReadFile(filepath.Join(destDir, f))
		if err != nil {
			return diff, err
		}
		b2, err := ioutil.ReadFile(filepath.Join(sourceDir, f))
		if err != nil {
			return diff, err
		}

		s1 := strings.TrimSpace(strings.TrimPrefix(string(b1), trimPrefix))
		s2 := strings.TrimSpace(strings.TrimPrefix(string(b2), trimPrefix))

		if s1 != s2 {
			fmt.Println(copyutil.PrettyFileDiff(s1, s2))
			diff.Insert(f)
		}
	}
	// return the differing files
	return diff, nil
}

const trimPrefix = `# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.`

func Replace(t *testing.T, path, old, new string) {
	b, err := ioutil.ReadFile(path)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// update the expected contents to reflect the set command
	b = []byte(strings.ReplaceAll(string(b), old, new))
	if !assert.NoError(t, ioutil.WriteFile(path, b, 0)) {
		t.FailNow()
	}
}

func Compare(t *testing.T, a, b string) {
	// Compare parses the yaml and serializes both files to normalize
	// formatting
	b1, err := ioutil.ReadFile(a)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	n1, err := yaml.Parse(string(b1))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	s1, err := n1.String()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	b2, err := ioutil.ReadFile(b)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	n2, err := yaml.Parse(string(b2))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	s2, err := n2.String()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, s1, s2) {
		t.FailNow()
	}
}

// AssertKptfile verifies the contents of the KptFile matches the provided value.
func (g *TestGitRepo) AssertKptfile(t *testing.T, cloned string, kpkg kptfile.KptFile) bool {
	// read the actual generated KptFile
	b, err := ioutil.ReadFile(filepath.Join(cloned, kptfile.KptFileName))
	if !assert.NoError(t, err) {
		return false
	}
	actual := kptfile.KptFile{}
	d := yaml.NewDecoder(bytes.NewBuffer(b))
	d.KnownFields(true)
	if !assert.NoError(t, d.Decode(&actual)) {
		return false
	}
	return assert.Equal(t, kpkg, actual)
}

// CheckoutBranch checks out the git branch in the repo
func (g *TestGitRepo) CheckoutBranch(branch string, create bool) error {
	return checkoutBranch(g.RepoDirectory, branch, create)
}

// DeleteBranch deletes the git branch in the repo
func (g *TestGitRepo) DeleteBranch(branch string) error {
	// checkout the branch
	cmd := exec.Command("git", []string{"branch", "-D", branch}...)
	cmd.Dir = g.RepoDirectory
	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

// Commit performs a git commit
func (g *TestGitRepo) Commit(message string) error {
	return commit(g.RepoDirectory, message)
}

// Commit performs a git commit
func Commit(t *testing.T, g *TestGitRepo, message string) {
	if !assert.NoError(t, g.Commit(message)) {
		t.FailNow()
	}
}

func CommitTag(t *testing.T, g *TestGitRepo, tag string) {
	Commit(t, g, tag)
	Tag(t, g, tag)
}

func CopyData(t *testing.T, g *TestGitRepo, data, dest string) {
	if !filepath.IsAbs(data) {
		data = filepath.Join(g.DatasetDirectory, data)
	}

	dest = filepath.Join(g.RepoDirectory, dest)
	err := os.MkdirAll(dest, 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.NoError(t, copyutil.CopyDir(data, dest)) {
		t.FailNow()
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = g.RepoDirectory
	stdoutStderr, err := cmd.CombinedOutput()
	if !assert.NoError(t, err, stdoutStderr) {
		t.FailNow()
	}
}

func (g *TestGitRepo) GetCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = g.RepoDirectory
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// RemoveAll deletes the test git repo
func (g *TestGitRepo) RemoveAll() error {
	err := os.RemoveAll(g.RepoDirectory)
	return err
}

func RemoveData(t *testing.T, g *TestGitRepo) {
	// remove the old data
	files, err := ioutil.ReadDir(g.RepoDirectory)
	if err != nil {
		t.FailNow()
	}
	for i := range files {
		f := files[i]
		if f.IsDir() && f.Name() == ".git" {
			continue
		}
		err := os.RemoveAll(filepath.Join(g.RepoDirectory, f.Name()))
		if err != nil {
			t.FailNow()
		}
	}
}

// ReplaceData replaces the data with a new source
func (g *TestGitRepo) ReplaceData(data string) error {
	if !filepath.IsAbs(data) {
		data = filepath.Join(g.DatasetDirectory, data)
	}

	return replaceData(g.RepoDirectory, data)
}

// SetupTestGitRepo initializes a new git repository and populates it with data from a source
func (g *TestGitRepo) SetupTestGitRepo(data string) error {
	// configure the path to the test dataset
	ds, err := GetTestDataPath()
	if err != nil {
		return err
	}
	g.DatasetDirectory = ds

	// create the test repo directory
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-upstream-", TmpDirPrefix))
	if err != nil {
		return err
	}
	g.RepoDirectory = dir
	g.RepoName = filepath.Base(g.RepoDirectory)

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return err
	}

	if !filepath.IsAbs(data) {
		data = filepath.Join(g.DatasetDirectory, data)
	}
	// populate the repo with
	err = copyAddData(dir, data)
	if err != nil {
		return err
	}
	return g.Commit("initial commit")
}

func GetTestDataPath() (string, error) {
	filename, err := getTestUtilGoFilePath()
	if err != nil {
		return "", err
	}
	ds, err := filepath.Abs(filepath.Join(filepath.Dir(filename), "testdata"))
	if err != nil {
		return "", err
	}
	return ds, nil
}

func getTestUtilGoFilePath() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.Errorf("failed to testutil package location")
	}
	return filename, nil
}

// Tag initializes tags the git repository
func (g *TestGitRepo) Tag(tagName string) error {
	return tag(g.RepoDirectory, tagName)
}

func Tag(t *testing.T, g *TestGitRepo, tag string) {
	if !assert.NoError(t, g.Tag(tag)) {
		t.FailNow()
	}
}

func CopyKptfile(t *testing.T, src, dest string) {
	b, err := ioutil.ReadFile(filepath.Join(src, kptfile.KptFileName))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = ioutil.WriteFile(filepath.Join(dest, kptfile.KptFileName), b, 0600)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}

// SetupDefaultRepoAndWorkspace handles setting up a default repo to clone, and a workspace to clone into.
// returns a cleanup function to remove the git repo and workspace.
func SetupDefaultRepoAndWorkspace(t *testing.T, dataset string) (*TestGitRepo, *TestWorkspace, func()) {
	// Capture the current working directory so we can set it back to the
	// original path after test has completed.
	cwd, err := os.Getwd()
	if err != nil {
		assert.NoError(t, err)
	}

	// setup the repo to clone from
	g := &TestGitRepo{}
	err = g.SetupTestGitRepo(dataset)
	assert.NoError(t, err)

	// setup the directory to clone to
	w := &TestWorkspace{
		PackageDir: g.RepoName,
	}
	err = w.SetupTestWorkspace()
	assert.NoError(t, err)
	err = os.Chdir(w.WorkspaceDirectory)
	assert.NoError(t, err)

	gr := gitutil.NewLocalGitRunner("./")
	if !assert.NoError(t, gr.Run("init")) {
		assert.FailNowf(t, "%s %s", gr.Stdout.String(), gr.Stderr.String())
	}

	// make sure that both master and main branches are created in the test repo
	// do not error if they already exist or
	_ = g.CheckoutBranch("master", true)
	_ = g.CheckoutBranch("main", true)

	// checkout to master branch
	err = g.CheckoutBranch("master", false)
	assert.NoError(t, err)

	return g, w, func() {
		// ignore cleanup failures
		_ = g.RemoveAll()
		_ = w.RemoveAll()
		_ = os.Chdir(cwd)
	}
}

func checkoutBranch(repo string, branch string, create bool) error {
	var args []string
	if create {
		args = []string{"checkout", "-b", branch}
	} else {
		args = []string{"checkout", branch}
	}

	// checkout the branch
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

func replaceData(repo, data string) error {
	// If the path is absolute we assume it is the full path to the
	// testdata. If it is relative, we assume it refers to one of the
	// test data sets.
	if !filepath.IsAbs(data) {
		ds, err := GetTestDataPath()
		if err != nil {
			return err
		}
		data = filepath.Join(ds, data)
	}
	// Walk the data directory and copy over all files. We have special
	// handling of the Kptfile to make sure we don't lose the Upstream data.
	if err := filepath.Walk(data, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(data, path)
		if err != nil {
			return err
		}

		_, err = os.Stat(filepath.Join(repo, rel))
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		// If the file/directory doesn't exist in the repo folder, we just
		// copy it over.
		if os.IsNotExist(err) {
			if info.IsDir() {
				err := os.Mkdir(filepath.Join(repo, rel), 0700)
				if err != nil {
					return err
				}
			} else {
				err := copyutil.SyncFile(path, filepath.Join(repo, rel))
				if err != nil {
					return err
				}
			}
			return nil
		}

		// If it is a directory and we know it already exists, we don't need
		// to do anything.
		if info.IsDir() {
			return nil
		}

		// For Kptfiles we need to keep the Upstream section even if we replace
		// the file.
		if rel == "Kptfile" {
			dataKptfile, err := kptfileutil.ReadFile(filepath.Dir(path))
			if err != nil {
				return err
			}
			repoKptfileDir := filepath.Dir(filepath.Join(repo, rel))
			repoKptfile, err := kptfileutil.ReadFile(repoKptfileDir)
			if err != nil {
				return err
			}
			dataKptfile.Upstream = repoKptfile.Upstream
			err = kptfileutil.WriteFile(repoKptfileDir, dataKptfile)
			if err != nil {
				return err
			}
		} else {
			err := copyutil.SyncFile(path, filepath.Join(repo, rel))
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// We then walk the repo folder and remove and files/directories that
	// exists in the repo, but doesn't exist in the data directory.
	if err := filepath.Walk(repo, func(path string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		// Find the relative path of the file/directory
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		// We skip anything that is inside the .git folder
		if strings.HasPrefix(rel, ".git") {
			return nil
		}

		// Check if a file/directory exists at the path relative path within the
		// data directory
		dataCopy := filepath.Join(data, rel)
		_, err = os.Stat(dataCopy)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		// If the file/directory doesn't exist in the data folder, we remove
		// them from the repo folder.
		if os.IsNotExist(err) {
			if info.IsDir() {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
			} else {
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Add the changes to git.
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repo
	_, err := cmd.CombinedOutput()
	return err
}

func copyAddData(repo string, data string) error {
	err := copyutil.CopyDir(data, repo)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repo
	_, err = cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

func commit(repo, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repo
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return err
	}
	return nil
}

func tag(repo, tag string) error {
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = repo
	b, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", b)
		return err
	}

	return nil
}

type TestWorkspace struct {
	WorkspaceDirectory string
	PackageDir         string
}

// FullPackagePath returns the full path to the roor package in the
// local workspace.
func (w *TestWorkspace) FullPackagePath() string {
	return filepath.Join(w.WorkspaceDirectory, w.PackageDir)
}

func (w *TestWorkspace) SetupTestWorkspace() error {
	var err error
	w.WorkspaceDirectory, err = ioutil.TempDir("", "test-kpt-local-")
	return err
}

func (w *TestWorkspace) RemoveAll() error {
	return os.RemoveAll(w.WorkspaceDirectory)
}

// CheckoutBranch checks out the git branch in the repo
func (w *TestWorkspace) CheckoutBranch(branch string, create bool) error {
	return checkoutBranch(w.WorkspaceDirectory, branch, create)
}

// ReplaceData replaces the data with a new source
func (w *TestWorkspace) ReplaceData(data string) error {
	return replaceData(filepath.Join(w.WorkspaceDirectory, w.PackageDir), data)
}

// Commit performs a git commit
func (w *TestWorkspace) Commit(message string) error {
	return commit(w.WorkspaceDirectory, message)
}

// Tag initializes tags the git repository
func (w *TestWorkspace) Tag(tagName string) error {
	return tag(w.WorkspaceDirectory, tagName)
}

func PrintPackage(paths ...string) error {
	path := filepath.Join(paths...)
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, "/.git") {
			return nil
		}
		fmt.Println(path)
		return nil
	})
}

func PrintFile(paths ...string) error {
	path := filepath.Join(paths...)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
