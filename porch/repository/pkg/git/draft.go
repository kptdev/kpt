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
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"k8s.io/klog/v2"
)

type gitPackageDraft struct {
	gitPackageRevision
}

var _ repository.PackageDraft = &gitPackageDraft{}

func (d *gitPackageDraft) UpdateResources(ctx context.Context, new *v1alpha1.PackageRevisionResources, change *v1alpha1.Task) error {
	parent, err := d.parent.repo.CommitObject(d.draft.Hash())
	if err != nil {
		return fmt.Errorf("cannot resolve parent commit hash to commit: %w", err)
	}
	root, err := parent.Tree()
	if err != nil {
		return fmt.Errorf("cannot resolve parent commit to root tree: %w", err)
	}

	dirs := map[string]*object.Tree{
		"": {
			// Root tree; Copy over all entries
			// TODO: Verify that on creation (first commit) the package directory doesn't exist.
			// TODO: Verify that on subsequent commits, only the package's directory is being modified.
			Entries: root.Entries,
		},
	}
	for k, v := range new.Spec.Resources {
		hash, err := storeBlob(d.parent.repo.Storer, v)
		if err != nil {
			return err
		}

		// TODO: decide whether paths should include package directory or not.
		p := path.Join(d.path, k)
		if err := storeBlobHashInTrees(dirs, p, hash); err != nil {
			return err
		}
	}

	treeHash, err := storeTrees(d.parent.repo.Storer, dirs, "")
	if err != nil {
		return err
	}

	commit, err := storeCommit(d.parent.repo.Storer, d.draft.Hash(), treeHash, change)
	if err != nil {
		return err
	}

	head := plumbing.NewHashReference(d.draft.Name(), commit)
	if err := d.parent.repo.Storer.SetReference(head); err != nil {
		return err
	}
	d.draft = head
	return nil
}

// Finish round of updates.
func (d *gitPackageDraft) Close(ctx context.Context) (repository.PackageRevision, error) {
	// TODO: This removal of drafts is a hack
	refSpec := config.RefSpec(fmt.Sprintf("%s:%s", d.draft.Name(), strings.ReplaceAll(d.draft.Name().String(), "/drafts/", "/")))
	klog.Infof("pushing refspec %v", refSpec)

	if err := d.parent.repo.Push(&git.PushOptions{
		RemoteName:        "origin",
		RefSpecs:          []config.RefSpec{refSpec},
		Auth:              d.parent.auth,
		RequireRemoteRefs: []config.RefSpec{},
	}); err != nil {
		return nil, fmt.Errorf("failed to push to git: %w", err)
	}

	// TODO: return Revision only.
	return d, nil
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
		if entries[i].Mode == entries[j].Mode {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Mode == filemode.Dir // Directories before files
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
	return store.SetEncodedObject(eo)
}

func storeCommit(store storer.EncodedObjectStorer, parent plumbing.Hash, tree plumbing.Hash, change *v1alpha1.Task) (plumbing.Hash, error) {
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
		Message:  fmt.Sprintf("Intermittent commit: %s", change.Type),
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
