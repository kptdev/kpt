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

package update_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	pkgtest "github.com/GoogleContainerTools/kpt/internal/pkg/testing"
	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/update"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

const (
	kptRepo         = "github.com/GoogleContainerTools/kpt"
	masterBranch    = "master"
	testPackageName = "test-package"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

// TestCommand_Run_noRefChanges updates a package without specifying a new ref.
// - Get a package using  a branch ref
// - Modify upstream with new content
// - Update the local package to fetch the upstream content
func TestCommand_Run_noRefChanges(t *testing.T) {
	for i := range kptfilev1.UpdateStrategies {
		strategy := kptfilev1.UpdateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						{
							Data:   testutil.Dataset1,
							Branch: masterBranch,
						},
						{
							Data: testutil.Dataset2,
						},
					},
				},
			}
			defer g.Clean()
			if !g.Init() {
				return
			}
			upstreamRepo := g.Repos[testutil.Upstream]

			// Update the local package
			if !assert.NoError(t, Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Strategy: strategy,
			}.Run(fake.CtxWithDefaultPrinter())) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(testutil.Dataset2, true) {
				return
			}
			commit, err := upstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(upstreamRepo.RepoName, commit, masterBranch,
				strategy) {
				return
			}
		})
	}
}

func TestCommand_Run_subDir(t *testing.T) {
	for i := range kptfilev1.UpdateStrategies {
		strategy := kptfilev1.UpdateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						{
							Data:   testutil.Dataset1,
							Branch: masterBranch,
						},
						{
							Tag:  "v1.2",
							Data: testutil.Dataset2,
						},
					},
				},
				GetSubDirectory: "java",
			}
			defer g.Clean()
			if !g.Init() {
				return
			}
			upstreamRepo := g.Repos[testutil.Upstream]

			// Update the local package
			if !assert.NoError(t, Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Ref:      "v1.2",
				Strategy: strategy,
			}.Run(fake.CtxWithDefaultPrinter())) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(filepath.Join(testutil.Dataset2, "java"), true) {
				return
			}
			commit, err := upstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(g.GetSubDirectory, commit, "v1.2",
				strategy) {
				return
			}
		})
	}
}

func TestCommand_Run_noChanges(t *testing.T) {
	updates := []struct {
		updater kptfilev1.UpdateStrategyType
		err     string
	}{
		{kptfilev1.FastForward, ""},
		{kptfilev1.ForceDeleteReplace, ""},
		// {AlphaGitPatch, "no updates"},
		{kptfilev1.ResourceMerge, ""},
	}
	for i := range updates {
		u := updates[i]
		t.Run(string(u.updater), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						{
							Data:   testutil.Dataset1,
							Branch: masterBranch,
						},
					},
				},
			}
			defer g.Clean()
			if !g.Init() {
				return
			}
			upstreamRepo := g.Repos[testutil.Upstream]

			// Update the local package
			err := Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Strategy: u.updater,
			}.Run(fake.CtxWithDefaultPrinter())
			if u.err == "" {
				if !assert.NoError(t, err) {
					return
				}
			} else {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), "no updates")
				}
			}

			if !g.AssertLocalDataEquals(testutil.Dataset1, true) {
				return
			}
			commit, err := upstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(upstreamRepo.RepoName, commit, masterBranch, u.updater) {
				return
			}
		})
	}
}

func TestCommand_Run_localPackageChanges(t *testing.T) {
	testCases := map[string]struct {
		strategy        kptfilev1.UpdateStrategyType
		initialUpstream testutil.Content
		updatedUpstream testutil.Content
		updatedLocal    testutil.Content
		expectedLocal   testutil.Content
		expectedErr     string
		expectedCommit  func(writer *testutil.TestSetupManager) (string, error)
	}{
		"update using resource-merge strategy with local changes": {
			strategy: kptfilev1.ResourceMerge,
			initialUpstream: testutil.Content{
				Data:   testutil.Dataset1,
				Branch: masterBranch,
			},
			updatedUpstream: testutil.Content{
				Data: testutil.Dataset2,
			},
			updatedLocal: testutil.Content{
				Data: testutil.Dataset3,
			},
			expectedLocal: testutil.Content{
				Data: testutil.DatasetMerged,
			},
			expectedCommit: func(writer *testutil.TestSetupManager) (string, error) {
				return writer.Repos[testutil.Upstream].GetCommit()
			},
		},
		"update using fast-forward strategy with local changes": {
			strategy: kptfilev1.FastForward,
			initialUpstream: testutil.Content{
				Data:   testutil.Dataset1,
				Branch: masterBranch,
			},
			updatedUpstream: testutil.Content{
				Data: testutil.Dataset2,
			},
			updatedLocal: testutil.Content{
				Data: testutil.Dataset3,
			},
			expectedLocal: testutil.Content{
				Data: testutil.Dataset3,
			},
			expectedErr: "local package files have been modified",
			expectedCommit: func(writer *testutil.TestSetupManager) (string, error) {
				upstreamRepo := writer.Repos[testutil.Upstream]
				f, err := pkg.ReadKptfile(filepath.Join(writer.LocalWorkspace.WorkspaceDirectory, upstreamRepo.RepoName))
				if err != nil {
					return "", err
				}
				return f.UpstreamLock.Git.Commit, nil
			},
		},
		"update using force-delete-replace strategy with local changes": {
			strategy: kptfilev1.ForceDeleteReplace,
			initialUpstream: testutil.Content{
				Data:   testutil.Dataset1,
				Branch: masterBranch,
			},
			updatedUpstream: testutil.Content{
				Data: testutil.Dataset2,
			},
			updatedLocal: testutil.Content{
				Data: testutil.Dataset3,
			},
			expectedLocal: testutil.Content{
				Data: testutil.Dataset2,
			},
			expectedCommit: func(writer *testutil.TestSetupManager) (string, error) {
				return writer.Repos[testutil.Upstream].GetCommit()
			},
		},
		"conflicting field with resource-merge strategy": {
			strategy: kptfilev1.ResourceMerge,
			initialUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: masterBranch,
			},
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("42", "spec", "replicas")),
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("21", "spec", "replicas")),
			},
			expectedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("42", "spec", "replicas")),
			},
			expectedCommit: func(writer *testutil.TestSetupManager) (string, error) {
				return writer.Repos[testutil.Upstream].GetCommit()
			},
		},
		"conflicting field with force-delete-replace strategy": {
			strategy: kptfilev1.ForceDeleteReplace,
			initialUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: masterBranch,
			},
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("42", "spec", "replicas")),
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("21", "spec", "replicas")),
			},
			expectedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource,
						pkgbuilder.SetFieldPath("42", "spec", "replicas")),
			},
			expectedCommit: func(writer *testutil.TestSetupManager) (string, error) {
				return writer.Repos[testutil.Upstream].GetCommit()
			},
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						tc.initialUpstream,
						tc.updatedUpstream,
					},
				},
			}
			defer g.Clean()

			if !reflect.DeepEqual(tc.updatedLocal, testutil.Content{}) {
				g.LocalChanges = []testutil.Content{tc.updatedLocal}
			}

			if !g.Init() {
				t.FailNow()
			}

			// record the expected commit after update
			expectedCommit, err := tc.expectedCommit(g)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// run the command
			err = Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Ref:      masterBranch,
				Strategy: tc.strategy,
			}.Run(fake.CtxWithDefaultPrinter())

			// check the error response
			if tc.expectedErr == "" {
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			} else {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), tc.expectedErr) {
					t.FailNow()
				}
			}

			expectedPath := tc.expectedLocal.Data
			if tc.expectedLocal.Pkg != nil {
				expectedPath = tc.expectedLocal.Pkg.ExpandPkgWithName(t,
					g.LocalWorkspace.PackageDir, testutil.ToReposInfo(g.Repos))
			}

			if !g.AssertLocalDataEquals(expectedPath, true) {
				t.FailNow()
			}
			if !g.AssertKptfile(g.Repos[testutil.Upstream].RepoName, expectedCommit, masterBranch,
				tc.strategy) {
				t.FailNow()
			}
		})
	}
}

