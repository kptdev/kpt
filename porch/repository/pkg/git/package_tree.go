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

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"
)

// packageTree holds a list of packages in the git repository
type packageTree struct {
	// parent is the gitRepository of which this is part
	parent *gitRepository

	// commit is the commit at which we scanned for packages
	commit *object.Commit

	// packages holds the packages we found
	packages map[string]*packageTreeEntry
}

// packageTreeEntry is a single package found in a git repository
type packageTreeEntry struct {
	// parent is the packageTree of which we are part
	parent *packageTree

	// Path is the relative path to the root of the package (directory containing the Kptfile)
	Path string

	// treeHash is the git-hash of the git tree corresponding to Path
	treeHash plumbing.Hash
}

// buildGitPackageRevision creates a gitPackageRevision for the packageTreeEntry
// TODO: Can packageTreeEntry just _be_ a gitPackageRevision?
func (p *packageTreeEntry) buildGitPackageRevision(revision string, ref *plumbing.Reference) (*gitPackageRevision, error) {
	return &gitPackageRevision{
		parent:   p.parent.parent,
		path:     p.Path,
		revision: revision,
		updated:  p.parent.commit.Author.When,
		ref:      ref,
		tree:     p.treeHash,
		commit:   p.parent.commit.Hash,
	}, nil
}

// DiscoverPackagesInTree finds the packages in the git repository, under commit.
// If filterPrefix is non-empty, only packages with the specified prefix will be returned.
// It is not an error if filterPrefix matches no packages or even is not a real directory name;
// we will simply return an empty list of packages.
func (r *gitRepository) DiscoverPackagesInTree(commit *object.Commit, filterPrefix string) (*packageTree, error) {
	t := &packageTree{
		parent:   r,
		commit:   commit,
		packages: make(map[string]*packageTreeEntry),
	}

	rootTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve commit %v to tree (corrupted repository?): %w", commit.Hash, err)
	}

	if filterPrefix != "" {
		tree, err := rootTree.Tree(filterPrefix)
		if err != nil {
			if err == object.ErrDirectoryNotFound {
				// We treat the filter prefix as a filter, the path doesn't have to exist
				klog.Warningf("could not find filterPrefix %q in commit %v; returning no packages", filterPrefix, commit.Hash)
				return t, nil
			} else {
				return nil, fmt.Errorf("error getting tree %s: %w", filterPrefix, err)
			}
		}
		rootTree = tree
	}

	if err := t.discoverPackages(rootTree, filterPrefix); err != nil {
		return nil, err
	}

	klog.V(2).Infof("discovered packages @%v with prefix %q: %#v", commit.Hash, filterPrefix, t.packages)
	return t, nil
}

// discoverPackages is the recursive function we use to traverse the tree and find packages.
// tree is the git-tree we are search, treePath is the repo-relative-path to tree.
func (t *packageTree) discoverPackages(tree *object.Tree, treePath string) error {
	for _, e := range tree.Entries {
		if e.Name == "Kptfile" {
			p := path.Join(treePath, e.Name)
			if !e.Mode.IsRegular() {
				klog.Warningf("skipping %q: Kptfile is not a file", p)
				continue
			}

			// Found a package
			klog.Infof("found package %q with Kptfile hash %q", p, e.Hash)
			t.packages[treePath] = &packageTreeEntry{
				Path:     treePath,
				treeHash: tree.Hash,
				parent:   t,
			}
		}
	}

	for _, e := range tree.Entries {
		if e.Mode != filemode.Dir {
			continue
		}

		dirTree, err := t.parent.repo.TreeObject(e.Hash)
		if err != nil {
			return fmt.Errorf("error getting git tree %v: %w", e.Hash, err)
		}

		if err := t.discoverPackages(dirTree, path.Join(treePath, e.Name)); err != nil {
			return err
		}
	}
	return nil
}
