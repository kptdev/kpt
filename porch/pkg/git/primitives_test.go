// Copyright 2022 The kpt Authors
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

package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestUpdateRef(t *testing.T) {
	gitdir := t.TempDir()
	repo := OpenGitRepositoryFromArchiveWithWorktree(t, filepath.Join("testdata", "drafts-repository.tar"), gitdir)

	const draftReferenceName plumbing.ReferenceName = "refs/heads/drafts/bucket/v1"

	draftRef := resolveReference(t, repo, draftReferenceName)

	{
		// Crete a commit and advance the draft reference.
		commit := createTestCommit(t, repo, draftRef.Hash(), "Commit One", "one.txt", "File one")
		newDraft := plumbing.NewHashReference(draftReferenceName, commit)

		if err := repo.Storer.CheckAndSetReference(newDraft, draftRef); err != nil {
			t.Fatalf("Failed to update reference %s with check %s", newDraft, draftRef)
		}
	}

	{
		// Create another (competing) commit with draft parent;
		// we shouldn't be able to update the ref to that commit
		commit := createTestCommit(t, repo, draftRef.Hash(), "Commit Two", "two.txt", "File two")
		newDraft := plumbing.NewHashReference(draftReferenceName, commit)
		if err := repo.Storer.CheckAndSetReference(newDraft, draftRef); err == nil {
			t.Fatalf("Unexpectedly succeeded to update reference %s with check %s", newDraft, draftRef)
		} else {
			t.Logf("Expected error: %v", err)
		}
	}
}

func TestSetNewRef(t *testing.T) {
	temp := t.TempDir()
	repo := OpenGitRepositoryFromArchive(t, filepath.Join("testdata", "simple-repository.tar"), temp)

	logRefs(t, repo, "Simple: ")

	basens := resolveReference(t, repo, "refs/tags/basens/v1")

	const draftReferenceName plumbing.ReferenceName = "refs/heads/test-reference"
	zero := plumbing.NewHashReference(draftReferenceName, plumbing.ZeroHash)
	test := plumbing.NewHashReference(draftReferenceName, basens.Hash())

	if err := repo.Storer.CheckAndSetReference(test, zero); err != nil {
		t.Errorf("CheckAndSetReference(%s, zero) failed: %v", test, err)
	}

	// Try again, this time it should fail.
	if err := repo.Storer.CheckAndSetReference(test, zero); err == nil {
		t.Errorf("Second CheckAndSetReference(%s, zero) unexpectedly succeeded", test)
	}
}

func TestSimpleFetch(t *testing.T) {
	upstreamDir := t.TempDir()
	downstreamDir := t.TempDir()
	upstream, address := ServeGitRepository(t, filepath.Join("testdata", "drafts-repository.tar"), upstreamDir)
	downstream := initRepositoryWithRemote(t, downstreamDir, address)

	const remoteDraftReferenceName = "refs/remotes/origin/drafts/bucket/v1"

	originRef := resolveReference(t, upstream, "refs/heads/drafts/bucket/v1")
	refMustNotExist(t, downstream, remoteDraftReferenceName)
	fetch(t, downstream)

	clonedRef := resolveReference(t, downstream, remoteDraftReferenceName)
	if got, want := clonedRef.Hash(), originRef.Hash(); got != want {
		t.Errorf("%s after fetch; got %s, want %s", remoteDraftReferenceName, clonedRef, originRef)
	}

	logRefs(t, downstream, "Fetched: ")
}

