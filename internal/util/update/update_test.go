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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	. "github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	updateStrategies = []StrategyType{
		FastForward,
		ForceDeleteReplace,
		AlphaGitPatch,
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
				// Update upstream to Dataset2
				UpstreamChanges: []Content{{Data: testutil.Dataset2}},
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
			if !g.AssertKptfile(commit, "master") {
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
				// Update upstream to Dataset2
				UpstreamChanges: []Content{{Tag: "v1.2", Data: testutil.Dataset2}},
				GetSubDirectory: "java",
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            "java",
				FullPackagePath: toAbsPath(t, "java"),
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
			if !g.AssertKptfile(commit, "v1.2") {
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
		{ForceDeleteReplace, ""},
		{AlphaGitPatch, "no updates"},
		{KResourceMerge, ""},
	}
	for i := range updates {
		u := updates[i]
		t.Run(string(u.updater), func(t *testing.T) {
			// Setup the test upstream and local packages
			g := &TestSetupManager{
				T: t, Name: string(u.updater),
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				return
			}

			// Update the local package
			err := Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
			if !g.AssertKptfile(commit, "master") {
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
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
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
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
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
		expectedCommit func(writer *TestSetupManager) string
	}{
		{FastForward,
			testutil.Dataset3,                        // expect no changes to the data
			"local package files have been modified", // expect an error
			func(writer *TestSetupManager) string { // expect Kptfile to keep the commit
				f, err := kptfileutil.ReadFile(filepath.Join(writer.LocalWorkspace.WorkspaceDirectory, writer.UpstreamRepo.RepoName))
				if !assert.NoError(writer.T, err) {
					return ""
				}
				return f.Upstream.Git.Commit
			},
		},
		// forcedeletereplace should reset hard to dataset 2 -- upstream modified copy
		{ForceDeleteReplace,
			testutil.Dataset2, // expect the upstream changes
			"",                // expect no error
			func(writer *TestSetupManager) string {
				c, _ := writer.UpstreamRepo.GetCommit() // expect the upstream commit
				return c
			},
		},
		// gitpatch should create a merge conflict between 2 and 3
		{AlphaGitPatch,
			testutil.UpdateMergeConflict,     // expect a merge conflict
			"Failed to merge in the changes", // expect an error
			func(writer *TestSetupManager) string {
				c, _ := writer.UpstreamRepo.GetCommit() // expect the upstream commit as a staged change
				return c
			},
		},
		{KResourceMerge,
			testutil.DatasetMerged, // expect a merge conflict
			"",                     // expect an error
			func(writer *TestSetupManager) string {
				c, _ := writer.UpstreamRepo.GetCommit() // expect the upstream commit as a staged change
				return c
			},
		},
	}
	for i := range updates {
		u := updates[i]
		t.Run(string(u.updater), func(t *testing.T) {
			g := &TestSetupManager{
				T: t, Name: string(u.updater),
				// Update upstream to Dataset2
				UpstreamChanges: []Content{{Data: testutil.Dataset2}},
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
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
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
			if !g.AssertKptfile(expectedCommit, "master") {
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
				// Update upstream to Dataset2
				UpstreamChanges: []Content{
					{Data: testutil.Dataset2, Branch: "exp", CreateBranch: true},
					{Data: testutil.Dataset3, Branch: "master"},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
			if !g.AssertKptfile(commit, "exp") {
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
				// Update upstream to Dataset2
				UpstreamChanges: []Content{
					{Data: testutil.Dataset2, Tag: "v1.0"},
					{Data: testutil.Dataset3},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				return
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
			if !g.AssertKptfile(commit, "v1.0") {
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
				// Update upstream to Dataset5
				UpstreamChanges: []Content{
					{Data: testutil.Dataset5, Tag: "v1.0"},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				t.FailNow()
			}

			// Update the local package
			if !assert.NoError(t, Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
			if !g.AssertKptfile(commit, "v1.0") {
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
				// Update upstream to Dataset2
				UpstreamChanges: []Content{
					{Data: testutil.Dataset4, Tag: "v1.0"},
				},
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				return
			}

			// append setters to local Kptfile with values in local package different from upstream(Dataset4)
			file, err := os.OpenFile(g.UpstreamRepo.RepoName+"/Kptfile", os.O_WRONLY|os.O_APPEND, 0644)
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
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
	g := &TestSetupManager{
		T: t, Name: string(AlphaGitPatch),
		UpstreamChanges: []Content{{Data: testutil.Dataset2}},
	}
	defer g.Clean()
	if !g.Init(testutil.Dataset1) {
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
		FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
				FullPackagePath: toAbsPath(t, path),
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
			g := &TestSetupManager{
				T: t, Name: string(strategy),
				// Update upstream to Dataset2
				UpstreamChanges: []Content{{Data: testutil.Dataset2}},
			}
			defer g.Clean()
			if !g.Init(testutil.Dataset1) {
				return
			}

			err := Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
	g := &TestSetupManager{
		T: t, Name: string(strategy),
		// Update upstream to Dataset2
		UpstreamChanges: []Content{{Data: testutil.Dataset2}},
	}
	defer g.Clean()
	if !g.Init(testutil.Dataset1) {
		return
	}

	// Update the local package
	err := Command{
		Path:            g.UpstreamRepo.RepoName,
		FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
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
		initialUpstream *pkgbuilder.Pkg
		updatedUpstream *pkgbuilder.Pkg
		updatedLocal    *pkgbuilder.Pkg
		expectedLocal   *pkgbuilder.Pkg
	}{
		{
			// TODO(mortent): This does not handle Kptfiles correctly.
			name: "update fetches any new subpackages",
			initialUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
				),
			updatedUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewPackage("nestedbar").
								WithKptfile(),
						),
					pkgbuilder.NewPackage("zork").
						WithKptfile(),
				),
			expectedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewPackage("nestedbar"),
						),
					pkgbuilder.NewPackage("zork"),
				),
		},
		{
			name: "local updates remain after noop update",
			initialUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
				),
			updatedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("zork").
						WithKptfile(),
				),
			expectedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("zork").
						WithKptfile(),
				),
		},
		{
			name: "non-overlapping additions in both upstream and local is ok",
			initialUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
				),
			updatedUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("zork").
						WithKptfile(),
				),
			updatedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("abc").
						WithKptfile(),
				),
			expectedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("zork"),
					pkgbuilder.NewPackage("abc").
						WithKptfile(),
				),
		},
		{
			// TODO(mortent): This probably shouldn't work.
			name: "overlapping additions in both upstream and local is not ok",
			initialUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
				),
			updatedUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("abc").
						WithKptfile(),
				),
			updatedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("abc").
						WithKptfile(),
				),
			expectedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
					pkgbuilder.NewPackage("abc").
						WithKptfile(),
				),
		},
		{
			// TODO(mortent): It seems like the behavior here is not correct.
			name: "subpackages deleted in upstream are deleted in fork",
			initialUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
				),
			updatedUpstream: pkgbuilder.NewPackage("foo").
				WithKptfile(),
			expectedLocal: pkgbuilder.NewPackage("foo").
				WithKptfile().
				WithSubPackages(
					pkgbuilder.NewPackage("bar").
						WithKptfile(),
				),
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			dir := expandPkg(t, test.initialUpstream)

			g := &TestSetupManager{
				T: t, Name: "UpdateTest",
			}
			defer g.Clean()
			if test.updatedUpstream != nil {
				g.UpstreamChanges = []Content{
					{
						Data: expandPkg(t, test.updatedUpstream),
					},
				}
			}
			if test.updatedLocal != nil {
				g.LocalChanges = []Content{
					{
						Data: expandPkg(t, test.updatedLocal),
					},
				}
			}
			if !g.Init(dir) {
				return
			}

			err := Command{
				Path:            g.UpstreamRepo.RepoName,
				FullPackagePath: toAbsPath(t, g.UpstreamRepo.RepoName),
				Strategy:        KResourceMerge,
			}.Run()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			if !g.AssertLocalDataEquals(expandPkg(t, test.expectedLocal)) {
				t.FailNow()
			}
		})
	}
}

