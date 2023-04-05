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

package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/addmergecomment"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	toposort "github.com/philopon/go-toposort"
	"github.com/stretchr/testify/assert"
	assertnow "gotest.tools/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const TmpDirPrefix = "test-kpt"

const (
	Dataset1            = "dataset1"
	Dataset2            = "dataset2"
	Dataset3            = "dataset3"
	Dataset4            = "dataset4" // Dataset4 is replica of Dataset2 with different setter values
	Dataset5            = "dataset5" // Dataset5 is replica of Dataset2 with additional non KRM files
	Dataset6            = "dataset6" // Dataset6 contains symlinks
	DatasetMerged       = "datasetmerged"
	DiffOutput          = "diff_output"
	UpdateMergeConflict = "updateMergeConflict"
)

// TestGitRepo manages a local git repository for testing
type TestGitRepo struct {
	T *testing.T

	// RepoDirectory is the temp directory of the git repo
	RepoDirectory string

	// DatasetDirectory is the directory of the testdata files
	DatasetDirectory string

	// RepoName is the name of the repository
	RepoName string

	// Commits keeps track of the commit shas for the changes
	// to the repo.
	Commits []string
}

var AssertNoError = assertnow.NilError

var KptfileSet = diffSet(kptfilev1.KptFileName)

func diffSet(path string) sets.String {
	s := sets.String{}
	s.Insert(path)
	return s
}

// AssertEqual verifies the contents of a source package matches the contents of the
// destination package it was fetched to.
// Excludes comparing the .git directory in the source package.
//
// While the sourceDir can be the TestGitRepo, because tests change the TestGitRepo
// may have been changed after the destDir was copied, it is often better to explicitly
// use a set of golden files as the sourceDir rather than the original TestGitRepo
// that was copied.
func (g *TestGitRepo) AssertEqual(t *testing.T, sourceDir, destDir string, addMergeCommentsToSource bool) bool {
	diff, err := Diff(sourceDir, destDir, addMergeCommentsToSource)
	if !assert.NoError(t, err) {
		return false
	}
	diff = diff.Difference(KptfileSet)
	return assert.Empty(t, diff.List())
}

// KptfileAwarePkgEqual compares two packages (including any subpackages)
// and has special handling of Kptfiles to handle fields that contain
// values which cannot easily be specified in the golden package.
func KptfileAwarePkgEqual(t *testing.T, pkg1, pkg2 string, addMergeCommentsToSource bool) bool {
	diff, err := Diff(pkg1, pkg2, addMergeCommentsToSource)
	if !assert.NoError(t, err) {
		return false
	}

	// TODO(mortent): See if we can avoid this. We just need to make sure
	// we can compare Kptfiles without any formatting issues.
	for _, s := range diff.List() {
		if !strings.HasSuffix(s, kptfilev1.KptFileName) {
			continue
		}

		pkg1Path := filepath.Join(pkg1, s)
		pkg1KfExists := kptfileExists(t, pkg1Path)

		pkg2Path := filepath.Join(pkg2, s)
		pkg2KfExists := kptfileExists(t, pkg2Path)

		if !pkg1KfExists || !pkg2KfExists {
			continue
		}

		// Read the Kptfiles and set the Commit field to an empty
		// string before we compare.
		pkg1kf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, filepath.Dir(pkg1Path))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		pkg2kf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, filepath.Dir(pkg2Path))
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		equal, err := kptfileutil.Equal(pkg1kf, pkg2kf)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		// If the two files are considered equal after we have compared
		// them with Kptfile-specific rules, we remove the path from the
		// diff set.
		if equal {
			diff = diff.Difference(diffSet(s))
		}
	}
	return assert.Empty(t, diff.List())
}

func kptfileExists(t *testing.T, path string) bool {
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		assert.NoError(t, err)
		t.FailNow()
	}
	return !os.IsNotExist(err)
}

