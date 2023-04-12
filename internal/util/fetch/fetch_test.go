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

package fetch_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	pkgtesting "github.com/GoogleContainerTools/kpt/internal/pkg/testing"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/fetch"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

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

func createKptfile(workspace *testutil.TestWorkspace, git *kptfilev1.Git, strategy kptfilev1.UpdateStrategyType) error {
	kf := kptfileutil.DefaultKptfile(workspace.PackageDir)
	kf.Upstream = &kptfilev1.Upstream{
		Type:           kptfilev1.GitOrigin,
		Git:            git,
		UpdateStrategy: strategy,
	}
	return kptfileutil.WriteFile(workspace.FullPackagePath(), kf)
}

func setKptfileName(workspace *testutil.TestWorkspace, name string) error {
	kf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, workspace.FullPackagePath())
	if err != nil {
		return err
	}

	kf.Name = name
	err = kptfileutil.WriteFile(workspace.FullPackagePath(), kf)
	return err
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
		Pkg: pkgtesting.CreatePkgOrFail(t, pkgPath),
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "no Kptfile found")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoGit(t *testing.T) {
	_, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, nil, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "kptfile upstream doesn't have git information")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failEmptyRepo(t *testing.T) {
	_, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1.Git{
		Repo:      "",
		Directory: "/",
		Ref:       "main",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "must specify repo")
}

// TestCommand_Run_failEmptyRepo verifies that Command fail if not repo is provided.
func TestCommand_Run_failNoRevision(t *testing.T) {
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1.Git{
		Repo:      "file://" + g.RepoDirectory,
		Directory: "/",
		Ref:       "",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
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
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1.Git{
		Repo:      "file://" + g.RepoDirectory,
		Directory: "/",
		Ref:       "master",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1), absPath, false)

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
				APIVersion: kptfilev1.KptFileGVK().GroupVersion().String(),
				Kind:       kptfilev1.KptFileGVK().Kind,
			},
		},
		Upstream: &kptfilev1.Upstream{
			Type: "git",
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      "file://" + g.RepoDirectory,
				Ref:       "master",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: "git",
			Git: &kptfilev1.GitLock{
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
	err := createKptfile(w, &kptfilev1.Git{
		Repo:      g.RepoDirectory,
		Directory: subdir,
		Ref:       "refs/heads/master",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	absPath := filepath.Join(w.WorkspaceDirectory, g.RepoName)
	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset1, subdir), absPath, false)

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
				APIVersion: kptfilev1.KptFileGVK().GroupVersion().String(),
				Kind:       kptfilev1.KptFileGVK().Kind},
		},
		Upstream: &kptfilev1.Upstream{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.Git{
				Directory: subdir,
				Ref:       "refs/heads/master",
				Repo:      g.RepoDirectory,
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: kptfilev1.GitOrigin,
			Git: &kptfilev1.GitLock{
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

	err = createKptfile(w, &kptfilev1.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/heads/exp",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), w.FullPackagePath(), false)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, w.FullPackagePath(), kptfilev1.KptFile{
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
				Repo:      g.RepoDirectory,
				Ref:       "refs/heads/exp",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
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

	err = createKptfile(w, &kptfilev1.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/tags/v2",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	assert.NoError(t, err)

	// verify the cloned contents matches the repository
	g.AssertEqual(t, filepath.Join(g.DatasetDirectory, testutil.Dataset2), w.FullPackagePath(), false)

	// verify the KptFile contains the expected values
	g.AssertKptfile(t, w.FullPackagePath(), kptfilev1.KptFile{
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
			Type: "git",
			Git: &kptfilev1.Git{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
			},
			UpdateStrategy: kptfilev1.ResourceMerge,
		},
		UpstreamLock: &kptfilev1.UpstreamLock{
			Type: "git",
			Git: &kptfilev1.GitLock{
				Directory: "/",
				Repo:      g.RepoDirectory,
				Ref:       "refs/tags/v2",
				Commit:    commit,
			},
		},
	})
}

func TestCommand_Run_subdir_at_tag(t *testing.T) {
	testCases := map[string]struct {
		dir         string
		tag         string
		upstreamPkg *pkgbuilder.RootPkg
	}{
		"reads subdirectory": {
			dir: "/java/expected",
			tag: "java/v2",
			upstreamPkg: pkgbuilder.NewRootPkg().
				WithSubPackages(pkgbuilder.NewSubPkg("java").
					WithResource("deployment").
					WithSubPackages(pkgbuilder.NewSubPkg("expected").
						WithFile("expected.txt", "My kptfile and I should be the only objects"))),
		},
		"reads subdirectory with no leading slash": {
			dir: "java/expected",
			tag: "java/v2",
			upstreamPkg: pkgbuilder.NewRootPkg().
				WithSubPackages(pkgbuilder.NewSubPkg("java").
					WithResource("deployment").
					WithSubPackages(pkgbuilder.NewSubPkg("expected").
						WithFile("expected.txt", "My kptfile and I should be the only objects"))),
		},
		"reads specific subdirectory": {
			dir: "/java/not_expected/java/expected",
			tag: "java/expected/v2",
			upstreamPkg: pkgbuilder.NewRootPkg().
				WithSubPackages(pkgbuilder.NewSubPkg("java").
					WithResource("deployment").
					WithSubPackages(pkgbuilder.NewSubPkg("not_expected").
						WithFile("not_actually_expected.txt", "I should not be present").
						WithSubPackages(pkgbuilder.NewSubPkg("java").
							WithSubPackages(pkgbuilder.NewSubPkg("expected").
								WithFile("expected.txt", "My kptfile and I should be the only objects"))))),
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			expectedName := "expected"
			repos, rw, clean := testutil.SetupReposAndWorkspace(t, map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg:    tc.upstreamPkg,
						Branch: "main",
						Tag:    tc.tag,
					},
				},
			})

			defer clean()

			g := repos[testutil.Upstream]
			err := createKptfile(rw, &kptfilev1.Git{
				Repo:      g.RepoDirectory,
				Directory: tc.dir,
				Ref:       tc.tag,
			}, kptfilev1.ResourceMerge)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			err = setKptfileName(rw, expectedName)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			actualPkg := pkgtesting.CreatePkgOrFail(t, rw.FullPackagePath())
			err = Command{
				Pkg: actualPkg,
			}.Run(fake.CtxWithDefaultPrinter())
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if !g.AssertEqual(t, rw.WorkspaceDirectory, actualPkg.UniquePath.String(), false) {
				t.FailNow()
			}
			expectedPkg := pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, tc.dir, tc.tag, "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, tc.dir, tc.tag, 0),
				).WithFile("expected.txt", "My kptfile and I should be the only objects")
			expectedPath := expectedPkg.ExpandPkgWithName(t, expectedName, testutil.ToReposInfo(repos))
			testutil.KptfileAwarePkgEqual(t, actualPkg.UniquePath.String(), expectedPath, false)
		})
	}
}

