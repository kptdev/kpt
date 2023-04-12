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
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	porchSignatureName  = "Package Orchestration Service"
	porchSignatureEmail = "porch@kpt.dev"
)

type commitHelper struct {
	repository *gitRepository

	// trees holds a map of all the tree objects we are writing to.
	// We reuse the existing object.Tree structures.
	// When a tree is dirty, we set the hash as plumbing.ZeroHash.
	trees map[string]*object.Tree

	// parentCommitHash holds the hash of the parent commit, or ZeroHash if this is the first commit.
	parentCommitHash plumbing.Hash

	// userInfoProvider provides user information for the commit
	userInfoProvider repository.UserInfoProvider
}

// if packageTree is zero, new tree for the package will be created (effectively replacing the package with the subsequently provided
// contents). If the packageTree is provided, the tree will be used as the initial package contents, possibly subsequently modified.
func newCommitHelper(repo *gitRepository, userInfoProvider repository.UserInfoProvider,
	parentCommitHash plumbing.Hash, packagePath string, packageTree plumbing.Hash) (*commitHelper, error) {
	var root *object.Tree

	if parentCommitHash.IsZero() {
		// No parent commit, start with an empty tree
		root = &object.Tree{}
	} else {
		parentCommit, err := repo.getCommit(parentCommitHash)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve parent commit hash %s to commit: %w", parentCommitHash, err)
		}
		t, err := parentCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("cannot resolve parent commit's (%s) tree (%s) to tree object: %w", parentCommitHash, parentCommit.TreeHash, err)
		}
		root = t
	}

	trees, err := initializeTrees(repo, root, packagePath, packageTree)
	if err != nil {
		return nil, err
	}

	ch := &commitHelper{
		repository:       repo,
		trees:            trees,
		parentCommitHash: parentCommitHash,
		userInfoProvider: userInfoProvider,
	}

	return ch, nil
}

// Initializes ancestor trees of the package, reading them from the storer.
// If packageTree hash is provided, it will be used as the package's initial tree. Otherwise, new tree will be used
// (effectively replacing the package with an empty one).
func initializeTrees(repo *gitRepository, root *object.Tree, packagePath string,
	packageTreeHash plumbing.Hash) (map[string]*object.Tree, error) {

	trees := map[string]*object.Tree{
		"": root,
	}

	parts := strings.Split(packagePath, "/")
	if len(parts) == 0 {
		// empty package path is invalid
		return nil, fmt.Errorf("invalid package path: %q", packagePath)
	}

	// Load all ancestor trees
	parent := root
	for i, max := 0, len(parts)-1; i < max; i++ {
		name := parts[i]
		path := strings.Join(parts[0:i+1], "/")

		var current *object.Tree
		switch existing := findEntry(parent, name); {
		case existing == nil:
			// Create new empty tree for this ancestor.
			current = &object.Tree{}

		case existing.Mode == filemode.Dir:
			// Existing entry is a tree. use it
			hash := existing.Hash
			curr, err := repo.getTree(hash)
			if err != nil {
				return nil, fmt.Errorf("cannot read existing tree %s; root %q, path %q", hash, root.Hash, path)
			}
			current = curr

		default:
			// Existing entry is not a tree. Error.
			return nil, fmt.Errorf("path %q is %s, not a directory in tree %s, root %q", path, existing.Mode, existing.Hash, root.Hash)
		}

		// Set tree in the parent
		setOrAddTreeEntry(parent, object.TreeEntry{
			Name: name,
			Mode: filemode.Dir,
			Hash: plumbing.ZeroHash,
		})

		trees[strings.Join(parts[0:i+1], "/")] = current
		parent = current
	}

	// Initialize the package tree.
	lastPart := parts[len(parts)-1]
	if !packageTreeHash.IsZero() {
		// Initialize with the supplied package tree.
		packageTree, err := repo.getTree(packageTreeHash)
		if err != nil {
			return nil, fmt.Errorf("cannot find existing package tree %s for package %q: %w", packageTreeHash, packagePath, err)
		}
		trees[packagePath] = packageTree
		setOrAddTreeEntry(parent, object.TreeEntry{
			Name: lastPart,
			Mode: filemode.Dir,
			Hash: plumbing.ZeroHash,
		})
	} else {
		// Remove the entry if one exists
		removeTreeEntry(parent, lastPart)
	}

	return trees, nil
}

