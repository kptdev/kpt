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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	. "github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

var (
	updateStrategies = []StrategyType{
		FastForward,
		KResourceMerge,
	}
)

// TestCommand_Run_noRefChanges updates a package without specifying a new ref.
// - Get a package using  a branch ref
// - Modify upstream with new content
// - Update the local package to fetch the upstream content
func TestCommand_Run_noRefChanges(t *testing.T) {
	for i := range updateStrategies {
		strategy := updateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset2
				UpstreamChanges: []testutil.Content{
					{
						Data: testutil.Dataset2,
					},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        strategy,
			}.Run()) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(testutil.Dataset2) {
				return
			}
			commit, err := g.UpstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(g.UpstreamRepo.RepoName, commit, "master") {
				return
			}
		})
	}
}

func TestCommand_Run_subDir(t *testing.T) {
	for i := range updateStrategies {
		strategy := updateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset2
				UpstreamChanges: []testutil.Content{{Tag: "v1.2", Data: testutil.Dataset2}},
				GetSubDirectory: "java",
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            "java",
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Ref:             "v1.2",
				Strategy:        strategy,
			}.Run()) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(filepath.Join(testutil.Dataset2, "java")) {
				return
			}
			commit, err := g.UpstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(g.GetSubDirectory, commit, "v1.2") {
				return
			}
		})
	}
}

func TestCommand_Run_noChanges(t *testing.T) {
	updates := []struct {
		updater StrategyType
		err     string
	}{
		{FastForward, ""},
		// {ForceDeleteReplace, ""},
		// {AlphaGitPatch, "no updates"},
		{KResourceMerge, ""},
	}
	for i := range updates {
		u := updates[i]
		t.Run(string(u.updater), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// Update the local package
			err := Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        u.updater,
			}.Run()
			if u.err == "" {
				if !assert.NoError(t, err, u.updater) {
					return
				}
			} else {
				if assert.Error(t, err, u.updater) {
					assert.Contains(t, err.Error(), "no updates", u.updater)
				}
			}

			if !g.AssertLocalDataEquals(testutil.Dataset1) {
				return
			}
			commit, err := g.UpstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(g.UpstreamRepo.RepoName, commit, "master") {
				return
			}
		})
	}
}

func TestCommand_Run_noCommit(t *testing.T) {
	strategies := append([]StrategyType{Default}, updateStrategies...)
	for i := range strategies {
		strategy := strategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// don't commit the data
			err := copyutil.CopyDir(
				filepath.Join(g.UpstreamRepo.DatasetDirectory, testutil.Dataset3),
				filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.UpstreamRepo.RepoName))
			if !assert.NoError(t, err) {
				return
			}

			// Update the local package
			err = Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        strategy,
			}.Run()
			if !assert.Error(t, err) {
				return
			}
			assert.Contains(t, err.Error(), "must commit package")

			if !g.AssertLocalDataEquals(testutil.Dataset3) {
				return
			}
		})
	}
}

func TestCommand_Run_noAdd(t *testing.T) {
	strategies := append([]StrategyType{Default}, updateStrategies...)
	for i := range strategies {
		strategy := strategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// don't add the data
			err := ioutil.WriteFile(
				filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.UpstreamRepo.RepoName, "java", "added-file"), []byte(`hello`),
				0600)
			if !assert.NoError(t, err) {
				return
			}

			// Update the local package
			err = Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        strategy,
			}.Run()
			if !assert.Error(t, err) {
				return
			}
			assert.Contains(t, err.Error(), "must commit package")
		})
	}
}

