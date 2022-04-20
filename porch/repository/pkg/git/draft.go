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
	"os"
	"path"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type gitPackageDraft struct {
	parent    *gitRepository
	path      string
	revision  string
	lifecycle v1alpha1.PackageRevisionLifecycle // New value of the package revision lifecycle
	updated   time.Time
	base      *plumbing.Reference // ref to the base of the package update commit chain (used for conditional push)
	branch    BranchName          // name of the branch where the changes will be pushed
	commit    plumbing.Hash       // Current HEAD of the package changes (commit sha)
	tree      plumbing.Hash       // Cached tree of the package itself, some descendent of commit.Tree()
}

var _ repository.PackageDraft = &gitPackageDraft{}

func (d *gitPackageDraft) UpdateResources(ctx context.Context, new *v1alpha1.PackageRevisionResources, change *v1alpha1.Task) error {
	ch, err := newCommitHelper(d.parent.repo, d.parent.userInfoProvider, d.commit, d.path, plumbing.ZeroHash)
	if err != nil {
		return fmt.Errorf("failed to commit packgae: %w", err)
	}

	for k, v := range new.Spec.Resources {
		ch.storeFile(path.Join(d.path, k), v)
	}

	// Because we can't read the package back without a Kptfile, make sure one is present
	{
		p := path.Join(d.path, "Kptfile")
		_, err := ch.readFile(p)
		if os.IsNotExist(err) {
			// We could write the file here; currently we return an error
			return fmt.Errorf("package must contain Kptfile at root")
		}
	}

	message := fmt.Sprintf("Intermediate commit: %s", change.Type)
	commitHash, packageTree, err := ch.commit(ctx, message, d.path)
	if err != nil {
		return fmt.Errorf("failed to commit package: %w", err)
	}

	d.tree = packageTree
	d.commit = commitHash
	return nil
}

func (d *gitPackageDraft) UpdateLifecycle(ctx context.Context, new v1alpha1.PackageRevisionLifecycle) error {
	d.lifecycle = new
	return nil
}

// Finish round of updates.
func (d *gitPackageDraft) Close(ctx context.Context) (repository.PackageRevision, error) {
	return d.parent.closeDraft(ctx, d)
}

func (r *gitRepository) closeDraft(ctx context.Context, d *gitPackageDraft) (*gitPackageRevision, error) {
	refSpecs := newPushRefSpecBuilder()
	draftBranch := createDraftName(d.path, d.revision)
	proposedBranch := createProposedName(d.path, d.revision)

	var newRef *plumbing.Reference

	switch d.lifecycle {
	case v1alpha1.PackageRevisionLifecyclePublished:
		// Finalize the package revision. Commit it to main branch.
		commitHash, newTreeHash, commitBase, err := r.commitPackageToMain(ctx, d)
		if err != nil {
			return nil, err
		}

		tag := createFinalTagNameInLocal(d.path, d.revision)
		refSpecs.AddRefToPush(commitHash, r.branch.RefInLocal()) // Push new main branch
		refSpecs.AddRefToPush(commitHash, tag)                   // Push the tag
		refSpecs.RequireRef(commitBase)                          // Make sure main didn't advance

		// Delete base branch (if one exists and should be deleted)
		switch base := d.base; {
		case base == nil: // no branch to delete
		case base.Name() == draftBranch.RefInLocal(), base.Name() == proposedBranch.RefInLocal():
			refSpecs.AddRefToDelete(base)
		}

		// Update package draft
		d.commit = commitHash
		d.tree = newTreeHash
		newRef = plumbing.NewHashReference(tag, commitHash)

	case v1alpha1.PackageRevisionLifecycleProposed:
		// Push the package revision into a proposed branch.
		refSpecs.AddRefToPush(d.commit, proposedBranch.RefInLocal())

		// Delete base branch (if one exists and should be deleted)
		switch base := d.base; {
		case base == nil: // no branch to delete
		case base.Name() != proposedBranch.RefInLocal():
			refSpecs.AddRefToDelete(base)
		}

		// Update package referemce (commit and tree hash stay the same)
		newRef = plumbing.NewHashReference(proposedBranch.RefInLocal(), d.commit)

	case v1alpha1.PackageRevisionLifecycleDraft:
		// Push the package revision into a draft branch.
		refSpecs.AddRefToPush(d.commit, draftBranch.RefInLocal())
		// Delete base branch (if one exists and should be deleted)
		switch base := d.base; {
		case base == nil: // no branch to delete
		case base.Name() != draftBranch.RefInLocal():
			refSpecs.AddRefToDelete(base)
		}

		// Update package referemce (commit and tree hash stay the same)
		newRef = plumbing.NewHashReference(draftBranch.RefInLocal(), d.commit)

	default:
		return nil, fmt.Errorf("package has unrecognized lifecycle: %q", d.lifecycle)
	}

	if err := d.parent.pushAndCleanup(ctx, refSpecs); err != nil {
		return nil, err
	}

	return &gitPackageRevision{
		parent:   d.parent,
		path:     d.path,
		revision: d.revision,
		updated:  d.updated,
		ref:      newRef,
		tree:     d.tree,
		commit:   newRef.Hash(),
	}, nil
}

func (r *gitRepository) commitPackageToMain(ctx context.Context, d *gitPackageDraft) (commitHash, newPackageTreeHash plumbing.Hash, base *plumbing.Reference, err error) {
	branch := r.branch
	localRef := branch.RefInLocal()

	var zero plumbing.Hash
	auth, err := r.getAuthMethod(ctx)

	if err != nil {
		return zero, zero, nil, fmt.Errorf("failed to obtain git credentials: %w", err)
	}

	repo := r.repo

	// Fetch main
	switch err := repo.Fetch(&git.FetchOptions{
		RemoteName: OriginName,
		RefSpecs:   []config.RefSpec{branch.ForceFetchSpec()},
		Auth:       auth,
	}); err {
	case nil, git.NoErrAlreadyUpToDate:
		// ok
	default:
		return zero, zero, nil, fmt.Errorf("failed to fetch remote repository: %w", err)
	}

	// Find localTarget branch
	localTarget, err := repo.Reference(localRef, false)
	if err != nil {
		// TODO: handle empty repositories - NotFound error
		return zero, zero, nil, fmt.Errorf("failed to find 'main' branch: %w", err)
	}
	headCommit, err := repo.CommitObject(localTarget.Hash())
	if err != nil {
		return zero, zero, nil, fmt.Errorf("failed to resolve main branch to commit: %w", err)
	}
	packagePath := d.path
	packageTree := d.tree

	// TODO: Check for out-of-band update of the package in main branch
	// (compare package tree in target branch and common base)
	ch, err := newCommitHelper(repo, r.userInfoProvider, headCommit.Hash, packagePath, packageTree)
	if err != nil {
		return zero, zero, nil, fmt.Errorf("failed to initialize commit of package %s to %s", packagePath, localRef)
	}
	message := fmt.Sprintf("Approve %s", packagePath)
	commitHash, newPackageTreeHash, err = ch.commit(ctx, message, packagePath)
	if err != nil {
		return zero, zero, nil, fmt.Errorf("failed to commit package %s to %s", packagePath, localRef)
	}

	return commitHash, newPackageTreeHash, localTarget, nil
}
