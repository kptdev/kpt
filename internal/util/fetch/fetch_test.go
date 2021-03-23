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

package fetch_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	. "github.com/GoogleContainerTools/kpt/internal/util/fetch"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func setupWorkspace(t *testing.T) (*testutil.TestGitRepo, *testutil.TestWorkspace, func()) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})

	w.PackageDir = g.RepoName
	err := os.MkdirAll(w.FullPackagePath(), 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return g, w, clean
}

func createKptfile(workspace *testutil.TestWorkspace, git *kptfilev1alpha2.Git, strategy kptfilev1alpha2.UpdateStrategyType) error {
	kf := kptfileutil.DefaultKptfile(workspace.PackageDir)
	kf.Upstream = &kptfilev1alpha2.Upstream{
		Type:           kptfilev1alpha2.GitOrigin,
		Git:            git,
		UpdateStrategy: strategy,
	}
	return kptfileutil.WriteFile(workspace.FullPackagePath(), kf)
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if no Kptfile
func TestCommand_Run_failNoKptfile(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	pkgPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := os.MkdirAll(pkgPath, 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: pkgPath,
	}.Run()
	assert.EqualError(t, err, "no Kptfile found")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoGit(t *testing.T) {
	_, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, nil, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
	}.Run()
	assert.EqualError(t, err, "kptfile upstream doesn't have git information")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failEmptyRepo(t *testing.T) {
	_, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      "",
		Directory: "/",
		Ref:       "main",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
	}.Run()
	assert.EqualError(t, err, "must specify repo")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoRevision(t *testing.T) {
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      "file://" + g.RepoDirectory,
		Directory: "/",
		Ref:       "",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
	}.Run()
	assert.EqualError(t, err, "must specify ref")
}

// TestCommand_Run verifies that Command will clone the HEAD of the master branch.
//
// - destination directory should match the base name of the repo
// - KptFile should be populated with values pointing to the origin
func TestCommand_Run(t *testing.T) {
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      "file://" + g.RepoDirectory,
		Directory: "/",
		Ref:       "master",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Path: w.FullPackagePath(),
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
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.KptFileAPIVersion,
				Kind:       kptfilev1alpha2.KptFileName},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: "git",
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: "git",
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCommand_Run_subdir verifies that Command will clone a subdirectory of a repo.
//
// - destination dir should match the name of the subdirectory
// - KptFile should have the subdir listed
func TestCommand_Run_subdir(t *testing.T) {
	g, w, clean := setupWorkspace(t)
	defer clean()

	subdir := "java"
	err := createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      g.RepoDirectory,
		Directory: subdir,
		Ref:       "refs/heads/master",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Path: w.FullPackagePath(),
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
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.KptFileAPIVersion,
				Kind:       kptfilev1alpha2.KptFileName},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: kptfilev1alpha2.GitOrigin,
			Git: &kptfilev1alpha2.Git{
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Commit:    commit,
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
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
	g, w, clean := setupWorkspace(t)
	defer clean()

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

	err = createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/heads/exp",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), w.FullPackagePath())

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, w.FullPackagePath(), kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.KptFileAPIVersion,
				Kind:       kptfilev1alpha2.KptFileName},
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
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: kptfilev1alpha2.GitOrigin,
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/heads/exp",
				Commit:    commit,
			},
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
	g, w, clean := setupWorkspace(t)
	defer clean()

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

	err = createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/tags/v2",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), w.FullPackagePath())

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, w.FullPackagePath(), kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: g.RepoName,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.KptFileAPIVersion,
				Kind:       kptfilev1alpha2.KptFileName},
		},
		Upstream: &kptfilev1alpha2.Upstream{
			Type: "git",
			Git: &kptfilev1alpha2.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
			},
			UpdateStrategy: kptfilev1alpha2.ResourceMerge,
		},
		UpstreamLock: &kptfilev1alpha2.UpstreamLock{
			Type: "git",
			GitLock: &kptfilev1alpha2.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
				Commit:    commit,
			},
		},
	})
}

func TestCommand_Run_failInvalidRepo(t *testing.T) {
	_, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      "foo",
		Directory: "/",
		Ref:       "refs/heads/master",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "failed to lookup master(or main) branch") {
		t.FailNow()
	}
}

func TestCommand_Run_failInvalidBranch(t *testing.T) {
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/heads/foo",
	}, kptfilev1alpha2.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
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
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1alpha2.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/tags/foo",
	}, kptfilev1alpha2.FastForward)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Path: w.FullPackagePath(),
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