// TestCommand_Run_localPackageChanges updates a package that has been locally modified
// - Get a package using  a branch ref
// - Modify upstream with new content
// - Modify local package with new content
// - Update the local package to fetch the upstream content
func TestCommand_Run_localPackageChanges(t *testing.T) {
	updates := []struct {
		updater        StrategyType // update strategy type
		expectedData   string       // expect
		expectedErr    string
		expectedCommit func(writer *testutil.TestSetupManager) string
	}{
		{FastForward,
			testutil.Dataset3,                        // expect no changes to the data
			"local package files have been modified", // expect an error
			func(writer *testutil.TestSetupManager) string { // expect Kptfile to keep the commit
				f, err := kptfileutil.ReadFile(filepath.Join(writer.LocalWorkspace.WorkspaceDirectory, writer.UpstreamRepo.RepoName))
				if !assert.NoError(writer.T, err) {
					return ""
				}
				return f.Upstream.Git.Commit
			},
		},
		// // forcedeletereplace should reset hard to dataset 2 -- upstream modified copy
		// {ForceDeleteReplace,
		//	 testutil.Dataset2, // expect the upstream changes
		//	 "",                // expect no error
		//	 func(writer *testutil.TestSetupManager) string {
		// 		c, _ := writer.UpstreamRepo.GetCommit() // expect the upstream commit
		// 		return c
		//	 },
		// },
		// // gitpatch should create a merge conflict between 2 and 3
		// {AlphaGitPatch,
		// 	 testutil.UpdateMergeConflict,     // expect a merge conflict
		//	 "Failed to merge in the changes", // expect an error
		//	 func(writer *testutil.TestSetupManager) string {
		// 		c, _ := writer.UpstreamRepo.GetCommit() // expect the upstream commit as a staged change
		// 		return c
		// 	 },
		// },
		{KResourceMerge,
			testutil.DatasetMerged, // expect a merge conflict
			"",                     // expect an error
			func(writer *testutil.TestSetupManager) string {
				c, _ := writer.UpstreamRepo.GetCommit() // expect the upstream commit as a staged change
				return c
			},
		},
	}
	for i := range updates {
		u := updates[i]
		t.Run(string(u.updater), func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset2
				UpstreamChanges: []testutil.Content{{Data: testutil.Dataset2}},
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				t.FailNow()
			}

			// Modify local data to Dataset3
			if !g.SetLocalData(testutil.Dataset3) {
				t.FailNow()
			}

			// record the expected commit after update
			expectedCommit := u.expectedCommit(g)

			// run the command
			err := Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Ref:             "master",
				Strategy:        u.updater,
				SimpleMessage:   true, // so merge conflict marks are predictable
			}.Run()

			// check the error response
			if u.expectedErr == "" {
				if !assert.NoError(t, err, u.updater) {
					t.FailNow()
				}
			} else {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), u.expectedErr) {
					t.FailNow()
				}
			}

			if !g.AssertLocalDataEquals(u.expectedData) {
				t.FailNow()
			}
			if !g.AssertKptfile(g.UpstreamRepo.RepoName, expectedCommit, "master") {
				t.FailNow()
			}
		})
	}
}

// TestCommand_Run_toBranchRef verifies the package contents are set to the contents of the branch
// it was updated to.
func TestCommand_Run_toBranchRef(t *testing.T) {
	for i := range updateStrategies {
		strategy := updateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset2
				UpstreamChanges: []testutil.Content{
					{Data: testutil.Dataset2, Branch: "exp", CreateBranch: true},
					{Data: testutil.Dataset3, Branch: "master"},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        strategy,
				Ref:             "exp",
			}.Run()) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(testutil.Dataset2) {
				return
			}

			if !assert.NoError(t, g.UpstreamRepo.CheckoutBranch("exp", false)) {
				return
			}
			commit, err := g.UpstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(g.UpstreamRepo.RepoName, commit, "exp") {
				return
			}
		})
	}
}

// TestCommand_Run_toTagRef verifies the package contents are set to the contents of the tag
// it was updated to.
func TestCommand_Run_toTagRef(t *testing.T) {
	for i := range updateStrategies {
		strategy := updateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset2
				UpstreamChanges: []testutil.Content{
					{Data: testutil.Dataset2, Tag: "v1.0"},
					{Data: testutil.Dataset3},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        strategy,
				Ref:             "v1.0",
			}.Run()) {
				return
			}

			// Expect the local package to have Dataset2
			if !g.AssertLocalDataEquals(testutil.Dataset2) {
				return
			}

			if !assert.NoError(t, g.UpstreamRepo.CheckoutBranch("v1.0", false)) {
				return
			}
			commit, err := g.UpstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				return
			}
			if !g.AssertKptfile(g.UpstreamRepo.RepoName, commit, "v1.0") {
				return
			}
		})
	}
}