func TestSimplePush(t *testing.T) {
	upstreamDir := t.TempDir()
	downstreamDir := t.TempDir()

	upstream, address := ServeGitRepository(t, filepath.Join("testdata", "drafts-repository.tar"), upstreamDir)
	downstream := initRepositoryWithRemote(t, downstreamDir, address)
	fetch(t, downstream)

	const (
		draftReferenceName       plumbing.ReferenceName = "refs/heads/drafts/bucket/v1"
		remoteDraftReferenceName plumbing.ReferenceName = "refs/remotes/origin/drafts/bucket/v1"
	)

	draftRef := resolveReference(t, downstream, remoteDraftReferenceName)

	var commit plumbing.Hash
	{
		// Create a first commit in test branch
		commit = createTestCommit(t, downstream, draftRef.Hash(), "Draft Commit", "readme.txt", "Hello, World!")
		if err := downstream.Push(&git.PushOptions{
			RemoteName: OriginName,
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("%s:%s", commit, draftReferenceName)),
			},
			RequireRemoteRefs: []config.RefSpec{},
		}); err != nil {
			t.Fatalf("Push failed: %v", err)
		}

		// Verify draft advanced
		originDraft := resolveReference(t, upstream, draftReferenceName)
		if got, want := originDraft.Hash(), commit; got != want {
			t.Errorf("Updated draft reference at origin: %s, got %s, want %s", originDraft, got, want)
		}
	}

	{
		// Create a competing concurrent in a test branch
		concurrent := createTestCommit(t, downstream, draftRef.Hash(), "Competing Commit", "test.txt", "competing commit")
		switch err := downstream.Push(&git.PushOptions{
			RemoteName: OriginName,
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("%s:%s", concurrent, draftReferenceName)),
			},
			RequireRemoteRefs: []config.RefSpec{},
		}); {
		case err == git.ErrNonFastForwardUpdate:
			// ok
		case err == nil:
			t.Fatalf("Second push unexpectedly succeeded")
		case strings.Contains(err.Error(), "non-fast-forward update"):
			// ok
			// TODO: preferably we get strongly typed error here...
		default:
			t.Fatalf("Unexpected error when pushing competing commit: %v", err)
		}
	}

	// Verify the commit in both repositories to point at expected value (first commit)
	originDraftRef := resolveReference(t, upstream, draftReferenceName)
	localDraftRef := resolveReference(t, downstream, remoteDraftReferenceName)

	if got, want := originDraftRef.Hash(), commit; got != want {
		t.Errorf("Updated draft reference at origin: %s, got %s, want %s", originDraftRef, got, want)
	}
	if got, want := localDraftRef.Hash(), commit; got != want {
		t.Errorf("Updated draft reference in local repo: %s, got %s, want %s", localDraftRef, got, want)
	}
}

// Test concurrent tag pushes.
func TestFinalPush(t *testing.T) {
	upstreamDir := t.TempDir()
	downstreamDir := t.TempDir()
	upstream, address := ServeGitRepository(t, filepath.Join("testdata", "drafts-repository.tar"), upstreamDir)
	downstream := initRepositoryWithRemote(t, downstreamDir, address)

	fetch(t, downstream)

	const (
		mainReferenceName       plumbing.ReferenceName = "refs/heads/main"
		remoteMainReferenceName plumbing.ReferenceName = "refs/remotes/origin/main"
		packageTagReferenceName plumbing.ReferenceName = "refs/tags/package/v1"
	)

	main := resolveReference(t, downstream, remoteMainReferenceName)

	var commit plumbing.Hash
	{
		// Create first commit and tag (finalized package)
		commit = createTestCommit(t, downstream, main.Hash(), "Package One", "one.txt", "initial")
		if err := downstream.Push(&git.PushOptions{
			RemoteName: OriginName,
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("%s:%s", commit, mainReferenceName)),
				config.RefSpec(fmt.Sprintf("%s:%s", commit, packageTagReferenceName)),
			},
			RequireRemoteRefs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("%s:%s", main.Hash(), mainReferenceName)),
			},
		}); err != nil {
			t.Fatalf("Push failed: %v", err)
		}

		if tag, err := downstream.Reference(packageTagReferenceName, false); err != nil {
			t.Errorf("Failed to find pushed tag")
		} else if got, want := tag.Hash(), commit; got != want {
			t.Errorf("Tag hash after push: got %s, want %s", got, want)
		}
	}

	{
		// Create a competing concurrent finalized package.
		concurrent := createTestCommit(t, downstream, main.Hash(), "Package One", "one.txt", "concurrent")

		// Simulated concurrent push should fail
		switch err := downstream.Push(&git.PushOptions{
			RemoteName: OriginName,
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("%s:%s", concurrent, mainReferenceName)),
			},
			RequireRemoteRefs: []config.RefSpec{},
		}); {
		case err == git.ErrNonFastForwardUpdate:
			// ok
		case err == nil:
			t.Fatalf("Second push unexpectedly succeeded")
		case strings.Contains(err.Error(), "non-fast-forward update: refs/heads/main"):
			// ok
		default:
			t.Fatalf("Unexpected error pushing concurrent commit: %v", err)
		}

		// Liewise, push to the tag should fail
		switch err := downstream.Push(&git.PushOptions{
			RemoteName: OriginName,
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("%s:%s", concurrent, packageTagReferenceName)),
			},
			RequireRemoteRefs: []config.RefSpec{},
		}); {
		case err == git.ErrNonFastForwardUpdate:
			// ok
		case err == nil:
			t.Fatalf("Second push unexpectedly succeeded")
		case strings.Contains(err.Error(), "non-fast-forward update: refs/tags/package/v1"):
			// ok
		default:
			t.Fatalf("Unexpected error pushing concurrent commit: %v", err)
		}

	}

	// Double check that the upstream main is the expected commit
	upstreamMain := resolveReference(t, upstream, mainReferenceName)
	if got, want := upstreamMain.Hash(), commit; got != want {
		t.Errorf("Upstream main %s after push: got %s, want %s", upstreamMain, got, want)
	}
}

