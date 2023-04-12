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
	"archive/tar"
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func OpenGitRepositoryFromArchive(t *testing.T, tarfile, tempdir string) *gogit.Repository {
	t.Helper()

	extractTar(t, tarfile, tempdir)

	git, err := gogit.PlainOpen(filepath.Join(tempdir, ".git"))
	if err != nil {
		t.Fatalf("Failed to open Git Repository extracted from %q: %v", tarfile, err)
	}

	return git
}

func OpenGitRepositoryFromArchiveWithWorktree(t *testing.T, tarfile, path string) *gogit.Repository {
	t.Helper()

	extractTar(t, tarfile, path)

	repo, err := gogit.PlainOpen(path)
	if err != nil {
		t.Fatalf("Failed to open Git repository extracted from %q: %v", tarfile, err)
	}
	return repo
}

func InitEmptyRepositoryWithWorktree(t *testing.T, path string) *gogit.Repository {
	t.Helper()

	repo, err := gogit.PlainInit(path, false)
	if err != nil {
		t.Fatalf("Failed to initialize empty Git repository: %v", err)
	}
	if err := initializeDefaultBranches(repo); err != nil {
		t.Fatalf("Failed to remove default branches")
	}
	return repo
}

func ServeGitRepositoryWithBranch(t *testing.T, tarfile, tempdir, branch string) (*gogit.Repository, string) {
	t.Helper()

	git := OpenGitRepositoryFromArchive(t, tarfile, tempdir)
	InitializeBranch(t, git, branch)
	return git, ServeExistingRepository(t, git)
}

func InitializeBranch(t *testing.T, git *gogit.Repository, branch string) {
	t.Helper()

	// If main branch exists, rename it to the specified ref
	main, err := git.Reference(DefaultMainReferenceName, false)
	switch err {
	case nil:
		// found `main branch`
	case plumbing.ErrReferenceNotFound:
		// main doesn't exist, we won't create the target branch either.
		return
	default:
		t.Fatalf("Error getting %s branch: %v", DefaultMainReferenceName, err)
		return
	}

	// `main` branch was found. Create the target branch off of it if needed.
	name := plumbing.NewBranchReferenceName(branch)
	if name != DefaultMainReferenceName {
		ref := plumbing.NewHashReference(name, main.Hash())
		if err := git.Storer.SetReference(ref); err != nil {
			t.Fatalf("Error creating target branch %q from %q: %v", ref, main, err)
		}

		t.Cleanup(func() {
			// Verify that main didn't move during the test
			new, err := git.Reference(DefaultMainReferenceName, false)
			if err != nil {
				t.Fatalf("Error getting %s branch after test run: %v", gogit.DefaultRemoteName, err)
			}

			if main.Hash() != new.Hash() {
				t.Fatalf("%q branch moved unexpectedly during the test to %q", main, new)
			}
		})
	}
}

func ServeGitRepository(t *testing.T, tarfile, tempdir string) (*gogit.Repository, string) {
	t.Helper()

	git := OpenGitRepositoryFromArchive(t, tarfile, tempdir)
	return git, ServeExistingRepository(t, git)
}

func ServeExistingRepository(t *testing.T, git *gogit.Repository) string {
	t.Helper()

	repo, err := NewRepo(git)
	if err != nil {
		t.Fatalf("NewRepo failed: %v", err)
	}

	key := "default"

	repos := NewStaticRepos()
	if err := repos.Add(key, repo); err != nil {
		t.Fatalf("repos.Add failed: %v", err)
	}

	server, err := NewGitServer(repos)
	if err != nil {
		t.Fatalf("NewGitServer() failed: %v", err)
	}

	var wg sync.WaitGroup

	serverAddressChannel := make(chan net.Addr)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(ctx, "127.0.0.1:0", serverAddressChannel); err != nil {
			if ctx.Err() == nil {
				t.Errorf("Git Server ListenAndServe failed: %v", err)
			}
		}
	}()

	address, ok := <-serverAddressChannel
	if !ok {
		t.Fatalf("Git Server failed to start")
	}
	return "http://" + address.String() + "/" + key
}

func extractTar(t *testing.T, tarfile string, dir string) {
	t.Helper()

	reader, err := os.Open(tarfile)
	if err != nil {
		t.Fatalf("Open(%q) failed: %v", tarfile, err)
	}
	defer reader.Close()
	tr := tar.NewReader(reader)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Reading tar file %q failed: %v", tarfile, err)
		}
		if hdr.FileInfo().IsDir() {
			path := filepath.Join(dir, hdr.Name)
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatalf("MkdirAll(%q) failed: %v", path, err)
			}
			continue
		}
		path := filepath.Join(dir, filepath.Dir(hdr.Name))
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("MkdirAll(%q) failed: %v", path, err)
		}
		path = filepath.Join(dir, hdr.Name)
		saveToFile(t, path, tr)
	}
}

