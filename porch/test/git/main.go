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

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/pkg/git"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"
)

var (
	port = flag.Int("port", 9446, "Server port")
)

func main() {
	klog.InitFlags(nil)

	flag.Parse()

	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v", err)
	}
}

func run(dirs []string) error {
	var dir string

	switch len(dirs) {
	case 0:
		var err error
		dir, err = os.MkdirTemp("", "repo-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory for git repository: %w", err)
		}

	case 1:
		dir = dirs[0]

	default:
		return fmt.Errorf("can server only one git repository, not %d", len(dirs))
	}

	var repo *gogit.Repository
	var err error

	if repo, err = gogit.PlainOpen(dir); err != nil {
		if err != gogit.ErrRepositoryNotExists {
			return fmt.Errorf("failed to open git repository %q: %w", dir, err)
		}
		isBare := true
		repo, err = gogit.PlainInit(dir, isBare)
		if err != nil {
			return fmt.Errorf("failed to initialize git repository %q: %w", dir, err)
		}
		if err := createEmptyCommit(repo); err != nil {
			return err
		}

		// Delete go-git default branch
		_ = repo.Storer.RemoveReference(plumbing.Master)
	}

	server, err := git.NewGitServer(repo)
	if err != nil {
		return fmt.Errorf("filed to initialize git server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addressChannel := make(chan net.Addr)

	go func() {
		if err := server.ListenAndServe(ctx, fmt.Sprintf(":%d", *port), addressChannel); err != nil && err != http.ErrServerClosed {
			klog.Fatalf("Listen failed: %v", err)
		}
	}()

	address := <-addressChannel
	fmt.Fprintf(os.Stderr, "Listening on %s\n", address)

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, os.Interrupt)

	<-wait

	return nil
}

func createEmptyCommit(repo *gogit.Repository) error {
	store := repo.Storer
	// Create first commit using empty tree.
	emptyTree := object.Tree{}
	encodedTree := store.NewEncodedObject()
	if err := emptyTree.Encode(encodedTree); err != nil {
		return fmt.Errorf("failed to encode initial empty commit tree: %w", err)
	}

	treeHash, err := store.SetEncodedObject(encodedTree)
	if err != nil {
		return fmt.Errorf("failed to create initial empty commit tree: %w", err)
	}

	sig := object.Signature{
		Name:  "Git Server",
		Email: "git-server@kpt.dev",
		When:  time.Now(),
	}

	commit := object.Commit{
		Author:       sig,
		Committer:    sig,
		Message:      "Empty Commit",
		TreeHash:     treeHash,
		ParentHashes: []plumbing.Hash{}, // No parents
	}

	encodedCommit := store.NewEncodedObject()
	if err := commit.Encode(encodedCommit); err != nil {
		return fmt.Errorf("failed to encode initial empty commit: %w", err)
	}

	commitHash, err := store.SetEncodedObject(encodedCommit)
	if err != nil {
		return fmt.Errorf("failed to create initial empty commit: %w", err)
	}

	main := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/main"), commitHash)
	if err := repo.Storer.SetReference(main); err != nil {
		return fmt.Errorf("failed to set refs/heads/main to commit sha %s: %w", commitHash, err)
	}

	head := plumbing.NewSymbolicReference(plumbing.HEAD, "refs/heads/main")
	if err := repo.Storer.SetReference(head); err != nil {
		return fmt.Errorf("failed to set HEAD to refs/heads/main: %w", err)
	}

	return nil
}