// Returns a pointer to the entry if found (by name); nil if not found
func findEntry(tree *object.Tree, name string) *object.TreeEntry {
	for i := range tree.Entries {
		e := &tree.Entries[i]
		if e.Name == name {
			return e
		}
	}
	return nil
}

// setOrAddTreeEntry will overwrite the existing entry (by name) or insert if not present.
func setOrAddTreeEntry(tree *object.Tree, entry object.TreeEntry) {
	for i := range tree.Entries {
		e := &tree.Entries[i]
		if e.Name == entry.Name {
			*e = entry // Overwrite the tree entry
			return
		}
	}
	// Not found. append new
	tree.Entries = append(tree.Entries, entry)
}

// removeTreeEntry will remove the specified entry (by name)
func removeTreeEntry(tree *object.Tree, name string) {
	entries := tree.Entries
	for i := range entries {
		e := &entries[i]
		if e.Name == name {
			tree.Entries = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

// storeFile writes a blob with contents at the specified path
func (h *commitHelper) storeFile(path, contents string) error {
	hash, err := h.repository.storeBlob(contents)
	if err != nil {
		return err
	}

	if err := h.storeBlobHashInTrees(path, hash); err != nil {
		return err
	}
	return nil
}

// storeTree sets the tree of the provided path to the tree
// referenced by the provided hash.
func (h *commitHelper) storeTree(path string, hash plumbing.Hash) error {
	parentPath, pkg := split(path)
	tree := h.ensureTree(parentPath)
	setOrAddTreeEntry(tree, object.TreeEntry{
		Name: pkg,
		Mode: filemode.Dir,
	})
	pTree, err := h.repository.getTree(hash)
	if err != nil {
		return err
	}
	h.trees[path] = pTree
	return nil
}

// readFile returns the contents of the blob at path.
// If the file is not found it returns an error satisfying os.IsNotExist
func (h *commitHelper) readFile(path string) ([]byte, error) {
	dir, filename := split(path)
	tree := h.trees[dir]
	if tree == nil {
		return nil, fs.ErrNotExist
	}

	entry := findEntry(tree, filename)
	if entry == nil {
		return nil, fs.ErrNotExist
	}

	blob, err := h.repository.blobObject(entry.Hash)
	if err != nil {
		// This is an internal consistency error, so we don't return ErrNotExist
		return nil, fmt.Errorf("error reading from git: %w", err)
	}
	r, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("error reading from git: %w", err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading from git: %w", err)
	}
	return b, nil
}

// commit stores all changes in git and creates a commit object.
func (h *commitHelper) commit(ctx context.Context, message string, pkgPath string, additionalParentCommits ...plumbing.Hash) (commit, pkgTree plumbing.Hash, err error) {
	rootTreeHash, err := h.storeTrees("")
	if err != nil {
		return plumbing.ZeroHash, plumbing.ZeroHash, err
	}

	var ui *repository.UserInfo
	if h.userInfoProvider != nil {
		ui = h.userInfoProvider.GetUserInfo(ctx)
	}

	var parentCommits []plumbing.Hash
	if !h.parentCommitHash.IsZero() {
		parentCommits = append(parentCommits, h.parentCommitHash)
	}
	parentCommits = append(parentCommits, additionalParentCommits...)

	commit, err = h.storeCommit(parentCommits, rootTreeHash, ui, message)
	if err != nil {
		return plumbing.ZeroHash, plumbing.ZeroHash, err
	}
	// Update the parentCommitHash so the correct parent will be used for the
	// next commit.
	h.parentCommitHash = commit

	if pkg, ok := h.trees[pkgPath]; ok {
		pkgTree = pkg.Hash
	} else {
		pkgTree = plumbing.ZeroHash
	}

	return commit, pkgTree, nil
}

// split returns the full directory path and file name
// If there is no directory, it returns an empty directory path and the path as the filename.
func split(path string) (string, string) {
	i := strings.LastIndex(path, "/")
	if i >= 0 {
		return path[:i], path[i+1:]
	}
	return "", path
}

// ensureTrees ensures we have a trees for all directories in fullPath.
// fullPath is expected to be a directory path.
func (h *commitHelper) ensureTree(fullPath string) *object.Tree {
	if tree, ok := h.trees[fullPath]; ok {
		return tree
	}

	dir, base := split(fullPath)
	parent := h.ensureTree(dir)

	te := object.TreeEntry{
		Name: base,
		Mode: filemode.Dir,
	}

	for ei, ev := range parent.Entries {
		// Replace whole subtrees modified by the package contents.
		if ev.Name == te.Name && !ev.Hash.IsZero() {
			parent.Entries[ei] = te
			goto added
		}
	}
	// Append a new entry
	parent.Entries = append(parent.Entries, te)

added:
	tree := &object.Tree{}
	h.trees[fullPath] = tree
	return tree
}

// storeBlobHashInTrees writes the (previously stored) blob hash at fullpath, marking all the directory trees as dirty.
func (h *commitHelper) storeBlobHashInTrees(fullPath string, hash plumbing.Hash) error {
	dir, file := split(fullPath)
	if file == "" {
		return fmt.Errorf("invalid resource path: %q; no file name", fullPath)
	}

	tree := h.ensureTree(dir)
	setOrAddTreeEntry(tree, object.TreeEntry{
		Name: file,
		Mode: filemode.Regular,
		Hash: hash,
	})

	return nil
}

// storeTrees writes the tree at treePath to git, first writing all child trees.
func (h *commitHelper) storeTrees(treePath string) (plumbing.Hash, error) {
	tree, ok := h.trees[treePath]
	if !ok {
		return plumbing.Hash{}, fmt.Errorf("failed to find a tree %q", treePath)
	}

	entries := tree.Entries
	sort.Slice(entries, func(i, j int) bool {
		return entrySortKey(&entries[i]) < entrySortKey(&entries[j])
	})

	// Store all child trees and get their hashes
	for i := range entries {
		e := &entries[i]
		if e.Mode != filemode.Dir {
			continue
		}
		if !e.Hash.IsZero() {
			continue
		}

		hash, err := h.storeTrees(path.Join(treePath, e.Name))
		if err != nil {
			return plumbing.Hash{}, err
		}
		e.Hash = hash
	}

	treeHash, err := h.repository.storeTree(tree)
	if err != nil {
		return plumbing.Hash{}, err
	}

	tree.Hash = treeHash
	return treeHash, nil
}

// Git sorts tree entries as though directories have '/' appended to them.
func entrySortKey(e *object.TreeEntry) string {
	if e.Mode == filemode.Dir {
		return e.Name + "/"
	}
	return e.Name
}

// storeCommit creates and writes a commit object to git.
func (h *commitHelper) storeCommit(parentCommits []plumbing.Hash, tree plumbing.Hash, userInfo *repository.UserInfo, message string) (plumbing.Hash, error) {
	now := time.Now()
	var authorName, authorEmail string
	if userInfo != nil {
		// Authenticated user info only provides one value...
		authorName = userInfo.Name
		authorEmail = userInfo.Email
	} else {
		// Defaults
		authorName = porchSignatureName
		authorEmail = porchSignatureEmail
	}
	commit := &object.Commit{
		Author: object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  now,
		},
		Committer: object.Signature{
			Name:  porchSignatureName,
			Email: porchSignatureEmail,
			When:  now,
		},
		Message:  message,
		TreeHash: tree,
	}

	if len(parentCommits) > 0 {
		commit.ParentHashes = parentCommits
	}

	return h.repository.storeCommit(commit)
}