// TestCommand_Run_toBranchRef verifies the package contents are set to the contents of the branch
// it was updated to.
func TestCommand_Run_toBranchRef(t *testing.T) {
	for i := range kptfilev1.UpdateStrategies {
		strategy := kptfilev1.UpdateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						{
							Data:   testutil.Dataset1,
							Branch: masterBranch,
						},
						{
							Data:   testutil.Dataset2,
							Branch: "exp", CreateBranch: true,
						},
						{
							Data:   testutil.Dataset3,
							Branch: masterBranch,
						},
					},
				},
			}
			defer g.Clean()
			if !g.Init() {
				return
			}
			upstreamRepo := g.Repos[testutil.Upstream]

			// Update the local package
			if !assert.NoError(t, Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Strategy: strategy,
				Ref:      "exp",
			}.Run(fake.CtxWithDefaultPrinter())) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(testutil.Dataset2, true) {
				return
			}

			if !assert.NoError(t, upstreamRepo.CheckoutBranch("exp", false)) {
				return
			}
			commit, err := upstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(upstreamRepo.RepoName, commit, "exp",
				strategy) {
				return
			}
		})
	}
}

func TestCommand_Run_toBranchRefWithSubpkgs(t *testing.T) {
	testCases := map[string]struct {
		strategy      kptfilev1.UpdateStrategyType
		updateRef     string
		reposChanges  map[string][]testutil.Content
		updateFunc    func(g *testutil.TestSetupManager)
		updateSubPkg  string
		expectedLocal *pkgbuilder.RootPkg
	}{
		"update with single subpkg from same repo": {
			strategy:  kptfilev1.ResourceMerge,
			updateRef: "subpkg-update",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource,
									pkgbuilder.SetFieldPath("42", "spec", "replicas"),
								),
							),
						Branch: "subpkg-update", CreateBranch: true,
					},
				},
			},
			updateFunc: func(g *testutil.TestSetupManager) {
				g.GetSubPkg("test", testutil.Upstream, "subpkg")
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithUpstreamRef(testutil.Upstream, "/", "subpkg-update", "resource-merge").
					WithUpstreamLockRef(testutil.Upstream, "/", "subpkg-update", 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(pkgbuilder.NewKptfile()).
						WithResource(pkgbuilder.DeploymentResource,
							pkgbuilder.SetFieldPath("42", "spec", "replicas")),
					pkgbuilder.NewSubPkg("test").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef(testutil.Upstream, "/subpkg", "subpkg-update", "resource-merge").
								WithUpstreamLockRef(testutil.Upstream, "/subpkg", "subpkg-update", 1),
						).
						WithResource(pkgbuilder.DeploymentResource,
							pkgbuilder.SetFieldPath("42", "spec", "replicas")),
				),
		},
		"update strategy with single subpkg from same repo": {
			strategy:  kptfilev1.FastForward,
			updateRef: "subpkg-update",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource,
									pkgbuilder.SetFieldPath("42", "spec", "replicas"),
								),
							),
						Branch: "subpkg-update", CreateBranch: true,
					},
				},
			},
			updateFunc: func(g *testutil.TestSetupManager) {
				g.GetSubPkg("test", testutil.Upstream, "subpkg")
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithUpstreamRef(testutil.Upstream, "/", "subpkg-update", "fast-forward").
					WithUpstreamLockRef(testutil.Upstream, "/", "subpkg-update", 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(pkgbuilder.NewKptfile()).
						WithResource(pkgbuilder.DeploymentResource,
							pkgbuilder.SetFieldPath("42", "spec", "replicas")),
					pkgbuilder.NewSubPkg("test").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef(testutil.Upstream, "/subpkg", "subpkg-update", "fast-forward").
								WithUpstreamLockRef(testutil.Upstream, "/subpkg", "subpkg-update", 1),
						).
						WithResource(pkgbuilder.DeploymentResource,
							pkgbuilder.SetFieldPath("42", "spec", "replicas")),
				),
		},
		"update with single subpkg from different repo": {
			strategy:  kptfilev1.ResourceMerge,
			updateRef: "subpkg-update",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.SecretResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.ConfigMapResource),
							),
						Branch: "subpkg-update", CreateBranch: true,
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource,
									pkgbuilder.SetFieldPath("42", "spec", "replicas"),
								),
							),
						Branch: "subpkg-update", CreateBranch: true,
					}},
			},
			updateFunc: func(g *testutil.TestSetupManager) {
				g.GetSubPkg("fooSubPkg", "foo", "subpkg")
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithUpstreamRef(testutil.Upstream, "/", "subpkg-update", "resource-merge").
					WithUpstreamLockRef(testutil.Upstream, "/", "subpkg-update", 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(pkgbuilder.NewKptfile()).
						WithResource(pkgbuilder.ConfigMapResource),
					pkgbuilder.NewSubPkg("fooSubPkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/subpkg", "master", "resource-merge").
								WithUpstreamLockRef("foo", "/subpkg", "master", 0),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		"update with multiple subpkgs from same repo": {
			strategy:  kptfilev1.ResourceMerge,
			updateRef: "subpkg-update",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg1").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.SecretResource).
								WithSubPackages(
									pkgbuilder.NewSubPkg("nestedOne").
										WithKptfile(pkgbuilder.NewKptfile()).
										WithResource(pkgbuilder.DeploymentResource),
									pkgbuilder.NewSubPkg("nestedTwo").
										WithKptfile(pkgbuilder.NewKptfile()).
										WithResource(pkgbuilder.DeploymentResource),
								),
								pkgbuilder.NewSubPkg("subpkg2").WithKptfile(pkgbuilder.NewKptfile()).
									WithResource(pkgbuilder.SecretResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("nestedThree").
											WithKptfile(pkgbuilder.NewKptfile()).
											WithResource(pkgbuilder.DeploymentResource),
										pkgbuilder.NewSubPkg("nestedFour").
											WithKptfile(pkgbuilder.NewKptfile()).
											WithResource(pkgbuilder.DeploymentResource),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(pkgbuilder.NewSubPkg("subpkg1").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.SecretResource).
								WithSubPackages(
									pkgbuilder.NewSubPkg("nestedOne").
										WithKptfile(pkgbuilder.NewKptfile()).
										WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("42", "spec", "replicas")),
									pkgbuilder.NewSubPkg("nestedTwo").
										WithKptfile(pkgbuilder.NewKptfile()).
										WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("43", "spec", "replicas")),
								),
								pkgbuilder.NewSubPkg("subpkg2").WithKptfile(pkgbuilder.NewKptfile()).
									WithResource(pkgbuilder.SecretResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("nestedThree").
											WithKptfile(pkgbuilder.NewKptfile()).
											WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("44", "spec", "replicas")),
										pkgbuilder.NewSubPkg("nestedFour").
											WithKptfile(pkgbuilder.NewKptfile()).
											WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("45", "spec", "replicas")),
									),
							),
						Branch: "subpkg-update", CreateBranch: true,
					},
				},
			},
			updateFunc: func(g *testutil.TestSetupManager) {
				g.GetSubPkg("subpkgvariant", testutil.Upstream, "subpkg1")
				g.GetSubPkg("subpkgvariant/nestedTwoVariant", testutil.Upstream, "subpkg1/nestedTwo")
				g.GetSubPkg("subpkgvariant/nestedThreeVariant", testutil.Upstream, "subpkg2/nestedThree")
			},
			updateSubPkg: "subpkgvariant",
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", "master", 0),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg1").
						WithKptfile(pkgbuilder.NewKptfile()).
						WithResource(pkgbuilder.SecretResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("nestedOne").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
							pkgbuilder.NewSubPkg("nestedTwo").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
						),
					pkgbuilder.NewSubPkg("subpkg2").WithKptfile(pkgbuilder.NewKptfile()).
						WithResource(pkgbuilder.SecretResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("nestedThree").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
							pkgbuilder.NewSubPkg("nestedFour").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
						),
					pkgbuilder.NewSubPkg("subpkgvariant").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef(testutil.Upstream, "/subpkg1", "subpkg-update", "resource-merge").
								WithUpstreamLockRef(testutil.Upstream, "/subpkg1", "subpkg-update", 1),
						).
						WithResource(pkgbuilder.SecretResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("nestedOne").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("42", "spec", "replicas")),
							pkgbuilder.NewSubPkg("nestedTwo").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("43", "spec", "replicas")),
							pkgbuilder.NewSubPkg("nestedTwoVariant").
								WithKptfile(
									pkgbuilder.NewKptfile().
										WithUpstreamRef(testutil.Upstream, "/subpkg1/nestedTwo", "subpkg-update", "resource-merge").
										WithUpstreamLockRef(testutil.Upstream, "/subpkg1/nestedTwo", "subpkg-update", 1),
								).
								WithResource(pkgbuilder.DeploymentResource, pkgbuilder.SetFieldPath("43", "spec", "replicas")),
							pkgbuilder.NewSubPkg("nestedThreeVariant").
								WithKptfile(
									pkgbuilder.NewKptfile().
										WithUpstreamRef(testutil.Upstream, "/subpkg2/nestedThree", "master", "resource-merge").
										WithUpstreamLockRef(testutil.Upstream, "/subpkg2/nestedThree", "master", 0),
								).
								WithResource(pkgbuilder.DeploymentResource),
						),
				),
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T:            t,
				ReposChanges: tc.reposChanges,
			}
			defer g.Clean()

			if !g.Init() {
				return
			}

			// perform any additional updates to local pkg
			if tc.updateFunc != nil {
				tc.updateFunc(g)
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, path.Join(g.LocalWorkspace.FullPackagePath(), tc.updateSubPkg)),
				Strategy: tc.strategy,
				Ref:      tc.updateRef,
			}.Run(fake.CtxWithDefaultPrinter())) {
				return
			}
			expectedPath := tc.expectedLocal.ExpandPkgWithName(t,
				g.LocalWorkspace.PackageDir, testutil.ToReposInfo(g.Repos))
			testutil.KptfileAwarePkgEqual(t, expectedPath, g.LocalWorkspace.FullPackagePath(), true)
		})
	}
}