// TestCommand_ResourceMerge_NonKRMUpdates tests if the local non KRM files are updated
func TestCommand_ResourceMerge_NonKRMUpdates(t *testing.T) {
	strategies := []StrategyType{KResourceMerge}
	for i := range strategies {
		strategy := strategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset5
				UpstreamChanges: []testutil.Content{
					{Data: testutil.Dataset5, Tag: "v1.0"},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				t.FailNow()
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        strategy,
				Ref:             "v1.0",
			}.Run()) {
				t.FailNow()
			}

			// Expect the local package to have Dataset5
			if !g.AssertLocalDataEquals(testutil.Dataset5) {
				t.FailNow()
			}

			if !assert.NoError(t, g.UpstreamRepo.CheckoutBranch("v1.0", false)) {
				t.FailNow()
			}
			commit, err := g.UpstreamRepo.GetCommit()
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !g.AssertKptfile(g.UpstreamRepo.RepoName, commit, "v1.0") {
				t.FailNow()
			}
		})
	}
}

// TestCommand_Run_toTagRef verifies the package contents are set to the contents of the tag
// it was updated to with local values set to different values in upstream.
func TestCommand_ResourceMerge_WithSetters_TagRef(t *testing.T) {
	strategies := []StrategyType{KResourceMerge}
	for i := range strategies {
		strategy := strategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset2
				UpstreamChanges: []testutil.Content{
					{Data: testutil.Dataset4, Tag: "v1.0"},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			// append setters to local Kptfile with values in local package different from upstream(Dataset4)
			file, err := os.OpenFile(filepath.Join(g.LocalWorkspace.FullPackagePath(), "/Kptfile"),
				os.O_WRONLY|os.O_APPEND, 0644)
			if !assert.NoError(t, err) {
				return
			}
			defer file.Close()

			_, err = file.WriteString(`openAPI:
  definitions:
    io.k8s.cli.setters.name:
      x-k8s-cli:
        setter:
          name: name
          value: "app-config"
    io.k8s.cli.setters.replicas:
      x-k8s-cli:
        setter:
          name: replicas
          value: "3"`)

			if !assert.NoError(t, err) {
				return
			}

			localGit := gitutil.NewLocalGitRunner(g.LocalWorkspace.WorkspaceDirectory)
			if !assert.NoError(g.T, localGit.Run("add", ".")) {
				t.FailNow()
			}
			if !assert.NoError(g.T, localGit.Run("commit", "-m", "add files")) {
				t.FailNow()
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Strategy:        strategy,
				Ref:             "v1.0",
			}.Run()) {
				return
			}

			// Expect the local package to have Dataset2
			// Dataset2 is replica of Dataset4 but with setter values same as local package
			// This tests the feature https://github.com/GoogleContainerTools/kpt/issues/284
			if !g.AssertLocalDataEquals(testutil.Dataset2) {
				return
			}

			if !assert.NoError(t, g.UpstreamRepo.CheckoutBranch("v1.0", false)) {
				return
			}
		})
	}
}

func TestCommand_Run_emitPatch(t *testing.T) {
	// Setup the test upstream and local packages
	g := &testutil.TestSetupManager{
		T:               t,
		UpstreamChanges: []testutil.Content{{Data: testutil.Dataset2}},
	}
	defer g.Clean()
	if !g.Init(testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}) {
		return
	}

	f, err := ioutil.TempFile("", "*.patch")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(f.Name())

	// Update the local package
	b := &bytes.Buffer{}
	err = Command{
		Path:            g.UpstreamRepo.RepoName,
		FullPackagePath: g.LocalWorkspace.FullPackagePath(),
		Strategy:        AlphaGitPatch,
		DryRun:          true,
		Output:          b,
	}.Run()
	if !assert.NoError(t, err) {
		return
	}

	assert.Contains(t, b.String(), `-          initialDelaySeconds: 30
-          periodSeconds: 10
+          initialDelaySeconds: 45
+          periodSeconds: 15
           timeoutSeconds: 5
`)
}

