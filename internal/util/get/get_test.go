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
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failEmptyRepo(t *testing.T) {
	_, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{}, map[string]string{})
	defer clean()

	err := Command{
		Destination: w.WorkspaceDirectory,
	}.Run()
	assert.EqualError(t, err, "must specify repo")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoRevision(t *testing.T) {
	_, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{}, map[string]string{})
	defer clean()

	err := Command{
		Git: kptfile.Git{
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{Git: kptfile.Git{
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
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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

// TestCommand_Run_subdir verifies that Command will clone a subdirectory of a repo.
//
// - destination dir should match the name of the subdirectory
// - KptFile should have the subdir listed
func TestCommand_Run_subdir(t *testing.T) {
	subdir := "java"
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, subdir)
	err := Command{Git: kptfile.Git{
		Repo: g.RepoDirectory, Ref: "refs/heads/master", Directory: subdir},
		Destination: absPath,
	}.Run()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: subdir,
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
				Commit:    commit,
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
		},
	})
}

// TestCommand_Run_destination verifies Command clones the repo to a destination with a specific name rather
// than using the name of the source repo.
func TestCommand_Run_destination(t *testing.T) {
	dest := "my-dataset"
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, dest)
	err := Command{
		Git: kptfile.Git{
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
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
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
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit,
			},
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, dest)
	err := Command{
		Git: kptfile.Git{
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
	g.AssertKptfile(t, absPath, kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
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
				Commit:    commit,
				Directory: subdir,
				Ref:       "master",
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	// add commits to the exp branch
	err := g.CheckoutBranch("exp", true)
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new dataset")
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
		Git: kptfile.Git{
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
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	// create a commit with dataset2 and tag it v2, then add another commit on top with dataset3
	commit0, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data for v2")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.Tag("v2")
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset3)
	assert.NoError(t, err)
	err = g.Commit("new-data post-v2")
	assert.NoError(t, err)
	commit2, err := g.GetCommit()
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit0)
	assert.NotEqual(t, commit, commit2)

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Git: kptfile.Git{
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
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
				Commit:    commit,
			},
		},
	})
}

// TestCommand_Run_clean verifies that the Command delete the existing directory if Clean is set.
//
// 1. clone the master branch
// 2. add data to the master branch and commit it
// 3. clone the master branch again
// 4. verify the new master branch data is present
func TestCommand_Run_clean(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{
		Git: kptfile.Git{
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

	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})

	// update the data that would be cloned
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data")
	assert.NoError(t, err)

	// verify the KptFile contains the expected values
	commit, err = g.GetCommit()
	assert.NoError(t, err)

	// configure clone to clean the existing dir
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
		Clean:       true,
	}.Run()
	assert.NoError(t, err)

	// verify files are updated
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCommand_Run_failClean verifies that the Command will not clean the existing directory if it
// fails to clone.
//
// 1. clone the master branch
// 2. clone a non-existing branch
// 3. verify the master branch data is still present
func TestCommand_Run_failClean(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{
		Git: kptfile.Git{
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
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})

	// configure clone to clean the existing dir, but fail
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/heads/not-real",
			Directory: "/",
		},
		Destination: absPath,
		Clean:       true,
	}.Run()
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/heads/not-real") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}

	// verify files weren't deleted
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

// TestCommand_Run_failExistingDir verifies that command will fail without changing anything if the
// directory already exists
func TestCommand_Run_failExistingDir(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err := Command{
		Git: kptfile.Git{
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
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})

	// update the data that would be cloned
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data")
	assert.NoError(t, err)

	// try to clone and expect a failure
	err = Command{
		Git: kptfile.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run()
	assert.EqualError(t, err, fmt.Sprintf("destination directory %s already exists", absPath))

	// verify files are unchanged
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath)
	g.AssertKptfile(t, absPath, kptfile.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
	})
}