// TestCommand_Run_toTagRef verifies the package contents are set to the contents of the tag
// it was updated to.
func TestCommand_Run_toTagRef(t *testing.T) {
	for i := range kptfilev1.UpdateStrategies {
		strategy := kptfilev1.UpdateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						{
							Data:   testutil.Dataset1,
							Branch: masterBranch,
						},
						{
							Data: testutil.Dataset2,
							Tag:  "v1.0",
						},
						{
							Data: testutil.Dataset3,
						},
					},
				},
			}
			defer g.Clean()
			if !g.Init() {
				return
			}
			upstreamRepo := g.Repos[testutil.Upstream]

			// Update the local package
			if !assert.NoError(t, Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Strategy: strategy,
				Ref:      "v1.0",
			}.Run(fake.CtxWithDefaultPrinter())) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(testutil.Dataset2, true) {
				return
			}

			if !assert.NoError(t, upstreamRepo.CheckoutBranch("v1.0", false)) {
				return
			}
			commit, err := upstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(upstreamRepo.RepoName, commit, "v1.0",
				strategy) {
				return
			}
		})
	}
}

// TestCommand_ResourceMerge_NonKRMUpdates tests if the local non KRM files are updated
func TestCommand_ResourceMerge_NonKRMUpdates(t *testing.T) {
	strategies := []kptfilev1.UpdateStrategyType{kptfilev1.ResourceMerge}
	for i := range strategies {
		strategy := strategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						{
							Data:   testutil.Dataset1,
							Branch: masterBranch,
						},
						{
							Data: testutil.Dataset5,
							Tag:  "v1.0",
						},
					},
				},
			}
			defer g.Clean()
			if !g.Init() {
				t.FailNow()
			}
			upstreamRepo := g.Repos[testutil.Upstream]

			// Update the local package
			if !assert.NoError(t, Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Strategy: strategy,
				Ref:      "v1.0",
			}.Run(fake.CtxWithDefaultPrinter())) {
				t.FailNow()
			}

			// Expect the local package to have Dataset5
			if !g.AssertLocalDataEquals(testutil.Dataset5, true) {
				t.FailNow()
			}

			if !assert.NoError(t, upstreamRepo.CheckoutBranch("v1.0", false)) {
				t.FailNow()
			}
			commit, err := upstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !g.AssertKptfile(upstreamRepo.RepoName, commit, "v1.0",
				strategy) {
				t.FailNow()
			}
		})
	}
}