// TestCommand_Run_failInvalidPath verifies Run fails if the path is invalid
func TestCommand_Run_failInvalidPath(t *testing.T) {
	for i := range updateStrategies {
		strategy := updateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			path := filepath.Join("fake", "path")
			err := Command{
				Path:            path,
				FullPackagePath: path,
				Strategy:        strategy,
			}.Run()
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "no such file or directory")
			}
		})
	}
}

// TestCommand_Run_failInvalidRef verifies Run fails if the ref is invalid
func TestCommand_Run_failInvalidRef(t *testing.T) {
	for i := range updateStrategies {
		strategy := updateStrategies[i]
		t.Run(string(strategy), func(t *testing.T) {
			g := &testutil.TestSetupManager{
				T: t,
				// Update upstream to Dataset2
				UpstreamChanges: []testutil.Content{{Data: testutil.Dataset2}},
			}
			defer g.Clean()
			if !g.Init(testutil.Content{
				Data:   testutil.Dataset1,
				Branch: "master",
			}) {
				return
			}

			err := Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: g.LocalWorkspace.FullPackagePath(),
				Ref:             "exp",
				Strategy:        strategy,
			}.Run()
			if !assert.Error(t, err) {
				return
			}
			assert.Contains(t, err.Error(), "failed to clone git repo")

			if !g.AssertLocalDataEquals(testutil.Dataset1) {
				return
			}
		})
	}
}

func TestCommand_Run_badStrategy(t *testing.T) {
	strategy := StrategyType("foo")

	// Setup the test upstream and local packages
	g := &testutil.TestSetupManager{
		T: t,
		// Update upstream to Dataset2
		UpstreamChanges: []testutil.Content{{Data: testutil.Dataset2}},
	}
	defer g.Clean()
	if !g.Init(testutil.Content{
		Data:   testutil.Dataset1,
		Branch: "master",
	}) {
		return
	}

	// Update the local package
	err := Command{
		Path:            g.UpstreamRepo.RepoName,
		FullPackagePath: g.LocalWorkspace.FullPackagePath(),
		Strategy:        strategy,
	}.Run()
	if !assert.Error(t, err, strategy) {
		return
	}
	assert.Contains(t, err.Error(), "unrecognized update strategy")
}

