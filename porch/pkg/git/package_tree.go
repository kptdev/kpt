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

// FindPackage finds the packages in the git repository, under commit, if it is exists at path.
// If no package is found at that path, returns nil, nil
func (r *gitRepository) FindPackage(commit *object.Commit, packagePath string) (*packageListEntry, error) {
	t, err := r.DiscoverPackagesInTree(commit, DiscoverPackagesOptions{FilterPrefix: packagePath, Recurse: false})
	if err != nil {
		return nil, err
	}
	return t.packages[packagePath], nil
}

// DiscoverPackagesInTree finds the packages in the git repository, under commit.
// If filterPrefix is non-empty, only packages with the specified prefix will be returned.
// It is not an error if filterPrefix matches no packages or even is not a real directory name;
// we will simply return an empty list of packages.
func (r *gitRepository) DiscoverPackagesInTree(commit *object.Commit, opt DiscoverPackagesOptions) (*packageList, error) {
	t := &packageList{
		parent:   r,
		commit:   commit,
		packages: make(map[string]*packageListEntry),
	}

	rootTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve commit %v to tree (corrupted repository?): %w", commit.Hash, err)
	}

	if opt.FilterPrefix != "" {
		tree, err := rootTree.Tree(opt.FilterPrefix)
		if err != nil {
			if err == object.ErrDirectoryNotFound {
				// We treat the filter prefix as a filter, the path doesn't have to exist
				klog.Warningf("could not find filterPrefix %q in commit %v; returning no packages", opt.FilterPrefix, commit.Hash)
				return t, nil
			} else {
				return nil, fmt.Errorf("error getting tree %s: %w", opt.FilterPrefix, err)
			}
		}
		rootTree = tree
	}

	if err := t.discoverPackages(rootTree, opt.FilterPrefix, opt.Recurse); err != nil {
		return nil, err
	}

	klog.V(2).Infof("discovered packages @%v with prefix %q: %#v", commit.Hash, opt.FilterPrefix, t.packages)
	return t, nil
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
			klog.Infof("found package %q with Kptfile hash %q", p, e.Hash)
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