// Diff returns a list of files that differ between the source and destination.
//
// Diff is guaranteed to return a non-empty set if any files differ, but
// this set is not guaranteed to contain all differing files.
func Diff(sourceDir, destDir string, addMergeCommentsToSource bool) (sets.String, error) {
	// get set of filenames in the package source
	var newSourceDir string
	if addMergeCommentsToSource {
		dir, clean, err := addmergecomment.ProcessWithCleanup(sourceDir)
		defer clean()
		if err != nil {
			return sets.String{}, err
		}
		newSourceDir = dir
	}
	if newSourceDir != "" {
		sourceDir = newSourceDir
	}
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
		b1, err := os.ReadFile(filepath.Join(destDir, f))
		if err != nil {
			return diff, err
		}
		b2, err := os.ReadFile(filepath.Join(sourceDir, f))
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

const trimPrefix = `# Copyright 2019 The kpt Authors
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

// AssertKptfile verifies the contents of the KptFile matches the provided value.
func (g *TestGitRepo) AssertKptfile(t *testing.T, cloned string, kpkg kptfilev1.KptFile) bool {
	// read the actual generated KptFile
	b, err := os.ReadFile(filepath.Join(cloned, kptfilev1.KptFileName))
	if !assert.NoError(t, err) {
		return false
	}
	var res bytes.Buffer
	d := yaml.NewEncoder(&res)
	if !assert.NoError(t, d.Encode(kpkg)) {
		return false
	}
	return assert.Equal(t, res.String(), string(b))
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

// Commit performs a git commit and returns the SHA for the newly
// created commit.
func (g *TestGitRepo) Commit(message string) (string, error) {
	return commit(g.RepoDirectory, message)
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

// ReplaceData replaces the data with a new source
func (g *TestGitRepo) ReplaceData(data string) error {
	if !filepath.IsAbs(data) {
		data = filepath.Join(g.DatasetDirectory, data)
	}

	return replaceData(g.RepoDirectory, data)
}

// CustomUpdate executes the provided update function and passes in the
// path to the directory of the repository.
func (g *TestGitRepo) CustomUpdate(f func(string) error) error {
	return f(g.RepoDirectory)
}

// SetupTestGitRepo initializes a new git repository and populates it with data from a source
func (g *TestGitRepo) SetupTestGitRepo(name string, data []Content, repos map[string]*TestGitRepo) error {
	defaultBranch := "main"
	if len(data) > 0 && len(data[0].Branch) > 0 {
		defaultBranch = data[0].Branch
	}

	err := g.createEmptyGitRepo(defaultBranch)
	if err != nil {
		return err
	}

	// configure the path to the test dataset
	ds, err := GetTestDataPath()
	if err != nil {
		return err
	}
	g.DatasetDirectory = ds

	return UpdateGitDir(g.T, name, g, data, repos)
}

func (g *TestGitRepo) createEmptyGitRepo(defaultBranch string) error {
	dir, err := os.MkdirTemp("", fmt.Sprintf("%s-upstream-", TmpDirPrefix))
	if err != nil {
		return err
	}
	g.RepoDirectory = dir
	g.RepoName = filepath.Base(g.RepoDirectory)

	cmd := exec.Command("git", "init",
		fmt.Sprintf("--initial-branch=%s", defaultBranch))
	cmd.Dir = dir
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return err
	}
	_, err = g.Commit("initial commit")
	return err
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

// SetupRepoAndWorkspace handles setting up a default repo to clone, and a workspace to clone into.
// returns a cleanup function to remove the git repo and workspace.
func SetupRepoAndWorkspace(t *testing.T, content Content) (*TestGitRepo, *TestWorkspace, func()) {
	repos, workspace, cleanup := SetupReposAndWorkspace(t, map[string][]Content{
		Upstream: {
			content,
		},
	})

	g := repos[Upstream]
	return g, workspace, cleanup
}

// SetupReposAndWorkspace handles setting up a set of repos as specified by
// the reposContent and a workspace to clone into. It returns a cleanup function
// that will remove the repos.
func SetupReposAndWorkspace(t *testing.T, reposContent map[string][]Content) (map[string]*TestGitRepo, *TestWorkspace, func()) {
	repos, repoCleanup := SetupRepos(t, reposContent)
	w, workspaceCleanup := SetupWorkspace(t)
	return repos, w, func() {
		repoCleanup()
		workspaceCleanup()
	}
}

// SetupWorkspace creates a local workspace which kpt packages can be cloned
// into. It returns a cleanup function that will remove the workspace.
func SetupWorkspace(t *testing.T) (*TestWorkspace, func()) {
	// setup the directory to clone to
	w := &TestWorkspace{}
	err := w.SetupTestWorkspace()
	assert.NoError(t, err)

	gr, err := gitutil.NewLocalGitRunner(w.WorkspaceDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	rr, err := gr.Run(fake.CtxWithDefaultPrinter(), "init")
	if !assert.NoError(t, err) {
		assert.FailNowf(t, "%s %s", rr.Stdout, rr.Stderr)
	}
	return w, func() {
		_ = w.RemoveAll()
	}
}

// AddKptfileToWorkspace writes the provided Kptfile to the workspace
// and makes a commit.
func AddKptfileToWorkspace(t *testing.T, w *TestWorkspace, kf *kptfilev1.KptFile) {
	err := os.MkdirAll(w.FullPackagePath(), 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = kptfileutil.WriteFile(w.FullPackagePath(), kf)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	gitRunner, err := gitutil.NewLocalGitRunner(w.WorkspaceDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = gitRunner.Run(fake.CtxWithDefaultPrinter(), "add", ".")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = w.Commit("added Kptfile")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}

// SetupRepos creates repos and returns a mapping from name to TestGitRepos.
// This only creates the first version of each repo as given by the first item
// in the repoContent slice.
func SetupRepos(t *testing.T, repoContent map[string][]Content) (map[string]*TestGitRepo, func()) {
	repos := make(map[string]*TestGitRepo)

	cleanupFunc := func() {
		for _, rp := range repos {
			_ = os.RemoveAll(rp.RepoDirectory)
		}
	}

	ordering, err := findRepoOrdering(repoContent)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	for _, name := range ordering {
		data := repoContent[name]
		if len(data) < 1 {
			continue
		}
		tgr := &TestGitRepo{T: t}
		repos[name] = tgr
		if err := tgr.SetupTestGitRepo(name, data[:1], repos); err != nil {
			return repos, cleanupFunc
		}
	}
	return repos, cleanupFunc
}

// UpdateRepos updates the existing repos with any additional Content
// items in the repoContent slice.
func UpdateRepos(t *testing.T, repos map[string]*TestGitRepo, repoContent map[string][]Content) error {
	ordering, err := findRepoOrdering(repoContent)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	for _, name := range ordering {
		data := repoContent[name]
		if len(data) < 1 {
			continue
		}

		r := repos[name]
		err := UpdateGitDir(t, name, r, data[1:], repos)
		if err != nil {
			return err
		}
	}
	return nil
}

// findRepoOrdering orders the repos based on their dependencies. So if repo
// A includes repo B as a subpackage, we can create repo B before we create
// repo B. This is done with a topological sort. If there are any circular
// dependencies between the repos, it will return an error.
func findRepoOrdering(repoContent map[string][]Content) ([]string, error) {
	var repoNames []string
	for n := range repoContent {
		repoNames = append(repoNames, n)
	}

	topo := toposort.NewGraph(len(repoNames))
	topo.AddNodes(repoNames...)
	// Keep track of which edges have been added to topo. The library doesn't
	// handle the same edge added multiple times.
	added := make(map[string]string)
	for n, contents := range repoContent {
		for _, c := range contents {
			if c.Pkg == nil {
				continue
			}
			pkg := c.Pkg
			refRepos := pkg.AllReferencedRepos()
			for _, refRepo := range refRepos {
				if v, ok := added[refRepo]; ok && v == n {
					continue
				}
				topo.AddEdge(refRepo, n)
				added[refRepo] = n
			}
		}
	}
	ordering, ok := topo.Toposort()
	if !ok {
		return nil, fmt.Errorf("unable to sort repo references. Cycles are not allowed")
	}
	return ordering, nil
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

// nolint:gocyclo
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
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				path, err := os.Readlink(path)
				if err != nil {
					return err
				}
				return os.Symlink(path, filepath.Join(repo, rel))
			case info.IsDir():
				return os.Mkdir(filepath.Join(repo, rel), 0700)
			default:
				return copyutil.SyncFile(path, filepath.Join(repo, rel))
			}
		}

		// If it is a directory and we know it already exists, we don't need
		// to do anything.
		if info.IsDir() {
			return nil
		}

		// For Kptfiles we want to keep the Upstream section if the Kptfile
		// in the data directory doesn't already include one.
		if filepath.Base(path) == "Kptfile" {
			dataKptfile, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, filepath.Dir(path))
			if err != nil {
				return err
			}
			repoKptfileDir := filepath.Dir(filepath.Join(repo, rel))
			repoKptfile, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, repoKptfileDir)
			if err != nil {
				return err
			}
			// Only copy over the Upstream section from the existing
			// Kptfile if other values hasn't been provided.
			if dataKptfile.Upstream == nil || reflect.DeepEqual(dataKptfile.Upstream, kptfilev1.Upstream{}) {
				dataKptfile.Upstream = repoKptfile.Upstream
			}
			if dataKptfile.UpstreamLock == nil || reflect.DeepEqual(dataKptfile.UpstreamLock, kptfilev1.UpstreamLock{}) {
				dataKptfile.UpstreamLock = repoKptfile.UpstreamLock
			}
			dataKptfile.Name = repoKptfile.Name
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

		// Never delete the Kptfile in the root package.
		if rel == kptfilev1.KptFileName {
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

func commit(repo, message string) (string, error) {
	cmd := exec.Command("git", "commit", "-m", message, "--allow-empty")
	cmd.Dir = repo
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return "", err
	}

	sha, err := git.LookupCommit(repo)
	if err != nil {
		return "", err
	}

	return sha, nil
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
	w.WorkspaceDirectory, err = os.MkdirTemp("", "test-kpt-local-")
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

// CustomUpdate executes the provided update function and passes in the
// path to the directory of the repository.
func (w *TestWorkspace) CustomUpdate(f func(string) error) error {
	return f(w.WorkspaceDirectory)
}

// Commit performs a git commit
func (w *TestWorkspace) Commit(message string) (string, error) {
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
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func Chdir(t *testing.T, path string) func() {
	cwd, err := os.Getwd()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	revertFunc := func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	}
	err = os.Chdir(path)
	if !assert.NoError(t, err) {
		defer revertFunc()
		t.FailNow()
	}
	return revertFunc
}

// ConfigureTestKptCache sets up a temporary directory for the kpt git
// cache, sets the env variable so it will be used for tests, and cleans
// up the directory afterwards.
func ConfigureTestKptCache(m *testing.M) int {
	cacheDir, err := os.MkdirTemp("", "kpt-test-cache-repos-")
	if err != nil {
		panic(fmt.Errorf("error creating temp dir for cache: %w", err))
	}
	defer func() {
		_ = os.RemoveAll(cacheDir)
	}()
	if err := os.Setenv(gitutil.RepoCacheDirEnv, cacheDir); err != nil {
		panic(fmt.Errorf("error setting repo cache env variable: %w", err))
	}
	return m.Run()
}

var EmptyReposInfo = &ReposInfo{}

func ToReposInfo(repos map[string]*TestGitRepo) *ReposInfo {
	return &ReposInfo{
		repos: repos,
	}
}

type ReposInfo struct {
	repos map[string]*TestGitRepo
}

func (ri *ReposInfo) ResolveRepoRef(repoRef string) (string, bool) {
	repo, found := ri.repos[repoRef]
	if !found {
		return "", false
	}
	return repo.RepoDirectory, true
}

func (ri *ReposInfo) ResolveCommitIndex(repoRef string, index int) (string, bool) {
	repo, found := ri.repos[repoRef]
	if !found {
		return "", false
	}
	commits := repo.Commits
	if len(commits) <= index {
		return "", false
	}
	return commits[index], true
}
