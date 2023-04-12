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
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repos interface {
	FindRepo(ctx context.Context, id string) (*Repo, error)
}

// StaticRepos holds multiple registered git repositories
type StaticRepos struct {
	mutex sync.Mutex
	repos map[string]*Repo
}

// NewRepos constructs an instance of Repos
func NewStaticRepos() *StaticRepos {
	return &StaticRepos{
		repos: make(map[string]*Repo),
	}
}

// FindRepo returns a repo registered under the specified id, or nil if none is registered.
func (r *StaticRepos) FindRepo(ctx context.Context, id string) (*Repo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.repos[id], nil
}

// Add registers a git repository under the specified id
func (r *StaticRepos) Add(id string, repo *Repo) error {
	if !isRepoIDAllowed(id) {
		return fmt.Errorf("invalid name %q", id)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, found := r.repos[id]; found {
		return fmt.Errorf("repo %q already exists", id)
	}
	r.repos[id] = repo
	return nil
}

func NewDynamicRepos(baseDir string, gitRepoOptions []GitRepoOption) *DynamicRepos {
	return &DynamicRepos{
		baseDir:        baseDir,
		repos:          make(map[string]*dynamicRepo),
		gitRepoOptions: gitRepoOptions,
	}
}

type DynamicRepos struct {
	mutex          sync.Mutex
	repos          map[string]*dynamicRepo
	baseDir        string
	gitRepoOptions []GitRepoOption
}

type dynamicRepo struct {
	mutex          sync.Mutex
	repo           *Repo
	dir            string
	gitRepoOptions []GitRepoOption
}

func isRepoIDAllowed(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			// OK
		} else if r >= '0' && r <= '9' {
			// OK
		} else {
			switch r {
			case '-':
				// OK
			default:
				return false
			}
		}
	}
	return true
}

func (r *DynamicRepos) FindRepo(ctx context.Context, id string) (*Repo, error) {
	dir := filepath.Join(r.baseDir, id)
	if !isRepoIDAllowed(id) {
		return nil, fmt.Errorf("invalid name %q", id)
	}

	r.mutex.Lock()
	repo := r.repos[id]
	if repo == nil {
		repo = &dynamicRepo{
			dir:            dir,
			gitRepoOptions: r.gitRepoOptions,
		}
		r.repos[id] = repo
	}
	r.mutex.Unlock()

	return repo.open()
}

func (r *dynamicRepo) open() (*Repo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.repo == nil {
		var gogitRepo *gogit.Repository
		var err error

		if gogitRepo, err = gogit.PlainOpen(r.dir); err != nil {
			if err != gogit.ErrRepositoryNotExists {
				return nil, fmt.Errorf("failed to open git repository %q: %w", r.dir, err)
			}
			isBare := true
			gogitRepo, err = gogit.PlainInit(r.dir, isBare)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize git repository %q: %w", r.dir, err)
			}
			if err := CreateEmptyCommit(gogitRepo); err != nil {
				return nil, err
			}

			// Delete go-git default branch
			_ = gogitRepo.Storer.RemoveReference(plumbing.Master)
		}
		repo, err := NewRepo(gogitRepo, r.gitRepoOptions...)
		if err != nil {
			return nil, err
		}
		r.repo = repo
	}

	return r.repo, nil
}

func CreateEmptyCommit(repo *gogit.Repository) error {
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