func TestCommand_Run_failInvalidRepo(t *testing.T) {
	_, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, "foo")
	err := Command{
		Git: kptfile.Git{
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoDirectory)
	err := Command{
		Git: kptfile.Git{
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
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}, map[string]string{})
	defer clean()

	err := Command{
		Git: kptfile.Git{
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
		upstream       testutil.Content
		refRepos       map[string][]testutil.Content
		expectedResult *pkgbuilder.RootPkg
		expectedErrMsg string
	}{
		{
			name:      "basic package",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Branch: "master",
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.DeploymentResource),
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master"),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		{
			name:      "package with subpackages",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
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
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master"),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		{
			name:      "package with local and remote subpackages",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Branch: "master",
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithSubpackages(
								pkgbuilder.NewSubpackage("foo", "/", "main", "fast-forward", "foo"),
							),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithSubpackages(
												pkgbuilder.NewSubpackage("bar", "/", "main", "fast-forward", "bar"),
											),
									).
									WithResource(pkgbuilder.ConfigMapResource),
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
						WithSubpackages(
							pkgbuilder.NewSubpackage("foo", "/", "main", "fast-forward", "foo"),
						).
						WithUpstreamRef("upstream", "/", "master"),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "main"),
						).
						WithResource(pkgbuilder.DeploymentResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(
									pkgbuilder.NewKptfile().
										WithSubpackages(
											pkgbuilder.NewSubpackage("bar", "/", "main", "fast-forward", "bar"),
										),
								).
								WithResource(pkgbuilder.ConfigMapResource).
								WithSubPackages(
									pkgbuilder.NewSubPkg("bar").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstreamRef("bar", "/", "main"),
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
			upstream: testutil.Content{
				Branch: "main",
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithSubpackages(
								pkgbuilder.NewSubpackage("foo", "/subpkg", "v1.2", "fast-forward", "foo"),
							),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			refRepos: map[string][]testutil.Content{
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
						WithUpstreamRef("upstream", "/bar", "main"),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		{
			name:      "package with remote subpackage with a tag reference",
			directory: "/",
			ref:       "main",
			upstream: testutil.Content{
				Branch: "main",
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithSubpackages(
								pkgbuilder.NewSubpackage("foo", "/subpkg", "v1.2", "fast-forward", "foo"),
							),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			refRepos: map[string][]testutil.Content{
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
						Tag: "v1.2",
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithSubpackages(
							pkgbuilder.NewSubpackage("foo", "/subpkg", "v1.2", "fast-forward", "foo"),
						).
						WithUpstreamRef("upstream", "/", "main"),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/subpkg", "v1.2"),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		{
			name:      "same remote subpackage referenced multiple times",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Branch: "master",
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithSubpackages(
								pkgbuilder.NewSubpackage("foo", "/subpkg", "subpkg/v1.2", "fast-forward", "foo-sub"),
								pkgbuilder.NewSubpackage("foo", "/", "master", "resource-merge", "foo-root"),
							),
					),
			},
			refRepos: map[string][]testutil.Content{
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
						WithSubpackages(
							pkgbuilder.NewSubpackage("foo", "/subpkg", "subpkg/v1.2", "fast-forward", "foo-sub"),
							pkgbuilder.NewSubpackage("foo", "/", "master", "resource-merge", "foo-root"),
						).
						WithUpstreamRef("upstream", "/", "master"),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo-sub").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/subpkg", "subpkg/v1.2"),
						).
						WithResource(pkgbuilder.DeploymentResource),
					pkgbuilder.NewSubPkg("foo-root").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "master"),
						).
						WithResource(pkgbuilder.ConfigMapResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("subpkg").
								WithKptfile().
								WithResource(pkgbuilder.DeploymentResource),
						),
				),
		},
		{
			name:      "conflict between local and remote subpackage",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Branch: "master",
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithSubpackages(
								pkgbuilder.NewSubpackage("foo", "/", "master", "fast-forward", "foo"),
							),
					).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			expectedErrMsg: "local subpackage in directory \"foo\" already exist",
		},
		{
			name:      "conflict between two remote subpackages",
			directory: "/",
			ref:       "master",
			upstream: testutil.Content{
				Branch: "master",
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithSubpackages(
								pkgbuilder.NewSubpackage("foo", "/", "master", "fast-forward", "subpkg"),
								pkgbuilder.NewSubpackage("bar", "/", "master", "fast-forward", "subpkg"),
							),
					),
			},
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
				"bar": {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			expectedErrMsg: "multiple remote subpackages with localDir \"subpkg\"",
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			refRepos, err := testutil.SetupRepos(t, test.refRepos)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			repoPaths := make(map[string]string)
			for name, tgr := range refRepos {
				repoPaths[name] = tgr.RepoDirectory
			}

			err = testutil.UpdateRefRepos(t, refRepos, test.refRepos, repoPaths)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, test.upstream, repoPaths)
			defer clean()

			var targetDir string
			if test.directory == "/" {
				targetDir = filepath.Base(g.RepoName)
			} else {
				targetDir = filepath.Base(test.directory)
			}
			w.PackageDir = targetDir
			destinationDir := filepath.Join(w.WorkspaceDirectory, targetDir)

			err = Command{
				Git: kptfile.Git{
					Repo:      g.RepoDirectory,
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
				MatchFilesGlob: []string{kptfile.KptFileName},
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{rw},
				Filters: []kio.Filter{filters.FormatFilter{}},
				Outputs: []kio.Writer{rw},
			}.Execute()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			expectedPath := pkgbuilder.ExpandPkgWithName(t, test.expectedResult, targetDir, repoPaths)
			testutil.KptfileAwarePkgEqual(t, expectedPath, w.FullPackagePath())
		})
	}
}