func TestCommand_Run_noUpstreamReference(t *testing.T) {
	_, w, clean := testutil.SetupReposAndWorkspace(t, map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Branch: masterBranch,
			},
		},
	})
	defer clean()

	w.PackageDir = testPackageName
	kf := kptfileutil.DefaultKptfile(testPackageName)
	testutil.AddKptfileToWorkspace(t, w, kf)

	// Update the local package
	err := Command{
		Pkg: pkgtest.CreatePkgOrFail(t, w.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())

	assert.Contains(t, err.Error(), "must have an upstream reference")

}

// TestCommand_Run_failInvalidPath verifies Run fails if the path is invalid
func TestCommand_Run_failInvalidPath(t *testing.T) {
	for i := range kptfilev1.UpdateStrategies {
		strategy := kptfilev1.UpdateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			path := filepath.Join("fake", "path")
			err := Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, path),
				Strategy: strategy,
			}.Run(fake.CtxWithDefaultPrinter())
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "no such file or directory")
			}
		})
	}
}

func TestCommand_Run_badUpstreamLock(t *testing.T) {
	testCases := map[string]struct {
		commit           string
		repo             string
		expectedErrorMsg string
	}{
		"bad commit": {
			commit:           "fake",
			repo:             testutil.Upstream,
			expectedErrorMsg: "unknown revision or path"},
		"bad repo": {
			repo:             "fake",
			expectedErrorMsg: "does not appear to be a git repository"},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			upstreamContent := testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: masterBranch,
			}

			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						upstreamContent,
					},
				},
			}

			defer g.Clean()

			if !g.Init() {
				return
			}

			// If tc.repo exists in the testbed, use its repo directory for the UpstreamLock.
			repoDir := tc.repo
			if repo, repoExists := g.Repos[tc.repo]; repoExists {
				repoDir = repo.RepoDirectory
			}
			localChanges := []testutil.Content{{
				Pkg: pkgbuilder.NewRootPkg().WithKptfile(pkgbuilder.NewKptfile().
					WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
					WithUpstreamLock(repoDir, "/", masterBranch, tc.commit),
				),
			}}
			err := testutil.UpdateGitDir(g.T, testutil.Local, g.LocalWorkspace, localChanges, g.Repos)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// Update the local package.
			err = Command{
				Pkg: pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
			}.Run(fake.CtxWithDefaultPrinter())

			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
			}
		})
	}
}

// TestCommand_Run_failInvalidRef verifies Run fails if the ref is invalid
func TestCommand_Run_failInvalidRef(t *testing.T) {
	for i := range kptfilev1.UpdateStrategies {
		strategy := kptfilev1.UpdateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T: t,
				ReposChanges: map[string][]testutil.Content{
					testutil.Upstream: {
						{
							Data:   testutil.Dataset1,
							Branch: masterBranch,
						},
						{
							Data: testutil.Dataset2,
						},
					},
				},
			}
			defer g.Clean()
			if !g.Init() {
				return
			}

			err := Command{
				Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
				Ref:      "exp",
				Strategy: strategy,
			}.Run(fake.CtxWithDefaultPrinter())
			if !assert.Error(t, err) {
				return
			}
			assert.Contains(t, err.Error(), "unknown revision or path not in the working tree")

			if !g.AssertLocalDataEquals(testutil.Dataset1, true) {
				return
			}
		})
	}
}

func TestCommand_Run_manualChange(t *testing.T) {
	g := &testutil.TestSetupManager{
		T: t,
		ReposChanges: map[string][]testutil.Content{
			testutil.Upstream: {
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch: masterBranch,
					Tag:    "v1",
				},
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.ConfigMapResource),
					Tag: "v2",
				},
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.SecretResource),
					Tag: "v3",
				},
			},
		},
		GetRef: "v1",
		LocalChanges: []testutil.Content{
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", "v3", "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", "v1", 0),
					).
					WithResource(pkgbuilder.DeploymentResource),
			},
		},
	}
	defer g.Clean()
	if !g.Init() {
		return
	}

	err := Command{
		Pkg: pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	expLocal := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstreamRef(testutil.Upstream, "/", "v3", "resource-merge").
				WithUpstreamLockRef(testutil.Upstream, "/", "v3", 2),
		).
		WithResource(pkgbuilder.SecretResource)
	expectedPath := expLocal.ExpandPkgWithName(t,
		g.LocalWorkspace.PackageDir, testutil.ToReposInfo(g.Repos))

	testutil.KptfileAwarePkgEqual(t, expectedPath, g.LocalWorkspace.FullPackagePath(), true)
}