// Simulate case where a remote ref (refs/remotes/origin/...) is out of sync
// with the remote repository and will be force-overwritten on fetch.
func TestRepoRecovery(t *testing.T) {
	upstreamDir := t.TempDir()
	downstreamDir := t.TempDir()
	upstream, address := ServeGitRepository(t, filepath.Join("testdata", "drafts-repository.tar"), upstreamDir)
	downstream := initRepositoryWithRemote(t, downstreamDir, address)

	const (
		draftReferenceName       plumbing.ReferenceName = "refs/heads/drafts/bucket/v1"
		remoteDraftReferenceName plumbing.ReferenceName = "refs/remotes/origin/drafts/bucket/v1"
		istionsReferenceName     plumbing.ReferenceName = "refs/tags/istions/v1"
	)

	fetch(t, downstream)

	upstreamDraftRef := resolveReference(t, upstream, draftReferenceName)

	// Simulate repository data corruption - reset remoteDraftReferenceName to another commit
	// We will create a new commit for the draft with the shared parent.
	draftRef := resolveReference(t, downstream, remoteDraftReferenceName)
	draftCommit := getCommitObject(t, downstream, draftRef.Hash())
	parent, err := draftCommit.Parent(0)
	if err != nil {
		t.Fatalf("Failed to get parent of commit %s: %v", draftRef, err)
	}

	conflicting := createTestCommit(t, downstream, parent.Hash, "Conflicting Commit", "conflict.txt", "file contents")
	// Overwrite the remote ref in the downstream repository
	newRef := plumbing.NewHashReference(remoteDraftReferenceName, conflicting)
	if err := downstream.Storer.CheckAndSetReference(newRef, draftRef); err != nil {
		t.Fatalf("Failed to intentionally overwrite a remote reference %s: %v", newRef, err)
	}

	// Corrupt the istions reference (tag) also
	istioRef := resolveReference(t, downstream, istionsReferenceName)
	istioNew := plumbing.NewHashReference(istionsReferenceName, conflicting)
	if err := downstream.Storer.CheckAndSetReference(istioNew, istioRef); err != nil {
		t.Fatalf("Failed to intentionally overwrite a tag: %s: %v", istioNew, err)
	}

	// Perhaps overly cautious; check the reference value
	{
		ref := resolveReference(t, downstream, remoteDraftReferenceName)
		if got, want := ref.Hash(), conflicting; got != want {
			t.Errorf("Unexpected ref value %s after overwrite; got %s, want %s", ref, got, want)
		}
	}

	// Re-fetch. Expect ref to go back
	fetch(t, downstream)

	// Re-resolve the corrupted reference
	{
		ref := resolveReference(t, downstream, remoteDraftReferenceName)
		if got, want := ref.Hash(), draftRef.Hash(); got != want {
			t.Errorf("Ref %s was not reset by re-fetch; got %s, want %s", ref, got, want)
		}

		// Check also against the upstreamDraftRef
		if got, want := ref.Hash(), upstreamDraftRef.Hash(); got != want {
			t.Errorf("Ref %s was reset to an unexpected value, not matching upstream repository; got %s, want %s", ref, got, want)
		}
	}

	// Re-resolve the tag
	{
		ref := resolveReference(t, downstream, istionsReferenceName)
		if got, want := ref.Hash(), istioRef.Hash(); got != want {
			t.Errorf("Tag %s was not reset by re-fetch; got %s, want %s", ref, got, want)
		}
	}
}