func TestCommand_Run_subpackages(t *testing.T) {
	testCases := []struct {
		name            string
		initialUpstream *pkgbuilder.RootPkg
		updatedUpstream testutil.Content
		updatedLocal    testutil.Content
		expectedResults []resultForStrategy
	}{
		{
			name: "update fetches any new subpackages",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(),
				),
			updatedUpstream: testutil.Content{
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
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge, FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
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
		{
			name: "local changes and a noop update",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(),
				),
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
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(),
							pkgbuilder.NewSubPkg("zork").
								WithKptfile(),
						),
				},
				{
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
				},
			},
		},
		{
			name: "non-overlapping additions in both upstream and local",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(),
						pkgbuilder.NewSubPkg("zork").
							WithKptfile(),
					),
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
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile().
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
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
				},
			},
		},
		{
			name: "overlapping additions in both upstream and local",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(),
						pkgbuilder.NewSubPkg("abc").
							WithKptfile(),
					),
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
					strategies:     []StrategyType{KResourceMerge},
					expectedErrMsg: "subpackage \"abc\" added in both upstream and local",
				},
				{
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
				},
			},
		},
		{
			name: "subpackages deleted in upstream",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge, FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(),
				},
			},
		},
		{
			name: "multiple layers of subpackages added in upstream",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile(),
			updatedUpstream: testutil.Content{
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
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge, FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
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
		{
			name: "new setters are added in upstream",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithSetters(
						pkgbuilder.NewSetter("gcloud.core.project", "PROJECT_ID"),
						pkgbuilder.NewSetter("gcloud.project.projectNumber", "PROJECT_NUMBER"),
					),
				).
				WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
					pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
					pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
				}).
				WithSubPackages(
					pkgbuilder.NewSubPkg("nosetters").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
					pkgbuilder.NewSubPkg("storage").
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetter("gcloud.core.project", "PROJECT_ID"),
								pkgbuilder.NewSetter("gcloud.project.projectNumber", "PROJECT_NUMBER"),
							),
						).
						WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
							pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
							pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
						}),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile().
						WithSetters(
							pkgbuilder.NewSetter("gcloud.core.project", "PROJECT_ID"),
							pkgbuilder.NewSetter("gcloud.project.projectNumber", "PROJECT_NUMBER"),
						),
					).
					WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
						pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
						pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
					}).
					WithSubPackages(
						pkgbuilder.NewSubPkg("nosetters").
							WithKptfile(pkgbuilder.NewKptfile().
								WithSetters(
									pkgbuilder.NewSetter("new-setter", "some-value"),
								),
							).
							WithResource(pkgbuilder.DeploymentResource),
						pkgbuilder.NewSubPkg("storage").
							WithKptfile(pkgbuilder.NewKptfile().
								WithSetters(
									pkgbuilder.NewSetter("gcloud.core.project", "PROJECT_ID"),
									pkgbuilder.NewSetter("gcloud.project.projectNumber", "PROJECT_NUMBER"),
								),
							).
							WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
								pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
								pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
							}),
					),
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile().
						WithSetters(
							pkgbuilder.NewSetSetter("gcloud.core.project", "my-project"),
							pkgbuilder.NewSetSetter("gcloud.project.projectNumber", "a1234"),
						),
					).
					WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
						pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
						pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
					},
						pkgbuilder.SetFieldPath("my-project", "metadata", "namespace"),
						pkgbuilder.SetFieldPath("a1234", "spec", "foo"),
					).
					WithSubPackages(
						pkgbuilder.NewSubPkg("nosetters").
							WithKptfile().
							WithResource(pkgbuilder.DeploymentResource),
						pkgbuilder.NewSubPkg("storage").
							WithKptfile(pkgbuilder.NewKptfile().
								WithSetters(
									pkgbuilder.NewSetSetter("gcloud.core.project", "my-project"),
									pkgbuilder.NewSetSetter("gcloud.project.projectNumber", "a1234"),
								),
							).
							WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
								pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
								pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
							},
								pkgbuilder.SetFieldPath("my-project", "metadata", "namespace"),
								pkgbuilder.SetFieldPath("a1234", "spec", "foo"),
							),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetSetter("gcloud.core.project", "my-project"),
								pkgbuilder.NewSetSetter("gcloud.project.projectNumber", "a1234"),
							),
						).
						WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
							pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
							pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
						},
							pkgbuilder.SetFieldPath("my-project", "metadata", "namespace"),
							pkgbuilder.SetFieldPath("a1234", "spec", "foo"),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("nosetters").
								WithKptfile(pkgbuilder.NewKptfile().
									WithSetters(
										pkgbuilder.NewSetter("new-setter", "some-value"),
									),
								).
								WithResource(pkgbuilder.DeploymentResource),
							pkgbuilder.NewSubPkg("storage").
								WithKptfile(pkgbuilder.NewKptfile().
									WithSetters(
										pkgbuilder.NewSetSetter("gcloud.core.project", "my-project"),
										pkgbuilder.NewSetSetter("gcloud.project.projectNumber", "a1234"),
									),
								).
								WithResourceAndSetters(pkgbuilder.DeploymentResource, []pkgbuilder.SetterRef{
									pkgbuilder.NewSetterRef("gcloud.core.project", "metadata", "namespace"),
									pkgbuilder.NewSetterRef("gcloud.project.projectNumber", "spec", "foo"),
								},
									pkgbuilder.SetFieldPath("my-project", "metadata", "namespace"),
									pkgbuilder.SetFieldPath("a1234", "spec", "foo"),
								),
						),
				},
				{
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
				},
			},
		},
		{
			name: "removed Kptfile from upstream",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetter("name", "my-name"),
							),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithResource(pkgbuilder.DeploymentResource),
					),
			},
			expectedResults: []resultForStrategy{
				// TODO(mortent): Revisit this. Not clear that the Kptfile
				// shouldn't be deleted here since it doesn't really have any
				// local changes.
				{
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(pkgbuilder.NewKptfile().
									WithSetters(
										pkgbuilder.NewSetter("name", "my-name"),
									),
								).
								WithResource(pkgbuilder.DeploymentResource),
						),
				},
				{
					strategies: []StrategyType{FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithResource(pkgbuilder.DeploymentResource),
						),
				},
			},
		},
		{
			name: "Kptfile added only on local",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithResource(pkgbuilder.DeploymentResource),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithResource(pkgbuilder.DeploymentResource).
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(pkgbuilder.NewKptfile().
								WithSetters(
									pkgbuilder.NewSetter("name", "my-name"),
								),
							).
							WithResource(pkgbuilder.DeploymentResource),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(pkgbuilder.NewKptfile().
									WithSetters(
										pkgbuilder.NewSetter("name", "my-name"),
									),
								).
								WithResource(pkgbuilder.DeploymentResource).
								WithResource(pkgbuilder.ConfigMapResource),
						),
				},
				{
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
				},
			},
		},
		{
			name: "subpackage deleted from upstream but is unchanged in local",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetter("foo", "val"),
							),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge, FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(),
				},
			},
		},
		{
			name: "subpackage deleted from upstream but has local changes",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetter("foo", "val"),
							),
						).
						WithResource(pkgbuilder.DeploymentResource),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(),
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(pkgbuilder.NewKptfile().
								WithSetters(
									pkgbuilder.NewSetter("foo", "val"),
								),
							).
							WithResource(pkgbuilder.DeploymentResource,
								pkgbuilder.SetFieldPath("34", "spec", "replicas")),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithKptfile(pkgbuilder.NewKptfile().
									WithSetters(
										pkgbuilder.NewSetter("foo", "val"),
									),
								).
								WithResource(pkgbuilder.DeploymentResource,
									pkgbuilder.SetFieldPath("34", "spec", "replicas")),
						),
				},
				{
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
				},
			},
		},
		{
			name: "autosetters with deeply nested packages",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithSetters(
						pkgbuilder.NewSetter("band", "placeholder"),
					),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile().
						WithSetters(
							pkgbuilder.NewSetter("band", "placeholder"),
						),
					).WithSubPackages(
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetter("band", "placeholder"),
							),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("nestedbar").
								WithKptfile(pkgbuilder.NewKptfile().
									WithSetters(
										pkgbuilder.NewSetter("band", "placeholder"),
									),
								),
						),
				),
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile().
						WithSetters(
							pkgbuilder.NewSetSetter("band", "Hüsker Dü"),
						),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetSetter("band", "Hüsker Dü"),
							),
						).WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(pkgbuilder.NewKptfile().
								WithSetters(
									pkgbuilder.NewSetSetter("band", "Hüsker Dü"),
								),
							).
							WithSubPackages(
								pkgbuilder.NewSubPkg("nestedbar").
									WithKptfile(pkgbuilder.NewKptfile().
										WithSetters(
											pkgbuilder.NewSetSetter("band", "Hüsker Dü"),
										),
									),
							),
					),
				},
				{
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
				},
			},
		},
		{
			name: "Merging of Kptfile in nested package",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithUpstream("github.com/foo/bar", "somebranch"),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(pkgbuilder.NewKptfile().
							WithUpstream("gitlab.com/comp/proj", "1234abcd").
							WithSetters(
								pkgbuilder.NewSetter("setter1", "value1"),
							),
						),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile().
						WithUpstream("github.com/foo/bar", "somebranch"),
					).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(pkgbuilder.NewKptfile().
								WithUpstream("gitlab.com/comp/proj", "abcd1234").
								WithSetters(
									pkgbuilder.NewSetter("setter1", "value1"),
									pkgbuilder.NewSetter("setter2", "value2"),
								),
							),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge, FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(pkgbuilder.NewKptfile().
							WithUpstream("github.com/foo/bar", "somebranch"),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("subpkg").
								WithKptfile(pkgbuilder.NewKptfile().
									WithUpstream("gitlab.com/comp/proj", "abcd1234").
									WithSetters(
										pkgbuilder.NewSetter("setter1", "value1"),
										pkgbuilder.NewSetter("setter2", "value2"),
									),
								),
						),
				},
			},
		},
		{
			name: "Upstream package doesn't need to have a Kptfile in the root",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(pkgbuilder.NewKptfile()),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource).
					WithResource(pkgbuilder.ConfigMapResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(pkgbuilder.NewKptfile()),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge, FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile().
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
			name: "Non-krm files updated in upstream",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile()).
				WithFile("data.txt", "initial content").
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile(pkgbuilder.NewKptfile()).
						WithFile("information", "first version"),
				),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile()).
					WithFile("data.txt", "updated content").
					WithSubPackages(
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(pkgbuilder.NewKptfile()).
							WithFile("information", "second version"),
					),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge, FastForward},
					expectedLocal: pkgbuilder.NewRootPkg().
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
		{
			name: "Non-krm files updated in both upstream and local",
			initialUpstream: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile()).
				WithFile("data.txt", "initial content"),
			updatedUpstream: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile()).
					WithFile("data.txt", "updated content"),
			},
			updatedLocal: testutil.Content{
				Pkg: pkgbuilder.NewRootPkg().
					WithKptfile(pkgbuilder.NewKptfile()).
					WithFile("data.txt", "local content"),
			},
			expectedResults: []resultForStrategy{
				{
					strategies: []StrategyType{KResourceMerge},
					expectedLocal: pkgbuilder.NewRootPkg().
						WithKptfile(pkgbuilder.NewKptfile()).
						WithFile("data.txt", "local content"),
				},
				{
					strategies:     []StrategyType{FastForward},
					expectedErrMsg: "use a different update --strategy",
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
				var emptyMap map[string]string
				dir := pkgbuilder.ExpandPkg(t, test.initialUpstream, emptyMap)

				g := &testutil.TestSetupManager{
					T: t,
				}
				defer g.Clean()
				if test.updatedUpstream.Pkg != nil {
					g.UpstreamChanges = []testutil.Content{
						test.updatedUpstream,
					}
				}
				if test.updatedLocal.Pkg != nil {
					g.LocalChanges = []testutil.Content{
						test.updatedLocal,
					}
				}
				if !g.Init(testutil.Content{
					Data:   dir,
					Branch: "master",
				}) {
					return
				}

				err := Command{
					Path:            g.UpstreamRepo.RepoName,
					FullPackagePath: g.LocalWorkspace.FullPackagePath(),
					Strategy:        strategy,
				}.Run()

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

				expectedPath := pkgbuilder.ExpandPkgWithName(t, result.expectedLocal, g.LocalWorkspace.PackageDir, emptyMap)

				if !g.AssertLocalDataEquals(expectedPath) {
					t.FailNow()
				}
				commit, err := g.UpstreamRepo.GetCommit()
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				if !g.AssertKptfileEquals(expectedPath, commit, "master") {
					t.FailNow()
				}
			})
		}
	}
}

type resultForStrategy struct {
	strategies     []StrategyType
	expectedLocal  *pkgbuilder.RootPkg
	expectedErrMsg string
}

func findStrategiesForTestCase(expectedResults []resultForStrategy) []StrategyType {
	var strategies []StrategyType
	for _, er := range expectedResults {
		strategies = append(strategies, er.strategies...)
	}
	return strategies
}

func findExpectedResultForStrategy(strategyResults []resultForStrategy,
	strategy StrategyType) resultForStrategy {
	for _, sr := range strategyResults {
		for _, s := range sr.strategies {
			if s == strategy {
				return sr
			}
		}
	}
	panic(fmt.Errorf("unknown strategy %s", string(strategy)))
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
			tg.AssertEqual(t, local, filepath.Join(expectedLocal))
		})
	}
}
