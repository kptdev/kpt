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
	"path"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"k8s.io/klog/v2"
)

// packageList holds a list of packages in the git repository
type packageList struct {
	// parent is the gitRepository of which this is part
	parent *gitRepository

	// commit is the commit at which we scanned for packages
	commit *object.Commit

	// packages holds the packages we found
	packages map[string]*packageListEntry
}

// packageListEntry is a single package found in a git repository
type packageListEntry struct {
	// parent is the packageList of which we are part
	parent *packageList

	// path is the relative path to the root of the package (directory containing the Kptfile)
	path string

	// treeHash is the git-hash of the git tree corresponding to Path
	treeHash plumbing.Hash
}

// buildGitPackageRevision creates a gitPackageRevision for the packageListEntry
// TODO: Can packageListEntry just _be_ a gitPackageRevision?
func (p *packageListEntry) buildGitPackageRevision(ctx context.Context, revision string, workspace v1alpha1.WorkspaceName, ref *plumbing.Reference) (*gitPackageRevision, error) {
	repo := p.parent.parent
	tasks, err := repo.loadTasks(ctx, p.parent.commit, p.path, workspace)
	if err != nil {
		return nil, err
	}

	var updated time.Time
	var updatedBy string

	// For the published packages on a tag or draft and proposed branches we know that the latest commit
	// if specific to the package in question. Thus, we can just take the last commit on the tag/branch.
	// If the ref is nil, we consider the package as being final and on the package branch.
	if ref != nil && (isTagInLocalRepo(ref.Name()) || isDraftBranchNameInLocal(ref.Name()) || isProposedBranchNameInLocal(ref.Name())) {
		updated = p.parent.commit.Author.When
		updatedBy = p.parent.commit.Author.Email
	} else {
		// If we are on the package branch, we can not assume that the last commit
		// pertains to the package in question. So we scan the git history to find
		// the last commit for the package based on the porch commit tags. We don't
		// use the revision here, since we are looking at the package branch while
		// the revisions only helps identify the tags.
		commit, err := repo.findLatestPackageCommit(ctx, p.parent.commit, p.path)
		if err != nil {
			return nil, err
		}
		if commit != nil {
			updated = commit.Author.When
			updatedBy = commit.Author.Email
		}
		// If not commit was found with the porch commit tags, we don't really
		// know who approved the package or when it happend. We could find this
		// by scanning the tree for every commit, but that is a pretty expensive
		// operation.
	}

	// for backwards compatibility with packages that existed before porch supported
	// workspaceNames, we populate the workspaceName as the revision number if it is empty
	if workspace == "" {
		workspace = v1alpha1.WorkspaceName(revision)
	}

	return &gitPackageRevision{
		repo:          repo,
		path:          p.path,
		workspaceName: workspace,
		revision:      revision,
		updated:       updated,
		updatedBy:     updatedBy,
		ref:           ref,
		tree:          p.treeHash,
		commit:        p.parent.commit.Hash,
		tasks:         tasks,
	}, nil
}

// DiscoveryPackagesOptions holds the configuration for walking a git tree
type DiscoverPackagesOptions struct {
	// FilterPrefix restricts package discovery to a particular subdirectory.
	// The subdirectory is not required to exist (we will return an empty list of packages).
	FilterPrefix string

	// Recurse enables recursive traversal of the git tree.
	Recurse bool
}

// discoverPackages is the recursive function we use to traverse the tree and find packages.
// tree is the git-tree we are search, treePath is the repo-relative-path to tree.
func (t *packageList) discoverPackages(tree *object.Tree, treePath string, recurse bool) error {
	for _, e := range tree.Entries {
		if e.Name == "Kptfile" {
			p := path.Join(treePath, e.Name)
			if !e.Mode.IsRegular() {
				klog.Warningf("skipping %q: Kptfile is not a file", p)
				continue
			}

			// Found a package
			t.packages[treePath] = &packageListEntry{
				path:     treePath,
				treeHash: tree.Hash,
				parent:   t,
			}
		}
	}

	if recurse {
		for _, e := range tree.Entries {
			if e.Mode != filemode.Dir {
				continue
			}

			// This is safe because this function is only called holding the mutex in gitRepository
			dirTree, err := t.parent.repo.TreeObject(e.Hash)
			if err != nil {
				return fmt.Errorf("error getting git tree %v: %w", e.Hash, err)
			}

			if err := t.discoverPackages(dirTree, path.Join(treePath, e.Name), recurse); err != nil {
				return err
			}
		}
	}

	return nil
}
