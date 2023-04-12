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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/get"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const JavaSubdir = "java"

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failEmptyRepo(t *testing.T) {
	_, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{})
	defer clean()

	err := Command{
		Destination: w.WorkspaceDirectory,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "must specify git repo information")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoRevision(t *testing.T) {
	_, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{})
	defer clean()

	err := Command{
		Git: &kptfilev1.Git{
			Repo: "foo",
		},
		Destination: w.WorkspaceDirectory,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "must specify ref")
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
	err := Command{Git: &kptfilev1.Git{
		Repo:      "file://" + g.RepoDirectory,
		Ref:       "master",
		Directory: "/",
	},
		Destination: absPath}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath, true)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
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
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
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
	})
}

// TestCommand_Run_subdir verifies that Command will clone a subdirectory of a repo.
//
// - destination dir should match the name of the subdirectory
// - KptFile should have the subdir listed
func TestCommand_Run_subdir(t *testing.T) {
	subdir := JavaSubdir
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, subdir)
	err := Command{Git: &kptfilev1.Git{
		Repo: g.RepoDirectory, Ref: "refs/heads/master", Directory: subdir},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath, true)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: subdir,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.TypeMeta.APIVersion,
				Kind:       kptfilev1.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: "git",
			Git: &kptfilev1.GitLock{
				Commit:    commit,
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: "git",
			Git: &kptfilev1.Git{
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
	})
}

// TestCommand_Run_subdir_symlinks verifies Command will
// clone a subdirectory of a repo inside the subdirectory.
//
// - destination dir should match the name of the subdirectory
// - KptFile should have the subdir listed
// - Content outside the subdirectory should be ignored
// - symlinks inside the subdirectory should be ignored
func TestCommand_Run_subdir_symlinks(t *testing.T) {
	subdir := JavaSubdir
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset6,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	cliOutput := &bytes.Buffer{}

	absPath := filepath.Join(w.WorkspaceDirectory, subdir)
	err := Command{Git: &kptfilev1.Git{
		Repo: g.RepoDirectory, Ref: "refs/heads/master", Directory: subdir},
		Destination: absPath,
	}.Run(fake.CtxWithPrinter(cliOutput, cliOutput))
	assert.NoError(t, err)

	// ensure warning for symlink is printed on the CLI
	assert.Contains(t, cliOutput.String(), `[Warn] Ignoring symlink "config-symlink"`)

	// verify the cloned contents do not contains symlinks
	diff, err := testutil.Diff(filepath.Join(g.DatasetDirectory, testutil.Dataset6, subdir), absPath, true)
	assert.NoError(t, err)
	diff = diff.Difference(testutil.KptfileSet)
	// original repo contains symlink and cloned doesn't, so the difference
	assert.Contains(t, diff.List(), "config-symlink")

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: subdir,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.TypeMeta.APIVersion,
				Kind:       kptfilev1.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: "git",
			Git: &kptfilev1.GitLock{
				Commit:    commit,
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: "git",
			Git: &kptfilev1.Git{
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
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
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath, true)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.TypeMeta.APIVersion,
				Kind:       kptfilev1.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
	})
}

// TestCommand_Run_subdirAndDestination verifies that Command will copy a subdirectory of a repo to a
// specific destination.
//
// - name of the destination is used over the name of the subdir in the KptFile
func TestCommand_Run_subdirAndDestination(t *testing.T) {
	subdir := JavaSubdir
	dest := "new-java"
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, dest)
	err := Command{
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: subdir,
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath, true)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: dest,
				},
			},
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.TypeMeta.APIVersion,
				Kind:       kptfilev1.TypeMeta.Kind},
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Commit:    commit,
				Directory: subdir,
				Ref:       "master",
				Repo:      g.RepoDirectory,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: subdir,
				Ref:       "master",
				Repo:      g.RepoDirectory,
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
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
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/heads/exp",
			Directory: "/",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath, true)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
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
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/heads/exp",
				Commit:    commit,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/heads/exp",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
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
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Ref:       "refs/tags/v2",
			Directory: "/",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), absPath, true)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
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
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
				Commit:    commit,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
	})
}

