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
	"archive/tar"
	"context"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestDataAbs(t *testing.T, rel string) string {
	testdata, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("Failed to find testdata: %v", err)
	}
	return testdata
}

func CreateTestTempDir(t *testing.T) string {
	tempdir, err := ioutil.TempDir("", "test-git-*")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempdir); err != nil {
			t.Errorf("RemoveAll(%q) failed: %v", tempdir, err)
		}
	})
	return tempdir
}

func OpenGitRepositoryFromArchive(t *testing.T, tarfile, tempdir string) *gogit.Repository {
	extractTar(t, tarfile, tempdir)

	git, err := gogit.PlainOpen(filepath.Join(tempdir, ".git"))
	if err != nil {
		t.Fatalf("Failed to open Git Repository extracted from %q: %v", tarfile, err)
	}

	return git
}

func ServeGitRepository(t *testing.T, tarfile, tempdir string) string {
	git := OpenGitRepositoryFromArchive(t, tarfile, tempdir)

	server, err := NewGitServer(git)
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
	return "http://" + address.String()
}

func extractTar(t *testing.T, tarfile string, dir string) {
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
	ref, err := repo.Reference(name, true)
	if err != nil {
		t.Fatalf("Failed to resolve %q: %v", name, err)
	}
	return ref
}

func getCommitObject(t *testing.T, repo *gogit.Repository, hash plumbing.Hash) *object.Commit {
	commit, err := repo.CommitObject(hash)
	if err != nil {
		t.Fatalf("Failed to find commit object for %q: %v", hash, err)
	}
	return commit
}

func getCommitTree(t *testing.T, repo *gogit.Repository, hash plumbing.Hash) *object.Tree {
	commit := getCommitObject(t, repo, hash)
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("Failed to get tree from commit %q: %v", hash, err)
	}
	return tree
}

func findTreeEntry(t *testing.T, tree *object.Tree, path string) *object.TreeEntry {
	entry, err := tree.FindEntry(path)
	if err != nil {
		t.Fatalf("Failed to find path %q in tree %q: %v", path, tree.Hash, err)
	}
	return entry
}

func findFile(t *testing.T, tree *object.Tree, path string) *object.File {
	file, err := tree.File(path)
	if err != nil {
		t.Fatalf("Failed to find file %q under the root commit tree %q: %v", path, tree.Hash, err)
	}
	return file
}