func TestCommand_Run_no_subdir_at_valid_tag(t *testing.T) {
	dir := "/java/expected"
	tag := "java/v2"
	expectedName := "expected_dir_is_not_here"
	repos, rw, clean := testutil.SetupReposAndWorkspace(t, map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithSubPackages(pkgbuilder.NewSubPkg("java").
						WithResource("deployment").
						WithSubPackages(pkgbuilder.NewSubPkg("not_expected").
							WithFile("expected.txt", "My kptfile and I should be the only objects"))),
				Branch: "main",
				Tag:    tag,
			},
		},
	})

	defer clean()

	g := repos[testutil.Upstream]
	err := createKptfile(rw, &kptfilev1.Git{
		Repo:      g.RepoDirectory,
		Directory: dir,
		Ref:       tag,
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = setKptfileName(rw, expectedName)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	actualPkg := pkgtesting.CreatePkgOrFail(t, rw.FullPackagePath())
	err = Command{
		Pkg: actualPkg,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "does not exist in")
	assert.Contains(t, err.Error(), g.RepoDirectory)
	assert.Contains(t, err.Error(), dir)
}

func TestCommand_Run_no_subdir_at_invalid_tag(t *testing.T) {
	dir := "/java/expected"
	nonexistentTag := "notjava/v2"
	expectedName := "expected_dir_is_here"
	repos, rw, clean := testutil.SetupReposAndWorkspace(t, map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithSubPackages(pkgbuilder.NewSubPkg("java").
						WithResource("deployment").
						WithSubPackages(pkgbuilder.NewSubPkg(expectedName).
							WithFile("expected.txt", "My kptfile and I should be the only objects"))),
				Branch: "main",
				Tag:    "java/v2",
			},
		},
	})

	defer clean()

	g := repos[testutil.Upstream]
	err := createKptfile(rw, &kptfilev1.Git{
		Repo:      g.RepoDirectory,
		Directory: dir,
		Ref:       nonexistentTag,
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = setKptfileName(rw, expectedName)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	actualPkg := pkgtesting.CreatePkgOrFail(t, rw.FullPackagePath())
	err = Command{
		Pkg: actualPkg,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "unknown revision")
}

func TestCommand_Run_failInvalidRepo(t *testing.T) {
	_, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1.Git{
		Repo:      "foo",
		Directory: "/",
		Ref:       "refs/heads/master",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.Contains(t, err.Error(), "'foo' does not appear to be a git repository") {
		t.FailNow()
	}
}

func TestCommand_Run_failInvalidBranch(t *testing.T) {
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/heads/foo",
	}, kptfilev1.ResourceMerge)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
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
}

func TestCommand_Run_failInvalidTag(t *testing.T) {
	g, w, clean := setupWorkspace(t)
	defer clean()

	err := createKptfile(w, &kptfilev1.Git{
		Repo:      g.RepoDirectory,
		Directory: "/",
		Ref:       "refs/tags/foo",
	}, kptfilev1.FastForward)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	err = Command{
		Pkg: pkgtesting.CreatePkgOrFail(t, w.FullPackagePath()),
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
}