func TestCommand_Run_symlinks(t *testing.T) {
	g := &testutil.TestSetupManager{
		T: t,
		ReposChanges: map[string][]testutil.Content{
			testutil.Upstream: {
				{
					Branch: masterBranch,
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
				},
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.SecretResource).
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
		},
		GetRef: masterBranch,
	}
	defer g.Clean()
	if !g.Init() {
		return
	}
	upstreamRepo := g.Repos[testutil.Upstream]

	err := Command{
		Pkg: pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	expectedPkg := pkgbuilder.NewRootPkg().
		WithKptfile(
			pkgbuilder.NewKptfile().
				WithUpstreamRef(testutil.Upstream, "/", "master", "resource-merge").
				WithUpstreamLockRef(testutil.Upstream, "/", "master", 1),
		).
		WithResource(pkgbuilder.SecretResource).
		WithSubPackages(
			pkgbuilder.NewSubPkg("subpkg").
				WithKptfile().
				WithResource(pkgbuilder.ConfigMapResource),
		)
	expectedPath := expectedPkg.ExpandPkgWithName(t, upstreamRepo.RepoName, testutil.ToReposInfo(g.Repos))

	testutil.KptfileAwarePkgEqual(t, expectedPath, g.LocalWorkspace.FullPackagePath(), true)
}

func TestCommand_Run_badStrategy(t *testing.T) {
	strategy := kptfilev1.UpdateStrategyType("foo")

	// Setup the test upstream and local packages
	g := &testutil.TestSetupManager{
		T: t,
		ReposChanges: map[string][]testutil.Content{
			testutil.Upstream: {
				{
					Data:   testutil.Dataset1,
					Branch: masterBranch,
				},
				{
					Data: testutil.Dataset2,
				},
			},
		},
	}
	defer g.Clean()
	if !g.Init() {
		return
	}

	// Update the local package
	err := Command{
		Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
		Strategy: strategy,
	}.Run(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err, strategy) {
		return
	}
	assert.Contains(t, err.Error(), "unrecognized update strategy")
}

func TestCommand_Run_local_subpackages(t *testing.T) {
	testCases := []struct {
		name            string
		reposChanges    map[string][]testutil.Content
		updatedLocal    testutil.Content
		expectedResults []resultForStrategy
	}{
		{
			name: "update fetches any new subpackages",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile().
									WithSubPackages(
										pkgbuilder.NewSubPkg("nestedbar").
											WithKptfile(),
									),
								pkgbuilder.NewSubPkg("zork").
									WithKptfile(),
							),
					},
				},
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
						kptfilev1.FastForward,
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile().
								WithSubPackages(
									pkgbuilder.NewSubPkg("nestedbar").
										WithKptfile(),
								),
							pkgbuilder.NewSubPkg("zork").
								WithKptfile(),
						),
				},
			},
		},
		{
			name: "local changes and a noop update",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(),
							),
						Branch: masterBranch,
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(),
						pkgbuilder.NewSubPkg("zork").
							WithKptfile(),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 0),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(),
							pkgbuilder.NewSubPkg("zork").
								WithKptfile(),
						),
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.FastForward,
					},
					expectedErrMsg: "use a different update --strategy",
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 0),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(),
						),
				},
			},
		},
		{
			name: "non-overlapping additions in both upstream and local",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(),
								pkgbuilder.NewSubPkg("zork").
									WithKptfile(),
							),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(),
						pkgbuilder.NewSubPkg("abc").
							WithKptfile(),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(),
							pkgbuilder.NewSubPkg("zork").
								WithKptfile(),
							pkgbuilder.NewSubPkg("abc").
								WithKptfile(),
						),
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.FastForward,
					},
					expectedErrMsg: "use a different update --strategy",
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(),
							pkgbuilder.NewSubPkg("zork").
								WithKptfile(),
						),
				},
			},
		},
		{
			name: "overlapping additions in both upstream and local",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(),
								pkgbuilder.NewSubPkg("abc").
									WithKptfile(),
							),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(),
						pkgbuilder.NewSubPkg("abc").
							WithKptfile(),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
					},
					expectedErrMsg: "subpackage \"abc\" added in both upstream and local",
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.FastForward,
					},
					expectedErrMsg: "use a different update --strategy",
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(),
							pkgbuilder.NewSubPkg("abc").
								WithKptfile(),
						),
				},
			},
		},
		{
			name: "subpackages deleted in upstream",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(),
					},
				},
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
						kptfilev1.FastForward,
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						),
				},
			},
		},
		{
			name: "multiple layers of subpackages added in upstream",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile().
									WithSubPackages(
										pkgbuilder.NewSubPkg("nestedbar").
											WithKptfile(),
									),
							),
					},
				},
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
						kptfilev1.FastForward,
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile().
								WithSubPackages(
									pkgbuilder.NewSubPkg("nestedbar").
										WithKptfile(),
								),
						),
				},
			},
		},
		{
			name: "removed Kptfile from upstream",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(pkgbuilder.NewKptfile()).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithResource(pkgbuilder.DeploymentResource),
							),
					},
				},
			},
			expectedResults: []resultForStrategy{
				// TODO(mortent): Revisit this. Not clear that the Kptfile
				// shouldn't be deleted here since it doesn't really have any
				// local changes.
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource),
						),
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.FastForward,
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithResource(pkgbuilder.DeploymentResource),
						),
				},
			},
		},
		{
			name: "kptfile added only on local",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithResource(pkgbuilder.DeploymentResource).
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.DeploymentResource),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource).
								WithResource(pkgbuilder.ConfigMapResource),
						),
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.FastForward,
					},
					expectedErrMsg: "use a different update --strategy",
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithResource(pkgbuilder.DeploymentResource).
								WithResource(pkgbuilder.ConfigMapResource),
						),
				},
			},
		},
		{
			name: "subpackage deleted from upstream but is unchanged in local",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(pkgbuilder.NewKptfile()).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(),
					},
				},
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
						kptfilev1.FastForward,
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						),
				},
			},
		},
		{
			name: "subpackage deleted from upstream but has local changes",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(pkgbuilder.NewKptfile()).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(pkgbuilder.NewKptfile()).
							WithResource(pkgbuilder.DeploymentResource,
								pkgbuilder.SetFieldPath("34", "spec", "replicas")),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithResource(pkgbuilder.DeploymentResource,
									pkgbuilder.SetFieldPath("34", "spec", "replicas")),
						),
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.FastForward,
					},
					expectedErrMsg: "use a different update --strategy",
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						),
				},
			},
		},
		{
			name: "upstream package doesn't need to have a Kptfile in the root",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(pkgbuilder.NewKptfile()),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(pkgbuilder.NewKptfile()),
							),
					},
				},
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
						kptfilev1.FastForward,
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithResource(pkgbuilder.DeploymentResource).
						WithResource(pkgbuilder.ConfigMapResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()),
						),
				},
			},
		},
		{
			name: "non-krm files updated in upstream",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithFile("data.txt", "initial content").
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(pkgbuilder.NewKptfile()).
									WithFile("information", "first version"),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithFile("data.txt", "updated content").
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(pkgbuilder.NewKptfile()).
									WithFile("information", "second version"),
							),
					},
				},
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
						kptfilev1.FastForward,
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithFile("data.txt", "updated content").
						WithSubPackages(
							pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile()).
								WithFile("information", "second version"),
						),
				},
			},
		},
		{
			name: "non-krm files updated in both upstream and local",
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithFile("data.txt", "initial content"),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(pkgbuilder.NewKptfile()).
							WithFile("data.txt", "updated content"),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile()).
					WithFile("data.txt", "local content"),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ResourceMerge,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithFile("data.txt", "local content"),
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.FastForward,
					},
					expectedErrMsg: "use a different update --strategy",
				},
				{
					strategies: []kptfilev1.UpdateStrategyType{
						kptfilev1.ForceDeleteReplace,
					},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("upstream", "/", masterBranch, "PLACEHOLDER").
								WithUpstreamLockRef("upstream", "/", masterBranch, 1),
						).
						WithFile("data.txt", "updated content"),
				},
			},
		},
	}

	for i := range testCases {
		test := testCases[i]
		strategies := findStrategiesForTestCase(test.expectedResults)
		for i := range strategies {
			strategy := strategies[i]
			t.Run(fmt.Sprintf("%s#%s", test.name, string(strategy)), func(t *testing.T) {
				g := &testutil.TestSetupManager{
					T:            t,
					ReposChanges: test.reposChanges,
				}
				defer g.Clean()
				if test.updatedLocal.Pkg != nil {
					g.LocalChanges = []testutil.Content{
						test.updatedLocal,
					}
				}
				if !g.Init() {
					return
				}

				err := Command{
					Pkg:      pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
					Strategy: strategy,
				}.Run(fake.CtxWithDefaultPrinter())

				result := findExpectedResultForStrategy(test.expectedResults, strategy)

				if result.expectedErrMsg != "" {
					if !assert.Error(t, err) {
						t.FailNow()
					}
					assert.Contains(t, err.Error(), result.expectedErrMsg)
					return
				}
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				// Format the Kptfiles so we can diff the output without
				// formatting issues.
				rw := &kio.LocalPackageReadWriter{
					NoDeleteFiles:     true,
					PackagePath:       g.LocalWorkspace.FullPackagePath(),
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

				expectedPath := result.expectedLocal.ExpandPkgWithName(t,
					g.LocalWorkspace.PackageDir, testutil.ToReposInfo(g.Repos))
				kf, err := pkg.ReadKptfile(expectedPath)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				kf.Upstream.UpdateStrategy = strategy
				err = kptfileutil.WriteFile(expectedPath, kf)
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				testutil.KptfileAwarePkgEqual(t, expectedPath, g.LocalWorkspace.FullPackagePath(), true)
			})
		}
	}
}

