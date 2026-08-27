// Copyright 2021,2026 The kpt Authors
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

	internalgitutil "github.com/kptdev/kpt/internal/gitutil"
	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/internal/testutil/pkgbuilder"
	"github.com/kptdev/kpt/pkg/lib/errors"
	"github.com/kptdev/kpt/pkg/printer/fake"
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
		expectedErr    *internalgitutil.GitExecError
	}{
		"successful command with output to stdout": {
			command:        "branch",
			args:           []string{"--show-current"},
			expectedStdout: "main",
		},
		"failed command with output to stderr": {
			command: "checkout",
			args:    []string{"does-not-exist"},
			expectedErr: &internalgitutil.GitExecError{
				StdOut: "",
				StdErr: "error: pathspec 'does-not-exist' did not match any file(s) known to git",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			dir := t.TempDir()

			runner, err := internalgitutil.NewLocalGitRunner(dir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			_, err = runner.Run(fake.CtxWithDefaultPrinter(), "init", "--initial-branch=main")
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			rr, err := runner.Run(fake.CtxWithDefaultPrinter(), tc.command, tc.args...)
			if tc.expectedErr != nil {
				var gitExecError *internalgitutil.GitExecError
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

	_, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), dir)
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "does not appear to be a git repository")
}

func TestNewGitUpstreamRepo_noRefs(t *testing.T) {
	dir := t.TempDir()

	runner, err := internalgitutil.NewLocalGitRunner(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = runner.Run(fake.CtxWithDefaultPrinter(), "init", "--bare")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 0, len(gur.Heads()))
	assert.Equal(t, 0, len(gur.Tags()))
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
		sort.Strings(tc.expectedHeads)
		sort.Strings(tc.expectedTags)
		t.Run(tn, func(t *testing.T) {
			repoContent := map[string][]testutil.Content{
				testutil.Upstream: tc.repoContent,
			}
			g, _, clean := testutil.SetupReposAndWorkspace(t, repoContent)
			defer clean()
			if !assert.NoError(t, testutil.UpdateRepos(t, g, repoContent)) {
				t.FailNow()
			}

			gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.EqualValues(t, tc.expectedHeads, gur.Heads())
			assert.EqualValues(t, tc.expectedTags, gur.Tags())
		})
	}
}

func TestGitUpstreamRepo_GetDefaultBranch_noRefs(t *testing.T) {
	dir := t.TempDir()

	runner, err := internalgitutil.NewLocalGitRunner(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = runner.Run(fake.CtxWithDefaultPrinter(), "init", "--bare")
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), dir)
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

			gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
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
				runner, err := internalgitutil.NewLocalGitRunner(upstreamPath)
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
				runner, err := internalgitutil.NewLocalGitRunner(upstreamPath)
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

			gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			refs := tc.refsFunc(t, g[testutil.Upstream].RepoDirectory)
			dir, err := gur.GetRepo(fake.CtxWithDefaultPrinter(), refs)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			runner, err := internalgitutil.NewLocalGitRunner(dir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			for _, r := range refs {
				sha := gur.ResolveRef(r)
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

	gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	firstRepoDir := getRepoAndVerify(t, gur, branchName)
	_, err = os.Stat(filepath.Join(firstRepoDir, "deployment.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.NoError(t, testutil.UpdateRepos(t, g, repoContent)) {
		t.FailNow()
	}

	secondRepoDir := getRepoAndVerify(t, gur, branchName)
	_, err = os.Stat(filepath.Join(secondRepoDir, "configmap.yaml"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.Equal(t, firstRepoDir, secondRepoDir)
}

func getRepoAndVerify(t *testing.T, gur internalgitutil.GitUpstreamRepo, branchName string) string {
	dir, err := gur.GetRepo(fake.CtxWithDefaultPrinter(), []string{branchName})
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	runner, err := internalgitutil.NewLocalGitRunner(dir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	sha, _ := gur.ResolveBranch(branchName)
	_, err = runner.Run(fake.CtxWithDefaultPrinter(), "reset", "--hard", sha)
	assert.NoError(t, err)

	return dir
}

// TestGitUpstreamRepo_GetRepo_rejectsOptionInjection makes sure that a ref that
// git would interpret as a command-line option (e.g. one read from an
// attacker-controlled remote sub-package Kptfile) is rejected rather than
// passed to git, preventing argument injection.
func TestGitUpstreamRepo_GetRepo_rejectsOptionInjection(t *testing.T) {
	repoContent := map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: "main",
			},
		},
	}
	g, _, clean := testutil.SetupReposAndWorkspace(t, repoContent)
	defer clean()

	gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	_, err = gur.GetRepo(fake.CtxWithDefaultPrinter(), []string{"--output=../../../.bashrc"})
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "must not begin with '-'")
}

// TestNewGitUpstreamRepo_rejectsOptionInjectionURI makes sure that a repo URI
// that git would interpret as a command-line option (e.g. one read from an
// attacker-controlled Kptfile upstream block) is rejected before it is ever
// handed to git. Without this guard a value like `--upload-pack=<cmd>` would be
// treated as an option to `git fetch`/`git ls-remote` and could execute an
// arbitrary command.
func TestNewGitUpstreamRepo_rejectsOptionInjectionURI(t *testing.T) {
	testCases := map[string]string{
		"upload-pack argument injection": "--upload-pack=touch /tmp/kpt-pwned",
		"short option":                   "-o",
		"long option with value":         "--output=/tmp/kpt-pwned",
	}

	for tn, uri := range testCases {
		t.Run(tn, func(t *testing.T) {
			_, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), uri)
			if !assert.Error(t, err) {
				t.FailNow()
			}
			assert.Contains(t, err.Error(), "must not begin with '-'")
		})
	}
}

