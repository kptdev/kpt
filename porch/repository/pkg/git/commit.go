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
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
)

type commitHelper struct {
	storer storage.Storer
	trees  map[string]*object.Tree
	parent plumbing.Hash
}

// if packageTree is zero, new tree for the package will be created (effectively replacing the package with the subsequently provided
// contents). If the packageTree is provided, the tree will be used as the initial package contents, possibly subsequently modified.
func newCommitHelper(storer storage.Storer, parent plumbing.Hash, packagePath string, packageTree plumbing.Hash) (*commitHelper, error) {
	var root *object.Tree

	if parent.IsZero() {
		// No parent commit, start with an empty tree
		root = &object.Tree{}
	} else {
		parentCommit, err := object.GetCommit(storer, parent)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve parent commit hash %s to commit: %w", parent, err)
		}
		t, err := parentCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("cannot resolve parent commit's (%s) tree (%s) to tree object: %w", parent, parentCommit.TreeHash, err)
		}
		root = t
	}

	trees, err := initializeTrees(storer, root, packagePath, packageTree)
	if err != nil {
		return nil, err
	}

	ch := &commitHelper{
		storer: storer,
		trees:  trees,
		parent: parent,
	}

	return ch, nil
}

// Initializes ancestor trees of the package, reading them from the storer.
// If packageTree hash is provided, it will be used as the package's initial tree. Otherwise, new tree will be used
// (effectively replacing the package with an empty one).
func initializeTrees(storer storage.Storer, root *object.Tree, packagePath string,
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
			curr, err := object.GetTree(storer, hash)
			if err != nil {
				return nil, fmt.Errorf("cannot read existing tree %s; root %q, path %q", hash, root.Hash, path)
			}
			current = curr

		default:
			// Existing entry is not a tree. Error.
			return nil, fmt.Errorf("path %q is %s, not a directory in tree %s, root %q", path, existing.Mode, existing.Hash, root.Hash)
		}

		// Set tree in the parent
		setOrAddDirEntry(parent, name)

		trees[strings.Join(parts[0:i+1], "/")] = current
		parent = current
	}

	// Initialize the package tree.
	lastPart := parts[len(parts)-1]
	if !packageTreeHash.IsZero() {
		// Initialize with the supplied package tree.
		packageTree, err := object.GetTree(storer, packageTreeHash)
		if err != nil {
			return nil, fmt.Errorf("cannot find existing package tree %s for package %q", packageTreeHash, packagePath)
		}
		trees[packagePath] = packageTree
		setOrAddDirEntry(parent, lastPart)
	} else {
		// Remove the entry if one exists
		removeDirEntry(parent, lastPart)
	}

	return trees, nil
}

// Returns index of the entry if found (by name); -1 if not found
func findEntry(tree *object.Tree, name string) *object.TreeEntry {
	for i := range tree.Entries {
		e := &tree.Entries[i]
		if e.Name == name {
			return e
		}
	}
	return nil
}

func setOrAddDirEntry(tree *object.Tree, name string) {
	te := object.TreeEntry{
		Name: name,
		Mode: filemode.Dir,
		Hash: [20]byte{},
	}
	for i := range tree.Entries {
		e := &tree.Entries[i]
		if e.Name == name {
			if e.Hash.IsZero() {
				// Tree entry already exists and is new (zero hash); noop
				return
			}
			*e = te // Overwrite the tree entry with a new (zero hash) placeholder.
			return
		}
	}
	// Not found. append new
	tree.Entries = append(tree.Entries, te)
}

func removeDirEntry(tree *object.Tree, name string) {
	entries := tree.Entries
	for i := range entries {
		e := &entries[i]
		if e.Name == name {
			tree.Entries = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

func (h *commitHelper) storeFile(path, contents string) error {
	hash, err := storeBlob(h.storer, contents)
	if err != nil {
		return err
	}

	if err := storeBlobHashInTrees(h.trees, path, hash); err != nil {
		return err
	}
	return nil
}

func (h *commitHelper) commit(message string, pkgPath string) (commit, pkgTree plumbing.Hash, err error) {
	treeHash, err := storeTrees(h.storer, h.trees, "")
	if err != nil {
		return plumbing.ZeroHash, plumbing.ZeroHash, err
	}

	commit, err = storeCommit(h.storer, h.parent, treeHash, message)
	if err != nil {
		return plumbing.ZeroHash, plumbing.ZeroHash, err
	}

	if pkg, ok := h.trees[pkgPath]; ok {
		pkgTree = pkg.Hash
	} else {
		pkgTree = plumbing.ZeroHash
	}

	return commit, pkgTree, nil
}

func storeBlob(store storer.EncodedObjectStorer, value string) (plumbing.Hash, error) {
	data := []byte(value)
	eo := store.NewEncodedObject()
	eo.SetType(plumbing.BlobObject)
	eo.SetSize(int64(len(data)))

	w, err := eo.Writer()
	if err != nil {
		return plumbing.Hash{}, err
	}

	_, err = w.Write(data)
	w.Close()
	if err != nil {
		return plumbing.Hash{}, err
	}
	return store.SetEncodedObject(eo)
}

func split(path string) (string, string) {
	i := strings.LastIndex(path, "/")
	if i >= 0 {
		return path[:i], path[i+1:]
	}
	return "", path
}

func ensureTree(trees map[string]*object.Tree, fullPath string) *object.Tree {
	if tree, ok := trees[fullPath]; ok {
		return tree
	}

	dir, base := split(fullPath)
	parent := ensureTree(trees, dir)

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
	trees[fullPath] = tree
	return tree
}

func storeBlobHashInTrees(trees map[string]*object.Tree, fullPath string, hash plumbing.Hash) error {
	dir, file := split(fullPath)

	if file == "" {
		return fmt.Errorf("invalid resource path: %q; no file name", fullPath)
	}

	tree := ensureTree(trees, dir)
	tree.Entries = append(tree.Entries, object.TreeEntry{
		Name: file,
		Mode: filemode.Regular,
		Hash: hash,
	})

	return nil
}

func storeTrees(store storer.EncodedObjectStorer, trees map[string]*object.Tree, treePath string) (plumbing.Hash, error) {
	tree, ok := trees[treePath]
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

		hash, err := storeTrees(store, trees, path.Join(treePath, e.Name))
		if err != nil {
			return plumbing.Hash{}, err
		}
		e.Hash = hash
	}

	eo := store.NewEncodedObject()
	if err := tree.Encode(eo); err != nil {
		return plumbing.Hash{}, err
	}

	treeHash, err := store.SetEncodedObject(eo)
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

func storeCommit(store storer.EncodedObjectStorer, parent plumbing.Hash, tree plumbing.Hash, message string) (plumbing.Hash, error) {
	now := time.Now()
	commit := &object.Commit{
		Author: object.Signature{
			Name:  "Porch Author",
			Email: "author@kpt.dev",
			When:  now,
		},
		Committer: object.Signature{
			Name:  "Porch Committer",
			Email: "committer@kpt.dev",
			When:  now,
		},
		Message:  message,
		TreeHash: tree,
	}

	if !parent.IsZero() {
		commit.ParentHashes = []plumbing.Hash{parent}
	}

	eo := store.NewEncodedObject()
	if err := commit.Encode(eo); err != nil {
		return plumbing.Hash{}, err
	}
	return store.SetEncodedObject(eo)
}
