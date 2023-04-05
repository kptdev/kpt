// Copyright 2021 The kpt Authors
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

package gitutil_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	. "github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.ConfigureTestKptCache(m))
}

func TestLocalGitRunner(t *testing.T) {
	testCases := map[string]struct {
		command        string
		args           []string
		expectedStdout string
		expectedErr    *GitExecError
	}{
		"successful command with output to stdout": {
			command:        "branch",
			args:           []string{"--show-current"},
			expectedStdout: "main",
		},
		"failed command with output to stderr": {
			command: "checkout",
			args:    []string{"does-not-exist"},
			expectedErr: &GitExecError{
				StdOut: "",
				StdErr: "error: pathspec 'does-not-exist' did not match any file(s) known to git",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			dir := t.TempDir()

			runner, err := NewLocalGitRunner(dir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			_, err = runner.Run(fake.CtxWithDefaultPrinter(), "init", "--initial-branch=main")
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			rr, err := runner.Run(fake.CtxWithDefaultPrinter(), tc.command, tc.args...)
			if tc.expectedErr != nil {
				var gitExecError *GitExecError
				if !errors.As(err, &gitExecError) {
					t.Error("expected error of type *GitExecError")
					t.FailNow()
				}
				assert.Equal(t, tc.expectedErr.StdOut, strings.TrimSpace(gitExecError.StdOut))
				assert.Equal(t, tc.expectedErr.StdErr, strings.TrimSpace(gitExecError.StdErr))
				return
			}

			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.Equal(t, tc.expectedStdout, strings.TrimSpace(rr.Stdout))
		})
	}
}

func TestNewGitUpstreamRepo_noRepo(t *testing.T) {
	dir := t.TempDir()

	_, err := NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), dir)
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "does not appear to be a git repository")
}

func TestNewGitUpstreamRepo_noRefs(t *testing.T) {
	dir := t.TempDir()

	runner, err := NewLocalGitRunner(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = runner.Run(fake.CtxWithDefaultPrinter(), "init", "--bare")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	gur, err := NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 0, len(gur.Heads))
	assert.Equal(t, 0, len(gur.Tags))
}

func TestNewGitUpstreamRepo(t *testing.T) {
	testCases := map[string]struct {
		repoContent   []testutil.Content
		expectedHeads []string
		expectedTags  []string
	}{
		"single branch, no tags": {
			repoContent: []testutil.Content{
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch: "master",
				},
			},
			expectedHeads: []string{"master"},
			expectedTags:  []string{},
		},
		"multiple tags and branches": {
			repoContent: []testutil.Content{
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch: "master",
					Tag:    "v1",
				},
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch:       "main",
					CreateBranch: true,
					Tag:          "v2",
				},
			},
			expectedHeads: []string{"main", "master"},
			expectedTags:  []string{"v1", "v2"},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			repoContent := map[string][]testutil.Content{
				testutil.Upstream: tc.repoContent,
			}
			g, _, clean := testutil.SetupReposAndWorkspace(t, repoContent)
			defer clean()
			if !assert.NoError(t, testutil.UpdateRepos(t, g, repoContent)) {
				t.FailNow()
			}

			gur, err := NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.EqualValues(t, tc.expectedHeads, toKeys(gur.Heads))
			assert.EqualValues(t, tc.expectedTags, toKeys(gur.Tags))
		})
	}
}

func TestGitUpstreamRepo_GetDefaultBranch_noRefs(t *testing.T) {
	dir := t.TempDir()

	runner, err := NewLocalGitRunner(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = runner.Run(fake.CtxWithDefaultPrinter(), "init", "--bare")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	gur, err := NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = gur.GetDefaultBranch(fake.CtxWithDefaultPrinter())
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "unable to detect default branch in repo")
}

func TestGitUpstreamRepo_GetDefaultBranch(t *testing.T) {
	testCases := map[string]struct {
		repoContent []testutil.Content
		expectedRef string
	}{
		"selects the default branch if it is the only one available": {
			repoContent: []testutil.Content{
				{
					Data:   testutil.Dataset1,
					Branch: "main",
				},
			},
			expectedRef: "main",
		},
		"selects the default branch if there are multiple branches": {
			repoContent: []testutil.Content{
				{
					Data:   testutil.Dataset1,
					Branch: "foo",
				},
				{
					Data:   testutil.Dataset2,
					Branch: "main",
				},
				{
					Data:   testutil.Dataset3,
					Branch: "master",
				},
			},
			expectedRef: "foo",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g, _, clean := testutil.SetupReposAndWorkspace(t, map[string][]testutil.Content{
				testutil.Upstream: tc.repoContent,
			})
			defer clean()

			gur, err := NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			defaultRef, err := gur.GetDefaultBranch(fake.CtxWithDefaultPrinter())
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Equal(t, tc.expectedRef, defaultRef) {
				t.FailNow()
			}
		})
	}
}