// TestGitUpstreamRepo_GetRepo_rejectsOptionInjectionVariants exercises a range
// of option-injection payloads (including one hidden after a legitimate-looking
// ref) to ensure every ref is validated before any of them reaches git.
func TestGitUpstreamRepo_GetRepo_rejectsOptionInjectionVariants(t *testing.T) {
	testCases := map[string]struct {
		refs []string
	}{
		"upload-pack argument injection": {
			refs: []string{"--upload-pack=touch /tmp/kpt-pwned"},
		},
		"short option injection": {
			refs: []string{"-o"},
		},
		"output file write via show": {
			refs: []string{"--output=/tmp/kpt-injection"},
		},
		"malicious ref after a valid one": {
			refs: []string{"main", "--upload-pack=evil"},
		},
	}

	repoContent := map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: "main",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			g, _, clean := testutil.SetupReposAndWorkspace(t, repoContent)
			defer clean()

			gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			_, err = gur.GetRepo(fake.CtxWithDefaultPrinter(), tc.refs)
			if !assert.Error(t, err) {
				t.FailNow()
			}
			assert.Contains(t, err.Error(), "must not begin with '-'")
		})
	}
}

// TestGitUpstreamRepo_GetRepo_optionInjectionHasNoSideEffect verifies that a
// rejected `--output=<file>` payload never causes git to write the file, proving
// the option was never passed to git rather than merely surfacing an error.
func TestGitUpstreamRepo_GetRepo_optionInjectionHasNoSideEffect(t *testing.T) {
	repoContent := map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: "main",
			},
		},
	}
	g, _, clean := testutil.SetupReposAndWorkspace(t, repoContent)
	defer clean()

	gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	victim := filepath.Join(t.TempDir(), "victim.txt")
	_, err = gur.GetRepo(fake.CtxWithDefaultPrinter(), []string{"--output=" + victim})
	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "must not begin with '-'")

	_, statErr := os.Stat(victim)
	assert.True(t, os.IsNotExist(statErr),
		"injected --output must not have caused git to create a file")
}

// TestGitUpstreamRepo_GetRepo_staleBranchRefetch verifies that when a branch is
// moved to a new commit upstream after the broker was created, a subsequent
// GetRepo fetches the new commit rather than serving the stale commit that the
// branch pointed to at construction time. It also asserts that fetchedRefs is
// keyed on the resolved commit SHA, so both the old and new commits are tracked.
func TestGitUpstreamRepo_GetRepo_staleBranchRefetch(t *testing.T) {
	branchName := "kpt-test-stale"
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

	gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	firstRepoDir := getRepoAndVerify(t, gur, branchName)
	firstSha, ok := gur.ResolveBranch(branchName)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	// The first commit's content should be present, the second's should not.
	_, err = os.Stat(filepath.Join(firstRepoDir, "deployment.yaml"))
	assert.NoError(t, err)
	// fetchedRefs is keyed by the resolved commit SHA, not the branch name.
	assert.Equal(t, []string{firstSha}, gur.GetFetchedRefs())

	// Move the branch to a new commit upstream.
	if !assert.NoError(t, testutil.UpdateRepos(t, g, repoContent)) {
		t.FailNow()
	}

	secondRepoDir := getRepoAndVerify(t, gur, branchName)
	secondSha, ok := gur.ResolveBranch(branchName)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	// The branch must now resolve to a different commit than before.
	assert.NotEqual(t, firstSha, secondSha)
	// The cache dir is reused, but it must now contain the new commit's content.
	assert.Equal(t, firstRepoDir, secondRepoDir)
	_, err = os.Stat(filepath.Join(secondRepoDir, "configmap.yaml"))
	assert.NoError(t, err)

	// Both the stale and current commits should be recorded as fetched, proving
	// the moved branch was fetched again instead of skipped as already-fetched.
	fetched := gur.GetFetchedRefs()
	assert.Len(t, fetched, 2)
	assert.Contains(t, fetched, firstSha)
	assert.Contains(t, fetched, secondSha)
}

// TestGitUpstreamRepo_GetRepo_idempotentFetch verifies that fetching the same
// unchanged branch twice does not re-fetch it or add a duplicate entry to
// fetchedRefs, i.e. the commit-keyed cache is honored when nothing has moved.
func TestGitUpstreamRepo_GetRepo_idempotentFetch(t *testing.T) {
	branchName := "kpt-test-idempotent"
	repoContent := map[string][]testutil.Content{
		testutil.Upstream: {
			{
				Pkg: pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource),
				Branch: branchName,
			},
		},
	}
	g, _, clean := testutil.SetupReposAndWorkspace(t, repoContent)
	defer clean()

	gur, err := internalgitutil.NewGitUpstreamRepo(fake.CtxWithDefaultPrinter(), g[testutil.Upstream].RepoDirectory)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	getRepoAndVerify(t, gur, branchName)
	firstFetched := gur.GetFetchedRefs()
	if !assert.Len(t, firstFetched, 1) {
		t.FailNow()
	}

	// A second GetRepo of the same, unchanged branch must not add a new entry.
	getRepoAndVerify(t, gur, branchName)
	secondFetched := gur.GetFetchedRefs()
	assert.Equal(t, firstFetched, secondFetched)
	assert.Len(t, secondFetched, 1)
}