type resultForStrategy struct {
	strategies     []kptfilev1.UpdateStrategyType
	expectedLocal  *pkgbuilder.RootPkg
	expectedErrMsg string
}

func findStrategiesForTestCase(expectedResults []resultForStrategy) []kptfilev1.UpdateStrategyType {
	var strategies []kptfilev1.UpdateStrategyType
	for _, er := range expectedResults {
		strategies = append(strategies, er.strategies...)
	}
	return strategies
}

func findExpectedResultForStrategy(strategyResults []resultForStrategy,
	strategy kptfilev1.UpdateStrategyType) resultForStrategy {
	for _, sr := range strategyResults {
		for _, s := range sr.strategies {
			if s == strategy {
				return sr
			}
		}
	}
	panic(fmt.Errorf("unknown strategy %s", string(strategy)))
}

func TestRun_remote_subpackages(t *testing.T) {
	testCases := map[string]struct {
		reposChanges   map[string][]testutil.Content
		updatedLocal   testutil.Content
		expectedLocal  *pkgbuilder.RootPkg
		expectedErrMsg string
	}{
		"subpackages are updated based on the version specified in their Kptfile": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge"),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "resource-merge"),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Tag: "v1.0",
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
								WithUpstreamLockRef("foo", "/", "v1.0", 1),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"subpackage with changes can not be updated with fast-forward strategy": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "fast-forward"),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "fast-forward"),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Tag: "v1.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", masterBranch, "fast-forward").
									WithUpstreamLockRef("foo", "/", masterBranch, 0),
							).
							WithResource(pkgbuilder.DeploymentResource,
								pkgbuilder.SetFieldPath("34", "spec", "replicas")),
					),
			},
			expectedErrMsg: "use a different update --strategy",
		},
		"subpackage with changes can be updated with resource-merge": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge"),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "resource-merge"),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource,
								pkgbuilder.SetFieldPath("zork", "spec", "foo")),
						Tag: "v1.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
									WithUpstreamLockRef("foo", "/", masterBranch, 0),
							).
							WithResource(pkgbuilder.DeploymentResource,
								pkgbuilder.SetFieldPath("34", "spec", "replicas")),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
								WithUpstreamLockRef("foo", "/", "v1.0", 1),
						).
						WithResource(pkgbuilder.DeploymentResource,
							pkgbuilder.SetFieldPath("34", "spec", "replicas"),
							pkgbuilder.SetFieldPath("zork", "spec", "foo"),
						),
				),
		},
		"multiple layers of remote packages": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge"),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge"),
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
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("bar", "/", masterBranch, "fast-forward"),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("bar", "/", masterBranch, "resource-merge"),
									),
							),
					},
				},
				"bar": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: masterBranch,
					}, {
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
								WithUpstreamLockRef("foo", "/", masterBranch, 1),
						).
						WithResource(pkgbuilder.ConfigMapResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(
									pkgbuilder.NewKptfile().
										WithUpstreamRef("bar", "/", masterBranch, "resource-merge").
										WithUpstreamLockRef("bar", "/", masterBranch, 1),
								).
								WithResource(pkgbuilder.ConfigMapResource),
						),
				),
		},
		"remote subpackages distributed with the parent package": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.DeploymentResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("bar").
											WithKptfile(
												pkgbuilder.NewKptfile().
													WithUpstreamRef("bar", "/", masterBranch, "fast-forward").
													WithUpstreamLockRef("bar", "/", masterBranch, 0),
											).
											WithResource(pkgbuilder.DeploymentResource),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 1),
									).
									WithResource(pkgbuilder.ConfigMapResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("bar").
											WithKptfile(
												pkgbuilder.NewKptfile().
													WithUpstreamRef("bar", "/", masterBranch, "resource-merge").
													WithUpstreamLockRef("bar", "/", masterBranch, 1),
											).
											WithResource(pkgbuilder.ConfigMapResource),
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
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("bar", "/", masterBranch, "fast-forward").
											WithUpstreamLockRef("bar", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("bar", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("bar", "/", masterBranch, 1),
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
						Branch: masterBranch,
					}, {
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
								WithUpstreamLockRef("foo", "/", masterBranch, 1),
						).
						WithResource(pkgbuilder.ConfigMapResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(
									pkgbuilder.NewKptfile().
										WithUpstreamRef("bar", "/", masterBranch, "resource-merge").
										WithUpstreamLockRef("bar", "/", masterBranch, 1),
								).
								WithResource(pkgbuilder.ConfigMapResource),
						),
				),
		},
		"subpackage with resource-merge strategy updated in both local and upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
											WithUpstreamLockRef("foo", "/", "v1.0", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v2.0", "resource-merge").
											WithUpstreamLockRef("foo", "/", "v2.0", 1),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
						Tag:    "v1.0",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Tag: "v2.0",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.SecretResource),
						Tag: "v3.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "v3.0", "resource-merge").
									WithUpstreamLockRef("foo", "/", "v3.0", 2),
							).
							WithResource(pkgbuilder.SecretResource),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v3.0", "resource-merge").
								WithUpstreamLockRef("foo", "/", "v3.0", 2),
						).
						WithResource(pkgbuilder.SecretResource),
				),
		},
		"subpackage with force-delete-replace strategy updated in both local and upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "force-delete-replace").
											WithUpstreamLockRef("foo", "/", "v1.0", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v2.0", "force-delete-replace").
											WithUpstreamLockRef("foo", "/", "v2.0", 1),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
						Tag:    "v1.0",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Tag: "v2.0",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.SecretResource),
						Tag: "v3.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "v3.0", "force-delete-replace").
									WithUpstreamLockRef("foo", "/", "v3.0", 2),
							).
							WithResource(pkgbuilder.SecretResource),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v2.0", "force-delete-replace").
								WithUpstreamLockRef("foo", "/", "v2.0", 1),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"remote subpackage deleted from upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "force-delete-replace").
											WithUpstreamLockRef("foo", "/", "v1.0", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
						Tag:    "v1.0",
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		"remote subpackage deleted from upstream, but local has updated package": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
											WithUpstreamLockRef("foo", "/", "v1.0", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
						Tag:    "v1.0",
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Tag: "v2.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
							WithUpstreamLockRef("upstream", "/", masterBranch, 0),
					).
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "v2.0", "resource-merge").
									WithUpstreamLockRef("foo", "/", "v2.0", 1),
							).
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v2.0", "resource-merge").
								WithUpstreamLockRef("foo", "/", "v2.0", 1),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"subpackage with nested remote subpackages deleted from upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("foo").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "main", "resource-merge").
											WithUpstreamLockRef("foo", "/", "main", 0),
									).
									WithResource(pkgbuilder.DeploymentResource).
									WithSubPackages(
										pkgbuilder.NewSubPkg("bar").
											WithKptfile(
												pkgbuilder.NewKptfile().
													WithUpstreamRef("bar", "/", masterBranch, "resource-merge").
													WithUpstreamLockRef("bar", "/", masterBranch, 0),
											).
											WithResource(pkgbuilder.DeploymentResource),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("bar", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("bar", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
						Branch: "main",
					},
				},
				"bar": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: masterBranch,
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		"remote and local subpackages added in local": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
						Tag:    "v1.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
							WithUpstreamLockRef("upstream", "/", masterBranch, 0),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "v1.0", "fast-forward").
									WithUpstreamLockRef("foo", "/", "v1.0", 0),
							).
							WithResource(pkgbuilder.DeploymentResource),
						pkgbuilder.NewSubPkg("localsubpkg").
							WithKptfile().
							WithResource(pkgbuilder.SecretResource),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef("upstream", "/", masterBranch, "resource-merge").
						WithUpstreamLockRef("upstream", "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v1.0", "fast-forward").
								WithUpstreamLockRef("foo", "/", "v1.0", 0),
						).
						WithResource(pkgbuilder.DeploymentResource),
					pkgbuilder.NewSubPkg("localsubpkg").
						WithKptfile().
						WithResource(pkgbuilder.SecretResource),
				),
		},
		"two different remote packages in same path added in upstream and local": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
											WithUpstreamLockRef("foo", "/", "v1.0", 0),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
						Tag:    "v1.0",
					},
				},
				"bar": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: "main",
						Tag:    "v1.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("bar", "/", "v1.0", "resource-merge").
									WithUpstreamLockRef("bar", "/", "v1.0", 0),
							).
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			expectedErrMsg: "package \"subpkg\" added in both upstream and local",
		},
		"remote subpackage added in upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
								WithUpstreamLockRef("foo", "/", masterBranch, 0),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"remote subpackage deleted from local": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.ConfigMapResource).
									WithResource(pkgbuilder.DeploymentResource,
										pkgbuilder.SetFieldPath("45", "spec", "replicas")),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 0),
					).
					WithResource(pkgbuilder.DeploymentResource),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		"remote subpackage has gone from unfetched to fetched state in upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge"),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 1),
									).
									WithResource(pkgbuilder.DeploymentResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
					},
				},
			},
			expectedErrMsg: "no origin available for package",
		},
		"remote subpackage has gone from fetched to unfetched state in upstream": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge"),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
				},
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
								WithUpstreamLockRef("foo", "/", masterBranch, 0),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
		"fast-forward updates can be used even if subpackages has changes": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.SecretResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", masterBranch, "fast-forward").
							WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 0),
					).
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
									WithUpstreamLockRef("foo", "/", masterBranch, 0),
							).
							WithResource(pkgbuilder.ConfigMapResource,
								pkgbuilder.SetAnnotation("foo", "bar")),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", masterBranch, "fast-forward").
						WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.SecretResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
								WithUpstreamLockRef("foo", "/", masterBranch, 0),
						).
						WithResource(pkgbuilder.ConfigMapResource,
							pkgbuilder.SetAnnotation("foo", "bar")),
				),
		},
		"Kptfiles in unfetched subpackages are merged": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "resource-merge"),
									),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
											WithPipeline(
												pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:latest"),
											),
									),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
						Tag:    "v1.0",
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 0),
					).
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
									WithUpstreamLockRef("foo", "/", "v1.0", 0).
									WithPipeline(
										pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:latest"),
									),
							).
							WithResource(pkgbuilder.DeploymentResource),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", "v1.0", "resource-merge").
								WithUpstreamLockRef("foo", "/", "v1.0", 0).
								WithPipeline(
									pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:latest"),
									pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:latest"),
								),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
		},
		"Kptfile in the root package is merged": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithPipeline(
										pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:v1"),
									),
							).
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithPipeline(
										pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:v1"),
										pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:v1"),
									),
							).
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 0).
							WithPipeline(
								pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:v1"),
								pkgbuilder.NewFunction("gcr.io/kpt-dev/zork:v1"),
							),
					).
					WithResource(pkgbuilder.ConfigMapResource),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 1).
						WithPipeline(
							pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:v1"),
							pkgbuilder.NewFunction("gcr.io/kpt-dev/zork:v1"),
							pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:v1"),
						),
				).
				WithResource(pkgbuilder.ConfigMapResource),
		},
		"Kptfile in the nested package is merged": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 0).
											WithPipeline(
												pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:v1"),
											),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
											WithUpstreamLockRef("foo", "/", masterBranch, 1).
											WithPipeline(
												pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:latest"),
												pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:latest"),
											),
									).
									WithResource(pkgbuilder.ConfigMapResource),
							),
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithPipeline(
										pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:v1"),
										pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:latest"),
									),
							).
							WithResource(pkgbuilder.ConfigMapResource),
						Branch: masterBranch,
					},
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithPipeline(
										pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:v1"),
										pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:v1"),
									),
							).
							WithResource(pkgbuilder.ConfigMapResource),
					},
				},
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(
						pkgbuilder.NewKptfile().
							WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
							WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 0),
					).
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
									WithUpstreamLockRef("foo", "/", masterBranch, 0).
									WithPipeline(
										pkgbuilder.NewFunction("gcr.io/kpt-dev/zork:v1"),
									),
							).
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", masterBranch, "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", masterBranch, 1),
				).
				WithResource(pkgbuilder.ConfigMapResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "resource-merge").
								WithUpstreamLockRef("foo", "/", masterBranch, 1).
								WithPipeline(
									pkgbuilder.NewFunction("gcr.io/kpt-dev/zork:v1"),
									pkgbuilder.NewFunction("gcr.io/kpt-dev/foo:latest"),
									pkgbuilder.NewFunction("gcr.io/kpt-dev/bar:latest"),
								),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T:            t,
				ReposChanges: tc.reposChanges,
			}
			defer g.Clean()
			if tc.updatedLocal.Pkg != nil {
				g.LocalChanges = []testutil.Content{
					tc.updatedLocal,
				}
			}
			if !g.Init() {
				return
			}

			err := Command{
				Pkg: pkgtest.CreatePkgOrFail(t, g.LocalWorkspace.FullPackagePath()),
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
				PackagePath:       g.LocalWorkspace.FullPackagePath(),
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

			expectedPath := tc.expectedLocal.ExpandPkgWithName(t,
				g.LocalWorkspace.PackageDir, testutil.ToReposInfo(g.Repos))
			testutil.KptfileAwarePkgEqual(t, expectedPath, g.LocalWorkspace.FullPackagePath(), true)
		})
	}
}