func TestCommand_Run_ref(t *testing.T) {
	testCases := map[string]struct {
		reposContent map[string][]testutil.Content
		directory    string
		ref          func(repos map[string]*testutil.TestGitRepo) string
		expected     *pkgbuilder.RootPkg
	}{
		"package tag": {
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithSubPackages(
								pkgbuilder.NewSubPkg("kafka").
									WithResource(pkgbuilder.DeploymentResource).
									WithResource(pkgbuilder.SecretResource),
							),
						Branch: "master",
						Tag:    "kafka/v2",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithSubPackages(
								pkgbuilder.NewSubPkg("kafka").
									WithResource(pkgbuilder.DeploymentResource).
									WithResource(pkgbuilder.ConfigMapResource),
							),
						Tag: "v2",
					},
				},
			},
			directory: "kafka",
			ref: func(_ map[string]*testutil.TestGitRepo) string {
				return "v2"
			},
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "kafka", "v2", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "kafka", "kafka/v2", 0),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithResource(pkgbuilder.SecretResource),
		},
		"commit sha": {
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithSubPackages(
								pkgbuilder.NewSubPkg("kafka").
									WithResource(pkgbuilder.DeploymentResource).
									WithResource(pkgbuilder.SecretResource),
							),
						Branch: "master",
						Tag:    "kafka/v2",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithSubPackages(
								pkgbuilder.NewSubPkg("kafka").
									WithResource(pkgbuilder.DeploymentResource).
									WithResource(pkgbuilder.ConfigMapResource),
							),
						Tag: "v2",
					},
				},
			},
			directory: "kafka",
			ref: func(repos map[string]*testutil.TestGitRepo) string {
				return repos[testutil.Upstream].Commits[0]
			},
			expected: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "kafka", "COMMIT-INDEX:0", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "kafka", "COMMIT-INDEX:0", 0),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithResource(pkgbuilder.SecretResource),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repos, w, clean := testutil.SetupReposAndWorkspace(t, tc.reposContent)
			defer clean()
			err := testutil.UpdateRepos(t, repos, tc.reposContent)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			ref := tc.ref(repos)

			absPath := filepath.Join(w.WorkspaceDirectory, repos[testutil.Upstream].RepoName)
			err = Command{
				Git: &kptfilev1.Git{
					Repo:      repos[testutil.Upstream].RepoDirectory,
					Ref:       ref,
					Directory: tc.directory,
				},
				Destination: absPath,
			}.Run(fake.CtxWithDefaultPrinter())
			assert.NoError(t, err)

			expectedPath := tc.expected.ExpandPkgWithName(t, repos[testutil.Upstream].RepoName, testutil.ToReposInfo(repos))

			testutil.KptfileAwarePkgEqual(t, expectedPath, absPath, true)
		})
	}
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
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the KptFile contains the expected values
	commit, err := g.GetCommit()
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath, true)
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
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
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
	})

	// update the data that would be cloned
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	_, err = g.Commit("new-data")
	assert.NoError(t, err)

	// try to clone and expect a failure
	err = Command{
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "destination directory already exists")

	// verify files are unchanged
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath, true)
	g.AssertKptfile(t, absPath, kptfilev1.KptFile{
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
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
				Commit:    commit, // verify the commit matches the repo
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
	})
}

func TestCommand_Run_nonexistingParentDir(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	defer testutil.Chdir(t, w.WorkspaceDirectory)()

	absPath := filepath.Join(w.WorkspaceDirectory, "more", "dirs", g.RepoName)
	err := Command{
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Ref:       "master",
			Directory: "/",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath, true)
}

func TestCommand_Run_failInvalidRepo(t *testing.T) {
	_, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, "foo")
	err := Command{
		Git: &kptfilev1.Git{
			Repo:      "foo",
			Directory: "/",
			Ref:       "refs/heads/master",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "'foo' does not appear to be a git repository") {
		t.FailNow()
	}

	// Confirm destination directory no longer exists.
	_, err = os.Stat(absPath)
	if !assert.Error(t, err) {
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
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Directory: "/",
			Ref:       "refs/heads/foo",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/heads/foo") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}

	// Confirm destination directory no longer exists.
	_, err = os.Stat(absPath)
	if !assert.Error(t, err) {
		t.FailNow()
	}
}

func TestCommand_Run_failInvalidTag(t *testing.T) {
	g, w, clean := testutil.SetupRepoAndWorkspace(t, testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	})
	defer clean()

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoDirectory)
	err := Command{
		Git: &kptfilev1.Git{
			Repo:      g.RepoDirectory,
			Directory: "/",
			Ref:       "refs/tags/foo",
		},
		Destination: absPath,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "refs/tags/foo") {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "exit status 128") {
		t.FailNow()
	}

	// Confirm destination directory no longer exists.
	_, err = os.Stat(absPath)
	if !assert.Error(t, err) {
		t.FailNow()
	}
}

