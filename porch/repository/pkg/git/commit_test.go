// Copyright 2022 Google LLC
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
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestPackageCommitEmptyRepo(t *testing.T) {
	tempdir := t.TempDir()
	repo := OpenGitRepositoryFromArchive(t, filepath.Join("testdata", "empty-repository.tar"), tempdir)

	parent := plumbing.ZeroHash      // Empty repository
	packageTree := plumbing.ZeroHash // Empty package
	packagePath := "catalog/namespaces/istions"
	ch, err := newCommitHelper(repo.Storer, parent, packagePath, packageTree)
	if err != nil {
		t.Fatalf("newCommitHelper(%q) failed: %v", packagePath, err)
	}

	filePath := path.Join(packagePath, "hello.txt")
	fileContents := "Hello, World!"
	if err := ch.storeFile(filePath, fileContents); err != nil {
		t.Fatalf("storeFile(%q, %q) failed: %v", filePath, fileContents, err)
	}

	message := fmt.Sprintf("Commit Message: %d", time.Now().UnixMicro())
	commitHash, treeHash, err := ch.commit(message, packagePath)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if commitHash.IsZero() {
		t.Errorf("Commit returned zero commit hash")
	}
	if treeHash.IsZero() {
		t.Errorf("Commit returned zero package tree hash")
	}

	commit := getCommitObject(t, repo, commitHash)
	if got, want := commit.Message, message; got != want {
		t.Errorf("Commit message: got %q, want %q", got, want)
	}
	root, err := commit.Tree()
	if err != nil {
		t.Fatalf("Failed to get tree from commit %q: %v", commitHash, err)
	}
	entry := findTreeEntry(t, root, packagePath)
	if got, want := entry.Hash, treeHash; got != want {
		t.Errorf("Packag tree hash: got %s, want %s", got, want)
	}
	file := findFile(t, root, filePath)
	got, err := file.Contents()
	if err != nil {
		t.Fatalf("Failed to read contents of file %q under the root commit tree %q: %v", filePath, root.Hash, err)
	}
	if want := fileContents; got != want {
		t.Errorf("File contents: got %q, want %q", got, want)
	}
}

func TestPackageCommitToMain(t *testing.T) {
	tempdir := t.TempDir()
	repo := OpenGitRepositoryFromArchive(t, filepath.Join("testdata", "drafts-repository.tar"), tempdir)

	// Commit `bucket`` package from drafts/bucket/v1 into main

	main := resolveReference(t, repo, refMain)
	packagePath := "bucket"

	// Confirm no 'bucket' package in main
	mainRoot := getCommitTree(t, repo, main.Hash())
	{
		entry, err := mainRoot.FindEntry(packagePath)
		if entry != nil || err != object.ErrEntryNotFound {
			t.Fatalf("Unexpectedly found %q package in main branch: %v, %v", packagePath, entry, err)
		}
	}
	draft := resolveReference(t, repo, plumbing.NewBranchReferenceName("drafts/bucket/v1"))
	draftTree := getCommitTree(t, repo, draft.Hash())
	bucketEntry := findTreeEntry(t, draftTree, packagePath)
	bucketTree := bucketEntry.Hash
	ch, err := newCommitHelper(repo.Storer, main.Hash(), packagePath, bucketTree)
	if err != nil {
		t.Fatalf("Failed to create commit helper: %v", err)
	}

	commitHash, treeHash, err := ch.commit("Move bucket to main", packagePath)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if commitHash.IsZero() {
		t.Errorf("Commit returned zero commit hash")
	}
	if treeHash.IsZero() {
		t.Errorf("Commit returned zero package tree hash")
	}

	commitTree := getCommitTree(t, repo, commitHash)
	packageEntry := findTreeEntry(t, commitTree, packagePath)
	if got, want := packageEntry.Hash, bucketTree; got != want {
		t.Errorf("Package copied into main branch with unexpected tree hash; got %s, want %s", got, want)
	}
}