func TestRootPackageIsUnfetched(t *testing.T) {
	testCases := map[string]struct {
		reposChanges   map[string][]testutil.Content
		directory      string
		ref            string
		expectedLocal  *pkgbuilder.RootPkg
		expectedErrMsg string
	}{
		"unfetched with no subpackages": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource),
						Branch: "main",
					},
				},
			},
			directory: "/",
			ref:       "main",
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", "main", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", "main", 0),
				).
				WithResource(pkgbuilder.DeploymentResource),
		},
		"unfetched with subpackages": {
			reposChanges: map[string][]testutil.Content{
				testutil.Upstream: {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.DeploymentResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("subpkg-1").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "fast-forward"),
									),
								pkgbuilder.NewSubPkg("subpkg-2").
									WithKptfile(
										pkgbuilder.NewKptfile().
											WithUpstreamRef("foo", "/", masterBranch, "fast-forward").
											WithUpstreamLockRef("foo", "/", masterBranch, 0),
									).
									WithResource(pkgbuilder.SecretResource),
							),
						Branch: "main",
					},
				},
				"foo": {
					{
						Pkg: pkgbuilder.NewRootPkg().
							WithResource(pkgbuilder.SecretResource),
						Branch: masterBranch,
					},
				},
			},
			directory: "/",
			ref:       "main",
			expectedLocal: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithUpstreamRef(testutil.Upstream, "/", "main", "resource-merge").
						WithUpstreamLockRef(testutil.Upstream, "/", "main", 0),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg-1").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "fast-forward").
								WithUpstreamLockRef("foo", "/", masterBranch, 0),
						).
						WithResource(pkgbuilder.SecretResource),
					pkgbuilder.NewSubPkg("subpkg-2").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstreamRef("foo", "/", masterBranch, "fast-forward").
								WithUpstreamLockRef("foo", "/", masterBranch, 0),
						).
						WithResource(pkgbuilder.SecretResource),
				),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repos, w, clean := testutil.SetupReposAndWorkspace(t, tc.reposChanges)
			defer clean()

			w.PackageDir = testPackageName
			kf := kptfileutil.DefaultKptfile(testPackageName)
			kf.Upstream = &kptfilev1.Upstream{
				Type: kptfilev1.GitOrigin,
				Git: &kptfilev1.Git{
					Repo:      repos[testutil.Upstream].RepoDirectory,
					Directory: tc.directory,
					Ref:       tc.ref,
				},
				UpdateStrategy: kptfilev1.ResourceMerge,
			}
			testutil.AddKptfileToWorkspace(t, w, kf)

			err := Command{
				Pkg: pkgtest.CreatePkgOrFail(t, w.FullPackagePath()),
			}.Run(fake.CtxWithDefaultPrinter())
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			expectedPath := tc.expectedLocal.ExpandPkgWithName(t, testPackageName, testutil.ToReposInfo(repos))
			testutil.KptfileAwarePkgEqual(t, expectedPath, w.FullPackagePath(), true)
		})
	}
}