func TestCommand_Run_fetchIntoSubpackage(t *testing.T) {
	testCases := map[string]struct {
		initialUpstream *pkgbuilder.RootPkg
		refRepos        map[string][]testutil.Content
		subPkgPath      string
		repoRef         string
		directory       string
		ref             string
		expectedResult  *pkgbuilder.RootPkg
		expectedErrMsg  string
	}{
		"fetching a subpackage should update parent Kptfile": {
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: "master",
					},
				},
			},
			subPkgPath: "foo",
			repoRef:    "foo",
			directory:  "/",
			ref:        "master",
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master").
						WithSubpackages(
							pkgbuilder.NewSubpackage("foo", "/", "master", "resource-merge", "foo"),
						),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "master"),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"Kptfile should be updated when parent is not in the immediate parent folder": {
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithSubPackages(
								pkgbuilder.NewSubPkg("nested").
									WithResource(pkgbuilder.ConfigMapResource),
							),
						Branch: "main",
						Tag:    "my-tag",
					},
				},
			},
			subPkgPath: "deeply/nested/package",
			repoRef:    "foo",
			directory:  "nested",
			ref:        "my-tag",
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master").
						WithSubpackages(
							pkgbuilder.NewSubpackage("foo", "nested", "my-tag", "resource-merge", "deeply/nested/package"),
						),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("deeply").
						WithSubPackages(
							pkgbuilder.NewSubPkg("nested").
								WithSubPackages(
									pkgbuilder.NewSubPkg("package").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstreamRef("foo", "nested", "my-tag"),
										).
										WithResource(pkgbuilder.ConfigMapResource),
								),
						),
				),
		},
		"it is an error if there is already another subpackage in the specified path": {
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithSubpackages(
							pkgbuilder.NewSubpackage("bar", "/", "master", "fast-forward", "sub"),
						),
				).
				WithResource(pkgbuilder.DeploymentResource),
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: "main",
					},
				},
				"bar": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: "master",
					},
				},
			},
			subPkgPath:     "sub",
			repoRef:        "foo",
			directory:      "/",
			ref:            "main",
			expectedErrMsg: "subpackage with localDir \"sub\" already exist",
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master").
						WithSubpackages(
							pkgbuilder.NewSubpackage("bar", "/", "master", "fast-forward", "sub"),
						),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("sub").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("bar", "/", "master"),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"if the package can't be fetched, we roll back the change to the Kptfile": {
			initialUpstream: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.DeploymentResource),
			refRepos: map[string][]testutil.Content{
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: "main",
					},
				},
			},
			subPkgPath:     "sub",
			repoRef:        "foo",
			directory:      "/",
			ref:            "unknownRef",
			expectedErrMsg: "failed to clone git repo",
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master"),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T:               t,
				RefReposChanges: tc.refRepos,
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Pkg:    tc.initialUpstream,
				Branch: "master",
			}) {
				return
			}

			repoPath, found := g.RepoPaths[tc.repoRef]
			if !found {
				t.Errorf("expected to found a path for repoRef %q, but didn't", tc.repoRef)
			}
			fullSubPkgPath := filepath.Join(g.LocalWorkspace.FullPackagePath(), tc.subPkgPath)
			err := Command{
				Git: kptfile.Git{
					Repo:      repoPath,
					Directory: tc.directory,
					Ref:       tc.ref,
				},
				Destination: fullSubPkgPath,
			}.Run()

			if tc.expectedErrMsg != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tc.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Format the Kptfiles so we can diff the output without
			// formatting issues.
			rw := &kio.LocalPackageReadWriter{
				NoDeleteFiles:  true,
				PackagePath:    g.LocalWorkspace.FullPackagePath(),
				MatchFilesGlob: []string{kptfile.KptFileName},
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{rw},
				Filters: []kio.Filter{filters.FormatFilter{}},
				Outputs: []kio.Writer{rw},
			}.Execute()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			expectedPath := pkgbuilder.ExpandPkgWithName(t, tc.expectedResult, g.UpstreamRepo.RepoName, g.RepoPaths)
			testutil.KptfileAwarePkgEqual(t, expectedPath, g.LocalWorkspace.FullPackagePath())
		})
	}
}