func TestGitUpstreamRepo_GetRepo(t *testing.T) {
	testCases := map[string]struct {
		repoContent []testutil.Content
		refsFunc    func(*testing.T, string) []string
	}{
		"get branch": {
			repoContent: []testutil.Content{
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch: "foo",
				},
			},
			refsFunc: func(*testing.T, string) []string {
				return []string{"foo"}
			},
		},
		// TODO: We should test both lightweight tags and annotated tags.
		"get tag": {
			repoContent: []testutil.Content{
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch: "foo",
					Tag:    "abc/123",
				},
			},
			refsFunc: func(*testing.T, string) []string {
				return []string{"abc/123"}
			},
		},
		"get commit": {
			repoContent: []testutil.Content{
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch: "foo",
					Tag:    "abc/123",
				},
			},
			refsFunc: func(t *testing.T, upstreamPath string) []string {
				runner, err := NewLocalGitRunner(upstreamPath)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				rr, err := runner.Run(fake.CtxWithDefaultPrinter(), "show-ref", "-s", "abc/123")
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				return []string{strings.TrimSpace(rr.Stdout)}
			},
		},
		"get short commit": {
			repoContent: []testutil.Content{
				{
					Pkg: pkgbuilder.NewRootPkg().
						WithResource(pkgbuilder.DeploymentResource),
					Branch: "foo",
					Tag:    "abc/123",
				},
			},
			refsFunc: func(t *testing.T, upstreamPath string) []string {
				runner, err := NewLocalGitRunner(upstreamPath)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				rr, err := runner.Run(fake.CtxWithDefaultPrinter(), "show-ref", "-s", "abc/123")
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				sha := strings.TrimSpace(rr.Stdout)
				rr, err = runner.Run(fake.CtxWithDefaultPrinter(), "rev-parse", "--short", sha)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				return []string{strings.TrimSpace(rr.Stdout)}
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g, _, clean := testutil.SetupReposAndWorkspace(t, map[string][]testutil.Content{
				testutil.Upstream: tc.repoContent,
			})
			defer clean()

			gur, err := NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			refs := tc.refsFunc(t, g[testutil.Upstream].RepoDirectory)
			dir, err := gur.GetRepo(fake.CtxWithDefaultPrinter(), refs)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			runner, err := NewLocalGitRunner(dir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			for _, r := range refs {
				sha, found := gur.ResolveRef(r)
				if !found {
					// Assume the ref is a commit...
					sha = r
				}
				_, err := runner.Run(fake.CtxWithDefaultPrinter(), "reset", "--hard", sha)
				assert.NoError(t, err)
			}
		})
	}
}

// Verify that we can fetch two different version of the same ref into the
// same cached repo.
func TestGitUpstreamRepo_GetRepo_multipleUpdates(t *testing.T) {
	branchName := "kpt-test"
	repoContent := map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: branchName,
			},
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.ConfigMapResource),
				Branch: branchName,
			},
		},
	}
	g, _, clean := testutil.SetupReposAndWorkspace(t, repoContent)
	defer clean()

	firstRepoDir := getRepoAndVerify(t, g[testutil.Upstream].RepoDirectory, branchName)
	_, err := os.Stat(filepath.Join(firstRepoDir, "deployment.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.NoError(t, testutil.UpdateRepos(t, g, repoContent)) {
		t.FailNow()
	}

	secondRepoDir := getRepoAndVerify(t, g[testutil.Upstream].RepoDirectory, branchName)
	_, err = os.Stat(filepath.Join(secondRepoDir, "configmap.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.Equal(t, firstRepoDir, secondRepoDir)
}

func getRepoAndVerify(t *testing.T, repo, branchName string) string {
	gur, err := NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), repo)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	dir, err := gur.GetRepo(fake.CtxWithDefaultPrinter(), []string{branchName})
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	runner, err := NewLocalGitRunner(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	sha, _ := gur.ResolveBranch(branchName)
	_, err = runner.Run(fake.CtxWithDefaultPrinter(), "reset", "--hard", sha)
	assert.NoError(t, err)

	return dir
}

func toKeys(m map[string]string) []string {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
