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

package get_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/get"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failEmptyRepo(t *testing.T) {
	_, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{})
	defer clean()

	err := Command{
		Destination: w.WorkspaceDirectory,
	}.Run()
	assert.EqualError(t, err, "must specify git repo information")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoRevision(t *testing.T) {
	_, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{})
	defer clean()

	err := Command{
		Git: &kptfilev1alpha2.Git{
			Repo: "foo",
		},
		Destination: w.WorkspaceDirectory,
	}.Run()
	assert.EqualError(t, err, "must specify ref")
}

// TestCommand_Run verifies that Command will clone the HEAD of the master branch.
//
// - destination directory should match the base name of the repo
// - KptFile should be populated with values pointing to the origin
func TestCommand_Run(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{Git: &kptfilev1alpha2.Git{
		Repo:      "file://" + g.RepoDirectory,
		Ref:       "master",
		Directory: "/",
	},
		Destination: absPath}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
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
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})
}

// TestCommand_Run_subdir verifies that Command will clone a subdirectory of a repo.
//
// - destination dir should match the name of the subdirectory
// - KptFile should have the subdir listed
func TestCommand_Run_subdir(t *testing.T) {
	subdir := "java"
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, subdir)
	err := Command{Git: &kptfilev1alpha2.Git{
		Repo: g.RepoDirectory, Ref: "refs/heads/master", Directory: subdir},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: subdir,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.TypeMeta.APIVersion,
				Kind:       kptfilev1alpha2.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: "git",
			GitLock: &kptfilev1alpha2.GitLock{
				Commit:    commit,
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: "git",
			Git: &kptfilev1alpha2.Git{
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})
}

// TestCommand_Run_destination verifies Command clones the repo to a destination with a specific name rather
// than using the name of the source repo.
func TestCommand_Run_destination(t *testing.T) {
	dest := "my-dataset"
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, dest)
	err := Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.TypeMeta.APIVersion,
				Kind:       kptfilev1alpha2.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit,
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})
}

// TestCommand_Run_subdirAndDestination verifies that Command will copy a subdirectory of a repo to a
// specific destination.
//
// - name of the destination is used over the name of the subdir in the KptFile
func TestCommand_Run_subdirAndDestination(t *testing.T) {
	subdir := "java"
	dest := "new-java"
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, dest)
	err := Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: subdir,
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.TypeMeta.APIVersion,
				Kind:       kptfilev1alpha2.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Commit:    commit,
				Directory: subdir,
				Ref:       "master",
				Repo:      g.RepoDirectory,
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: subdir,
				Ref:       "master",
				Repo:      g.RepoDirectory,
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})
}

// TestCommand_Run_branch verifies Command can clone a git branch
//
// 1. create a new branch
// 2. add data to the branch
// 3. checkout the master branch again
// 4. clone the new branch
// 5. verify contents match the new branch
func TestCommand_Run_branch(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	// add commits to the exp branch
	err := g.CheckoutBranch("exp", true)
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	_, err = g.Commit("new dataset")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.CheckoutBranch("master", false)
	assert.NoError(t, err)
	commit2, err := g.GetCommit()
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit2)

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/heads/exp",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
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
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/heads/exp",
				Commit:    commit,
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/heads/exp",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})
}

// TestCommand_Run_tag verifies Command can clone from a git tag
//
// 1. add data to the master branch
// 2. commit and tag the master branch
// 3. add more data to the master branch, commit it
// 4. clone at the tag
// 5. verify the clone has the data from the tagged version
func TestCommand_Run_tag(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	// create a commit with dataset2 and tag it v2, then add another commit on top with dataset3
	commit0, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	_, err = g.Commit("new-data for v2")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.Tag("v2")
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset3)
	assert.NoError(t, err)
	_, err = g.Commit("new-data post-v2")
	assert.NoError(t, err)
	commit2, err := g.GetCommit()
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit0)
	assert.NotEqual(t, commit, commit2)

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/tags/v2",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
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
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
				Commit:    commit,
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})
}