func TestCommand_Run_subpackages(t *testing.T) {
	testCases := map[string]struct {
		directory      string
		ref            string
		updateStrategy kptfilev1.UpdateStrategyType
		reposContent   map[string][]testutil.Content
		expectedResult *pkgbuilder.RootPkg
		expectedErrMsg string
	}{
		"basic package without pipeline": {
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

		"basic package with non-KRM files": {
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithFile("foo.txt", `this is a test`),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "master", 0),
				).
				WithFile("foo.txt", `this is a test`),
		},
		"basic package with pipeline": {
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithPipeline(
										pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:latest"),
									),
							).
							WithResource(pkgbuilder.DeploymentResource),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "resource-merge").
						WithUpstreamLockRef("upstream", "/", "master", 0).
						WithPipeline(
							pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:latest"),
						),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		"basic package with no Kptfile in upstream": {
			directory: "/",
			ref:       "master",
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
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
		"basic package with explicit update strategy": {
			directory:      "/",
			ref:            "master",
			updateStrategy: kptfilev1.FastForward,
			reposContent: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Branch: "master",
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource),
					},
				},
			},
			expectedResult: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", "master", "fast-forward").
						WithUpstreamLockRef("upstream", "/", "master", 0),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		"package with subpackages": {
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
		"package with deeply nested subpackages": {
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
		"package with local and remote subpackages": {
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
		"fetch subpackage on a different branch than master": {
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
		"package with unfetched remote subpackage with a tag reference": {
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
		"same unfetched remote subpackage referenced multiple times": {
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

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repos, w, clean := testutil.SetupReposAndWorkspace(t, tc.reposContent)
			defer clean()
			upstreamRepo := repos[testutil.Upstream]
			err := testutil.UpdateRepos(t, repos, tc.reposContent)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			var targetDir string
			if tc.directory == "/" {
				targetDir = filepath.Base(upstreamRepo.RepoName)
			} else {
				targetDir = filepath.Base(tc.directory)
			}
			w.PackageDir = targetDir
			destinationDir := filepath.Join(w.WorkspaceDirectory, targetDir)

			err = Command{
				Git: &kptfilev1.Git{
					Repo:      upstreamRepo.RepoDirectory,
					Directory: tc.directory,
					Ref:       tc.ref,
				},
				Destination:    destinationDir,
				UpdateStrategy: tc.updateStrategy,
			}.Run(fake.CtxWithDefaultPrinter())

			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// Format the Kptfiles so we can diff the output without
			// formatting issues.
			rw := &kio.LocalPackageReadWriter{
				NoDeleteFiles:     true,
				PackagePath:       w.FullPackagePath(),
				MatchFilesGlob:    []string{kptfilev1.KptFileName},
				PreserveSeqIndent: true,
				WrapBareSeqNode:   true,
			}
			err = kio.Pipeline{
				Inputs:  []kio.Reader{rw},
				Filters: []kio.Filter{filters.FormatFilter{}},
				Outputs: []kio.Writer{rw},
			}.Execute()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			expectedPath := tc.expectedResult.ExpandPkgWithName(t, targetDir, testutil.ToReposInfo(repos))
			testutil.KptfileAwarePkgEqual(t, expectedPath, w.FullPackagePath(), true)
		})
	}
}

func TestCommand_Run_symlinks(t *testing.T) {
	repos, w, clean := testutil.SetupReposAndWorkspace(t, map[string][]testutil.Content{
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
				UpdateFunc: func(path string) error {
					// Create symlink in the upstream repo.
					return os.Symlink(filepath.Join(path, "subpkg"),
						filepath.Join(path, "subpkg-sym"))
				},
			},
		},
	})
	defer clean()
	upstreamRepo := repos[testutil.Upstream]

	destinationDir := filepath.Join(w.WorkspaceDirectory, upstreamRepo.RepoName)
	err := Command{
		Git: &kptfilev1.Git{
			Repo:      upstreamRepo.RepoDirectory,
			Directory: "/",
			Ref:       "master",
		},
		Destination: destinationDir,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	w.PackageDir = upstreamRepo.RepoName

	expectedPkg := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
				WithUpstreamLockRef(testutil.Upstream, "/", "master", 0),
		).
		WithResource(pkgbuilder.DeploymentResource).
		WithSubPackages(
			pkgbuilder.NewSubPkg("subpkg").
				WithKptfile().
				WithResource(pkgbuilder.ConfigMapResource),
		)
	expectedPath := expectedPkg.ExpandPkgWithName(t, upstreamRepo.RepoName, testutil.ToReposInfo(repos))

	testutil.KptfileAwarePkgEqual(t, expectedPath, w.FullPackagePath(), true)
}