func expandPkg(t *testing.T, pkg *pkgbuilder.Pkg) string {
	dir, err := ioutil.TempDir("", "test-kpt-builder-")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = pkg.Build(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return filepath.Join(dir, pkg.Name)
}

func toAbsPath(t *testing.T, path string) string {
	cwd, err := os.Getwd()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return filepath.Join(cwd, path)
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

type TestSetupManager struct {
	T *testing.T
	// Name is the name of the updater being used
	Name string

	// GetRef is the git ref to fetch
	GetRef string

	// GetSubDirectory is the repo subdirectory containing the package
	GetSubDirectory string

	// UpstreamInit are made before fetching the repo
	UpstreamInit []Content

	// UpstreamChanges are upstream content changes made after cloning the repo
	UpstreamChanges []Content

	LocalChanges []Content

	UpstreamRepo *testutil.TestGitRepo

	LocalWorkspace *testutil.TestWorkspace

	cleanTestRepo func()
	cacheDir      string
	targetDir     string
}

type Content struct {
	CreateBranch bool
	Branch       string
	Data         string
	Tag          string
	Message      string
}

// Init initializes test data
// - Setup a new upstream repo in a tmp directory
// - Set the initial upstream content to Dataset1
// - Setup a new cache location for git repos and update the environment variable
// - Setup fetch the upstream package to a local package
// - Verify the local package contains the upstream content
func (g *TestSetupManager) Init(dataset string) bool {
	// Default optional values
	if g.GetRef == "" {
		g.GetRef = "master"
	}
	if g.GetSubDirectory == "" {
		g.GetSubDirectory = "/"
	}

	// Configure the cache location for cloning repos
	cacheDir, err := ioutil.TempDir("", "kpt-test-cache-repos-")
	if !assert.NoError(g.T, err) {
		return false
	}
	g.cacheDir = cacheDir
	os.Setenv(gitutil.RepoCacheDirEnv, g.cacheDir)

	// Setup a "remote" source repo, and a "local" destination repo
	g.UpstreamRepo, g.LocalWorkspace, g.cleanTestRepo = testutil.SetupDefaultRepoAndWorkspace(g.T, dataset)
	g.UpstreamRepo.Updater = g.Name
	if g.GetSubDirectory == "/" {
		g.targetDir = filepath.Base(g.UpstreamRepo.RepoName)
	} else {
		g.targetDir = filepath.Base(g.GetSubDirectory)
	}
	if !assert.NoError(g.T, os.Chdir(g.UpstreamRepo.RepoDirectory)) {
		return false
	}

	if err := updateGitDir(g.T, g.UpstreamRepo, g.UpstreamInit); err != nil {
		return false
	}

	// Fetch the source repo
	if !assert.NoError(g.T, os.Chdir(g.LocalWorkspace.WorkspaceDirectory)) {
		return false
	}

	if !assert.NoError(g.T, get.Command{
		Destination: g.targetDir,
		Git: kptfile.Git{
			Repo:      g.UpstreamRepo.RepoDirectory,
			Ref:       g.GetRef,
			Directory: g.GetSubDirectory,
		}}.Run(), g.Name) {
		return false
	}
	localGit := gitutil.NewLocalGitRunner(g.LocalWorkspace.WorkspaceDirectory)
	if !assert.NoError(g.T, localGit.Run("add", ".")) {
		return false
	}
	if !assert.NoError(g.T, localGit.Run("commit", "-m", "add files")) {
		return false
	}

	// Modify source repository state after fetching it
	if err := updateGitDir(g.T, g.UpstreamRepo, g.UpstreamChanges); err != nil {
		return false
	}

	// Verify the local package has the correct dataset
	if same := g.AssertLocalDataEquals(filepath.Join(dataset, g.GetSubDirectory)); !same {
		return same
	}

	if err := updateGitDir(g.T, g.LocalWorkspace, g.LocalChanges); err != nil {
		return false
	}

	return true
}

type GitDirectory interface {
	CheckoutBranch(branch string, create bool) error
	ReplaceData(data string) error
	Commit(message string) error
	Tag(tagName string) error
}

func updateGitDir(t *testing.T, gitDir GitDirectory, changes []Content) error {
	for _, content := range changes {
		if content.Message == "" {
			content.Message = "initializing data"
		}
		if len(content.Branch) > 0 {
			err := gitDir.CheckoutBranch(content.Branch, content.CreateBranch)
			if !assert.NoError(t, err) {
				return err
			}
		}

		err := gitDir.ReplaceData(content.Data)
		if !assert.NoError(t, err) {
			return err
		}

		err = gitDir.Commit(content.Message)
		if !assert.NoError(t, err) {
			return err
		}
		if len(content.Tag) > 0 {
			err = gitDir.Tag(content.Tag)
			if !assert.NoError(t, err) {
				return err
			}
		}
	}
	return nil
}

func (g *TestSetupManager) AssertKptfile(commit, ref string) bool {
	name := g.UpstreamRepo.RepoName
	if g.targetDir != "" {
		name = g.targetDir
	}
	expectedKptfile := kptfile.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: name,
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
				Directory: g.GetSubDirectory,
				Repo:      g.UpstreamRepo.RepoDirectory,
				Ref:       ref,
				Commit:    commit,
			},
		},
	}

	return g.UpstreamRepo.AssertKptfile(
		g.T, filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir), expectedKptfile)
}

func (g *TestSetupManager) AssertLocalDataEquals(path string) bool {
	var sourceDir string
	if filepath.IsAbs(path) {
		sourceDir = path
	} else {
		sourceDir = filepath.Join(g.UpstreamRepo.DatasetDirectory, path)
	}
	destDir := filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.targetDir)
	return g.UpstreamRepo.AssertEqual(g.T, sourceDir, destDir)
}

func (g *TestSetupManager) SetLocalData(path string) bool {
	if !assert.NoError(g.T, copyutil.CopyDir(
		filepath.Join(g.UpstreamRepo.DatasetDirectory, path),
		filepath.Join(g.LocalWorkspace.WorkspaceDirectory, g.UpstreamRepo.RepoName))) {
		return false
	}
	localGit := gitutil.NewLocalGitRunner(g.LocalWorkspace.WorkspaceDirectory)
	if !assert.NoError(g.T, localGit.Run("add", ".")) {
		return false
	}
	if !assert.NoError(g.T, localGit.Run("commit", "-m", "add files")) {
		return false
	}
	return true
}

func (g *TestSetupManager) Clean() {
	g.cleanTestRepo()
	os.RemoveAll(g.cacheDir)
}