// TestCommand_Run_failExistingDir verifies that command will fail without changing anything if the
// directory already exists
func TestCommand_Run_failExistingDir(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
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
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})

	// update the data that would be cloned
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	_, err = g.Commit("new-data")
	assert.NoError(t, err)

	// try to clone and expect a failure
	err = Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.EqualError(t, err, fmt.Sprintf("destination directory %s already exists", absPath))

	// verify files are unchanged
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfilev1alpha2.KptFile{
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
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
	})
}

func TestCommand_Run_failInvalidRepo(t *testing.T) {
	_, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, "foo")
	err := Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      "foo",
			Directory: "/",
			Ref:       "refs/heads/master",
		},
		Destination: absPath,
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "failed to lookup master(or main) branch") {
		t.FailNow()
	}
}

func TestCommand_Run_failInvalidBranch(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoDirectory)
	err := Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Directory: "/",
			Ref:       "refs/heads/foo",
		},
		Destination: absPath,
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/heads/foo") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}
}

func TestCommand_Run_failInvalidTag(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	err := Command{
		Git: &kptfilev1alpha2.Git{
			Repo:      g.RepoDirectory,
			Directory: "/",
			Ref:       "refs/tags/foo",
		},
		Destination: filepath.Join(w.WorkspaceDirectory, g.RepoDirectory),
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/tags/foo") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}
}

func TestCommand_Run_subpackages(t *testing.T) {
	testCases := []struct {
		name           string
		directory      string
		ref            string
		reposContent   map[string][]testutil.Content
		expectedResult *pkgbuilder.RootPkg
		expectedErrMsg string
	}{
		{
			name:      "basic package",
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "master", 0),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		{
			name:      "package with subpackages",
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile().
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "master", 0),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		{
			name:      "package with deeply nested subpackages",
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithResource(pkgbuilder.ConfigMapResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("deepsubpkg").
											WithKptfile(
												pkgbuilder.NewKptfile().
													WithUpstreamRef("foo", "/", "main", "fast-forward"),
											),
									),
							),
					},
				},
				"foo": {
					{
						Branch: "main",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.SecretResource),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "master", 0),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithResource(pkgbuilder.ConfigMapResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("deepsubpkg").
								WithKptfile(
									pkgbuilder.NewKptfile().
										WithUpstreamRef("foo", "/", "main", "fast-forward").
										WithUpstreamLockRef("foo", "/", "main", 0),
								).
								WithResource(pkgbuilder.SecretResource),
						),
				),
		},
		{
			name:      "package with local and remote subpackages",
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithResource(pkgbuilder.ConfigMapResource),
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "main", "fast-forward").
											WithUpstreamLockRef("foo", "/", "main", 0),
									).
									WithResource(pkgbuilder.DeploymentResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("subpkg").
											WithKptfile(
												pkgbuilder.NewKptfile(),
											).
											WithResource(pkgbuilder.ConfigMapResource).
											WithSubPackages(
												pkgbuilder.NewSubPkg("bar").
													WithKptfile(
														pkgbuilder.NewKptfile().
															WithUpstreamRef("bar", "/", "main", "fast-forward").
															WithUpstreamLockRef("bar", "/", "main", 0),
													).WithResource(pkgbuilder.DeploymentResource),
											),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile(),
									).
									WithResource(pkgbuilder.ConfigMapResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("bar").
											WithKptfile(
												pkgbuilder.NewKptfile().
													WithUpstreamRef("bar", "/", "main", "fast-forward").
													WithUpstreamLockRef("bar", "/", "main", 0),
											).
											WithResource(pkgbuilder.DeploymentResource),
									),
							),
					},
				},
				"bar": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "master", 0),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "main", "fast-forward").
								WithUpstreamLockRef("foo", "/", "main", 0),
						).
						WithResource(pkgbuilder.DeploymentResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(
									pkgbuilder.NewKptfile(),
								).
								WithResource(pkgbuilder.ConfigMapResource).
								WithSubPackages(
									pkgbuilder.NewSubPkg("bar").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstreamRef("bar", "/", "main", "fast-forward").
												WithUpstreamLockRef("bar", "/", "main", 0),
										).
										WithResource(pkgbuilder.DeploymentResource),
								),
						),
					pkgbuilder.NewSubPkg("subpkg").
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		{
			name:      "fetch subpackage on a different branch than master",
			directory: "/bar",
			ref:       "main",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "main",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile().
									WithResource(pkgbuilder.ConfigMapResource),
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/subpkg", "v1.2", "fast-forward"),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/bar", "main", "resource-merge").
						WithUpstreamLockRef("upstream", "/bar", "main", 0),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		{
			name:      "package with unfetched remote subpackage with a tag reference",
			directory: "/",
			ref:       "main",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "main",
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile().
									WithResource(pkgbuilder.ConfigMapResource),
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/subpkg", "v1.2", "fast-forward"),
									),
							),
					},
				},
				"foo": {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile().
									WithResource(pkgbuilder.DeploymentResource),
							),
						Tag:    "v1.2",
						Branch: "master",
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "main", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "main", 0),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/subpkg", "v1.2", "fast-forward").
								WithUpstreamLockRef("foo", "/subpkg", "v1.2", 1),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		{
			name:      "same unfetched remote subpackage referenced multiple times",
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo-sub").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/subpkg", "subpkg/v1.2", "fast-forward"),
									),
								pkgbuilder.NewSubPkg("foo-root").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "master", "fast-forward"),
									),
							),
					},
				},
				"foo": {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile().
									WithResource(pkgbuilder.DeploymentResource),
							),
						Tag: "subpkg/v1.2",
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "master", 0),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo-sub").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/subpkg", "subpkg/v1.2", "fast-forward").
								WithUpstreamLockRef("foo", "/subpkg", "subpkg/v1.2", 1),
						).
						WithResource(pkgbuilder.DeploymentResource),
					pkgbuilder.NewSubPkg("foo-root").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "master", "fast-forward").
								WithUpstreamLockRef("foo", "/", "master", 1),
						).
						WithResource(pkgbuilder.ConfigMapResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("subpkg").
								WithKptfile().
								WithResource(pkgbuilder.DeploymentResource),
						),
				),
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			repos, w, clean := testutil.SetupReposAndWorkspace(t, test.reposContent)
			defer clean()
			upstreamRepo := repos[testutil.Upstream]
			err := testutil.UpdateRepos(t, repos, test.reposContent)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			var targetDir string
			if test.directory == "/" {
				targetDir = filepath.Base(upstreamRepo.RepoName)
			} else {
				targetDir = filepath.Base(test.directory)
			}
			w.PackageDir = targetDir
			destinationDir := filepath.Join(w.WorkspaceDirectory, targetDir)

			err = Command{
				Git: &kptfilev1alpha2.Git{
					Repo:      upstreamRepo.RepoDirectory,
					Directory: test.directory,
					Ref:       test.ref,
				},
				Destination: destinationDir,
			}.Run()

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

			// Format the Kptfiles so we can diff the output without
			// formatting issues.
			rw := &kio.LocalPackageReadWriter{
				NoDeleteFiles:  true,
				PackagePath:    w.FullPackagePath(),
				MatchFilesGlob: []string{kptfilev1alpha2.KptFileName},
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{rw},
				Filters: []kio.Filter{filters.FormatFilter{}},
				Outputs: []kio.Writer{rw},
			}.Execute()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			expectedPath := test.expectedResult.ExpandPkgWithName(t, targetDir, testutil.ToReposInfo(repos))
			testutil.KptfileAwarePkgEqual(t, expectedPath, w.FullPackagePath())
		})
	}
}