func TestProposal(t *testing.T) {
	upstreamDir := t.TempDir()
	downstreamDir := t.TempDir()
	upstream, address := ServeGitRepository(t, filepath.Join("testdata", "drafts-repository.tar"), upstreamDir)
	downstream := initRepositoryWithRemote(t, downstreamDir, address)

	fetch(t, downstream)

	const (
		draftRef            plumbing.ReferenceName = "refs/remotes/origin/drafts/bucket/v1"
		proposedRef         plumbing.ReferenceName = "refs/remotes/origin/proposed/bucket/v1"
		upstreamDraftRef    plumbing.ReferenceName = "refs/heads/drafts/bucket/v1"
		upstreamProposedRef plumbing.ReferenceName = "refs/heads/proposed/bucket/v1"
	)

	bucket := resolveReference(t, downstream, draftRef)

	// Simulate changing package to proposed
	if err := downstream.Push(&git.PushOptions{
		RemoteName: OriginName,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:%s", bucket.Hash(), upstreamProposedRef)),
			config.RefSpec(fmt.Sprintf(":%s", upstreamDraftRef)),
		},
		RequireRemoteRefs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:%s", bucket.Hash(), upstreamDraftRef)),
		},
	}); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	refMustNotExist(t, downstream, draftRef)

	// Verify upstream
	{
		ref := resolveReference(t, upstream, upstreamProposedRef)
		if got, want := ref.Hash(), bucket.Hash(); got != want {
			t.Errorf("Proposed in upstream: got %s, want %s", got, want)
		}

		refMustNotExist(t, upstream, upstreamDraftRef)
	}
}

func TestDeleteUpstreamBranches(t *testing.T) {
	upstreamDir := t.TempDir()
	downstreamDir := t.TempDir()
	upstream, address := ServeGitRepository(t, filepath.Join("testdata", "drafts-repository.tar"), upstreamDir)
	downstream := initRepositoryWithRemote(t, downstreamDir, address)

	logRefs(t, downstream, "Init: ")

	fetch(t, downstream)
	logRefs(t, downstream, "Before: ")

	// Delete upstream tags and branches
	forEachRef(t, upstream, func(ref *plumbing.Reference) error {
		if ref.Name() != DefaultMainReferenceName { // keep main
			return upstream.Storer.RemoveReference(ref.Name())
		}
		return nil
	})

	logRefs(t, upstream, "Upstream: ")

	// Refetch
	switch err := downstream.Fetch(&git.FetchOptions{
		RemoteName: OriginName,
		Tags:       git.NoTags,
		Prune:      git.Prune,
	}); err {
	case nil, git.NoErrAlreadyUpToDate:
		// ok
	default:
		t.Fatalf("Fetch failed: %s", err)
	}

	// Make sure the local refs were deleted.
	forEachRef(t, downstream, func(ref *plumbing.Reference) error {
		switch ref.Name() {
		case "HEAD", "refs/remotes/origin/main":
			// ok
		default:
			return fmt.Errorf("found unexpected reference %s", ref)
		}
		return nil
	})

	logRefs(t, downstream, "After:")
}

func initRepositoryWithRemote(t *testing.T, dir, address string) *git.Repository {
	repo := InitEmptyRepositoryWithWorktree(t, dir)

	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name:  OriginName,
		URLs:  []string{address},
		Fetch: defaultFetchSpec,
	}); err != nil {
		t.Fatalf("CreateRemote failed: %v", err)
	}
	return repo
}

func createTestCommit(t *testing.T, repo *git.Repository, parent plumbing.Hash, message, name, contents string) plumbing.Hash {
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed getting worktree: %v", err)
	}
	if err := wt.Checkout(&git.CheckoutOptions{
		Hash:  parent,
		Force: true,
		Keep:  false,
	}); err != nil {
		t.Fatalf("Failed checking out worktree: %v", err)
	}

	f, err := wt.Filesystem.Create(name)
	if err != nil {
		t.Fatalf("Failed creating file: %v", err)
	}
	defer f.Close()
	if _, err := f.Write([]byte(contents)); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	if _, err := wt.Add(name); err != nil {
		t.Fatalf("Failed to add file to index: %v", err)
	}
	sig := object.Signature{
		Name:  "Test",
		Email: "test@kpt.dev",
		When:  time.Now(),
	}
	commit, err := wt.Commit("Hello", &git.CommitOptions{
		Author:    &sig,
		Committer: &sig,
		Parents:   []plumbing.Hash{parent},
	})
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	return commit
}