type nonKRMTestCase struct {
	name            string
	updated         string
	original        string
	local           string
	modifyLocalFile bool
	expectedLocal   string
}

var nonKRMTests = []nonKRMTestCase{
	// Dataset5 is replica of Dataset2 with additional non KRM files
	{
		name:          "updated-filesDeleted",
		updated:       testutil.Dataset2,
		original:      testutil.Dataset5,
		local:         testutil.Dataset5,
		expectedLocal: testutil.Dataset2,
	},
	{
		name:          "updated-filesAdded",
		updated:       testutil.Dataset5,
		original:      testutil.Dataset2,
		local:         testutil.Dataset2,
		expectedLocal: testutil.Dataset5,
	},
	{
		name:          "local-filesAdded",
		updated:       testutil.Dataset2,
		original:      testutil.Dataset2,
		local:         testutil.Dataset5,
		expectedLocal: testutil.Dataset5,
	},
	{
		name:            "local-filesModified",
		updated:         testutil.Dataset5,
		original:        testutil.Dataset5,
		local:           testutil.Dataset5,
		modifyLocalFile: true,
		expectedLocal:   testutil.Dataset5,
	},
}

// TestReplaceNonKRMFiles tests if the non KRM files are updated in 3-way merge fashion
func TestReplaceNonKRMFiles(t *testing.T) {
	for i := range nonKRMTests {
		test := nonKRMTests[i]
		t.Run(test.name, func(t *testing.T) {
			ds, err := testutil.GetTestDataPath()
			assert.NoError(t, err)
			updated, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			original, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			local, err := ioutil.TempDir("", "")
			assert.NoError(t, err)
			expectedLocal, err := ioutil.TempDir("", "")
			assert.NoError(t, err)

			err = copyutil.CopyDir(filepath.Join(ds, test.updated), updated)
			assert.NoError(t, err)
			err = copyutil.CopyDir(filepath.Join(ds, test.original), original)
			assert.NoError(t, err)
			err = copyutil.CopyDir(filepath.Join(ds, test.local), local)
			assert.NoError(t, err)
			err = copyutil.CopyDir(filepath.Join(ds, test.expectedLocal), expectedLocal)
			assert.NoError(t, err)
			if test.modifyLocalFile {
				err = ioutil.WriteFile(filepath.Join(local, "somefunction.py"), []byte("Print some other thing"), 0600)
				assert.NoError(t, err)
				err = ioutil.WriteFile(filepath.Join(expectedLocal, "somefunction.py"), []byte("Print some other thing"), 0600)
				assert.NoError(t, err)
			}
			// Add a yaml file in updated that should never be moved to
			// expectedLocal.
			err = ioutil.WriteFile(filepath.Join(updated, "new.yaml"), []byte("a: b"), 0600)
			assert.NoError(t, err)
			err = ReplaceNonKRMFiles(updated, original, local)
			assert.NoError(t, err)
			tg := testutil.TestGitRepo{}
			tg.AssertEqual(t, local, filepath.Join(expectedLocal), false)
		})
	}
}