func saveToFile(t *testing.T, path string, src io.Reader) {
	t.Helper()

	dst, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) failed; %v", path, err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("Copy from tar to %q failed: %v", path, err)
	}
}

func resolveReference(t *testing.T, repo *gogit.Repository, name plumbing.ReferenceName) *plumbing.Reference {
	t.Helper()

	ref, err := repo.Reference(name, true)
	if err != nil {
		t.Fatalf("Failed to resolve %q: %v", name, err)
	}
	return ref
}

func getCommitObject(t *testing.T, repo *gogit.Repository, hash plumbing.Hash) *object.Commit {
	t.Helper()

	commit, err := repo.CommitObject(hash)
	if err != nil {
		t.Fatalf("Failed to find commit object for %q: %v", hash, err)
	}
	return commit
}

func getCommitTree(t *testing.T, repo *gogit.Repository, hash plumbing.Hash) *object.Tree {
	t.Helper()

	commit := getCommitObject(t, repo, hash)
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("Failed to get tree from commit %q: %v", hash, err)
	}
	return tree
}

func findTreeEntry(t *testing.T, tree *object.Tree, path string) *object.TreeEntry {
	t.Helper()

	entry, err := tree.FindEntry(path)
	if err != nil {
		t.Fatalf("Failed to find path %q in tree %q: %v", path, tree.Hash, err)
	}
	return entry
}

func findFile(t *testing.T, tree *object.Tree, path string) *object.File {
	t.Helper()

	file, err := tree.File(path)
	if err != nil {
		t.Fatalf("Failed to find file %q under the root commit tree %q: %v", path, tree.Hash, err)
	}
	return file
}

func findPackageRevision(t *testing.T, revisions []repository.PackageRevision, key repository.PackageRevisionKey) repository.PackageRevision {
	t.Helper()

	for _, r := range revisions {
		if r.Key() == key {
			return r
		}
	}
	t.Fatalf("PackageRevision %q not found", key)
	return nil
}

func packageMustNotExist(t *testing.T, revisions []repository.PackageRevision, key repository.PackageRevisionKey) {
	t.Helper()

	for _, r := range revisions {
		if key == r.Key() {
			t.Fatalf("PackageRevision %q expected to not exist was found", key)
		}
	}
}

func repositoryMustHavePackageRevision(t *testing.T, git GitRepository, name repository.PackageRevisionKey) {
	t.Helper()

	list, err := git.ListPackageRevisions(context.Background(), repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}
	findPackageRevision(t, list, name)
}

func repositoryMustNotHavePackageRevision(t *testing.T, git GitRepository, name repository.PackageRevisionKey) {
	t.Helper()

	list, err := git.ListPackageRevisions(context.Background(), repository.ListPackageRevisionFilter{})
	if err != nil {
		t.Fatalf("ListPackageRevisions failed: %v", err)
	}
	packageMustNotExist(t, list, name)
}

func refMustExist(t *testing.T, repo *gogit.Repository, name plumbing.ReferenceName) {
	t.Helper()

	switch _, err := repo.Reference(name, false); err {
	case nil:
		// ok
	case plumbing.ErrReferenceNotFound:
		t.Fatalf("Reference %s must exist but was not found: %v", name, err)
	default:
		t.Fatalf("Unexpected error resolving reference %q: %v", name, err)
	}
}

func refMustNotExist(t *testing.T, repo *gogit.Repository, name plumbing.ReferenceName) {
	t.Helper()

	switch ref, err := repo.Reference(name, false); err {
	case nil:
		t.Fatalf("Reference %s must not exist but was found: %s", name, ref)
	case plumbing.ErrReferenceNotFound:
		// ok
	default:
		t.Fatalf("Unexpected error resolving reference %q: %v", name, err)
	}
}

func forEachRef(t *testing.T, repo *gogit.Repository, fn func(*plumbing.Reference) error) {
	t.Helper()

	refs, err := repo.References()
	if err != nil {
		t.Fatalf("Failed to create references iterator: %v", err)
	}
	if err := refs.ForEach(fn); err != nil {
		t.Fatalf("References.ForEach failed: %v", err)
	}
}

func logRefs(t *testing.T, repo *gogit.Repository, logPrefix string) {
	t.Helper()

	forEachRef(t, repo, func(ref *plumbing.Reference) error {
		t.Logf("%s%s", logPrefix, ref)
		return nil
	})
}

func fetch(t *testing.T, repo *gogit.Repository) {
	t.Helper()

	switch err := repo.Fetch(&gogit.FetchOptions{
		RemoteName: OriginName,
		Tags:       gogit.NoTags,
	}); err {
	case nil, gogit.NoErrAlreadyUpToDate:
		// ok
	default:
		t.Fatalf("Fetch failed: %s", err)
	}
}
