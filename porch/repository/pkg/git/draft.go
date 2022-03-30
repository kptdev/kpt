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
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
)

type gitPackageDraft struct {
	parent    *gitRepository
	path      string
	revision  string
	lifecycle v1alpha1.PackageRevisionLifecycle // New value of the package revision lifecycle
	updated   time.Time
	ref       *plumbing.Reference // ref is the Git reference at which the package exists
	tree      plumbing.Hash       // tree of the package itself, some descendent of commit.Tree()
	commit    plumbing.Hash       // Current version of the package (commit sha)
}

var _ repository.PackageDraft = &gitPackageDraft{}

func (d *gitPackageDraft) UpdateResources(ctx context.Context, new *v1alpha1.PackageRevisionResources, change *v1alpha1.Task) error {
	ch, err := newCommitHelper(d.parent.repo.Storer, d.ref.Hash(), d.path, plumbing.ZeroHash)
	if err != nil {
		return fmt.Errorf("failed to commit packgae: %w", err)
	}

	for k, v := range new.Spec.Resources {
		ch.storeFile(path.Join(d.path, k), v)
	}

	message := fmt.Sprintf("Intermittent commit: %s", change.Type)
	commitHash, packageTree, err := ch.commit(message, d.path)
	if err != nil {
		return fmt.Errorf("failed to commit package: %w", err)
	}

	head := plumbing.NewHashReference(d.ref.Name(), commitHash)
	if err := d.parent.repo.Storer.SetReference(head); err != nil {
		return err
	}

	d.tree = packageTree
	d.ref = head
	d.commit = commitHash
	return nil
}

func (d *gitPackageDraft) UpdateLifecycle(ctx context.Context, new v1alpha1.PackageRevisionLifecycle) error {
	d.lifecycle = new
	return nil
}

// Finish round of updates.
func (d *gitPackageDraft) Close(ctx context.Context) (repository.PackageRevision, error) {
	remoteRef := refMain // TODO: support  branches registered with repository
	// RefSpec(s) to push to the origin
	pushRefSpecs := []config.RefSpec{}
	// References to clean up (delete) locally and remotely after successful push
	cleanup := map[plumbing.ReferenceName]bool{}
	refNames := createPackageRefNames(d.path, d.revision)
	repo := d.parent.repo

	switch d.lifecycle {
	case v1alpha1.PackageRevisionLifecycleFinal:
		// Finalize the package revision. Commit it to main branch.
		commitHash, newTreeHash, err := d.commitPackageToMain(ctx)
		if err != nil {
			return nil, err
		}

		// Create Tag
		switch _, err := repo.Storer.Reference(refNames.final); err {
		case nil:
			// Tag exists, cannot overwrite
			return nil, fmt.Errorf("another instance of finalized package %s:%s:%s already exists", d.parent.name, d.path, d.revision)
		case plumbing.ErrReferenceNotFound:
			// Tag does not yet exist
		}

		newRef, err := setReference(repo.Storer, refNames.final, commitHash)
		if err != nil {
			return nil, fmt.Errorf("failed to finalize package: %v", err)
		}

		pushRefSpecs = append(
			pushRefSpecs,
			config.RefSpec(fmt.Sprintf("%s:%s", commitHash, remoteRef)),      // Push new main
			config.RefSpec(fmt.Sprintf("%s:%s", commitHash, refNames.final)), // Push the tag
		)

		if d.ref != nil {
			cleanup[d.ref.Name()] = true
		}
		// also make sure to clean up draft and proposed refs for this package, if they exist.
		cleanup[refNames.draft] = true
		cleanup[refNames.proposed] = true

		// Update package draft
		d.commit = commitHash
		d.tree = newTreeHash
		d.ref = newRef

	case v1alpha1.PackageRevisionLifecycleProposed:
		newRef, err := setReference(repo.Storer, refNames.proposed, d.commit)
		if err != nil {
			return nil, fmt.Errorf("failed to update package lifecycle to \"proposed\": %v", err)
		}
		// Push the package revision into a proposed branch.
		pushRefSpecs = append(pushRefSpecs, config.RefSpec(fmt.Sprintf("%s:%s", d.commit, refNames.proposed)))
		cleanup[refNames.draft] = true

		// Update package referemce (commit and tree hash stay the same)
		d.ref = newRef

	case v1alpha1.PackageRevisionLifecycleDraft:
		newRef, err := setReference(repo.Storer, refNames.draft, d.commit)
		if err != nil {
			return nil, fmt.Errorf("failed to update package draft: %v", err)
		}
		// Push the package revision into a draft branch.
		// TODO: implement config resolution rather than forcing the push (+)
		pushRefSpecs = append(pushRefSpecs, config.RefSpec(fmt.Sprintf("+%s:%s", d.commit, refNames.draft)))
		cleanup[refNames.proposed] = true // In case client downgraded packaget to draft from proposed

		// Update package referemce (commit and tree hash stay the same)
		d.ref = newRef

	default:
		return nil, fmt.Errorf("package has unrecognized lifecycle: %q", d.lifecycle)
	}

	if err := d.parent.pushAndCleanupRefs(ctx, pushRefSpecs, cleanup); err != nil {
		return nil, err
	}

	return &gitPackageRevision{
		parent:   d.parent,
		path:     d.path,
		revision: d.revision,
		updated:  d.updated,
		ref:      d.ref,
		tree:     d.tree,
		commit:   d.ref.Hash(),
	}, nil
}

func (d *gitPackageDraft) commitPackageToMain(ctx context.Context) (commitHash, newPackageTreeHash plumbing.Hash, err error) {
	localRef := refMain // TODO: add support for the branch provided at repository registration
	remoteRef := refMain

	var zero plumbing.Hash
	auth, err := d.parent.getAuthMethod(ctx)
	if err != nil {
		return zero, zero, fmt.Errorf("failed to obtain git credentials: %w", err)
	}

	repo := d.parent.repo

	// Fetch main
	switch err := repo.Fetch(&git.FetchOptions{
		RemoteName: originName,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("+%s:%s", localRef, remoteRef))},
		Auth:       auth,
		Tags:       git.AllTags,
	}); err {
	case nil, git.NoErrAlreadyUpToDate:
		// ok
	default:
		return zero, zero, fmt.Errorf("failed to fetch remote repository: %w", err)
	}

	// Find localTarget branch
	localTarget, err := repo.Reference(localRef, true)
	if err != nil {
		// TODO: handle empty repositories - NotFound error
		return zero, zero, fmt.Errorf("failed to find 'main' branch: %w", err)
	}
	headCommit, err := repo.CommitObject(localTarget.Hash())
	if err != nil {
		return zero, zero, fmt.Errorf("failed to resolve main branch to commit: %w", err)
	}
	headRoot, err := headCommit.Tree()
	if err != nil {
		// TODO: handle empty repositories
		return zero, zero, fmt.Errorf("failed to get main commit tree; %w", err)
	}
	packagePath := d.path
	packageTree := d.tree
	if packageEntry, err := headRoot.FindEntry(packagePath); err == nil {
		if packageEntry.Hash != packageTree {
			return zero, zero, fmt.Errorf("internal error: package tree consistency check failed: %s != %s", packageEntry.Hash, packageTree)
		}
	}

	// TODO: Check for out-of-band update of the package in main branch
	// (compare package tree in target branch and common base)
	ch, err := newCommitHelper(repo.Storer, headCommit.Hash, packagePath, packageTree)
	if err != nil {
		return zero, zero, fmt.Errorf("failed to initialize commit of package %s to %s", packagePath, localRef)
	}
	message := fmt.Sprintf("Approve %s", packagePath)
	commitHash, newPackageTreeHash, err = ch.commit(message, packagePath)
	if err != nil {
		return zero, zero, fmt.Errorf("failed to commit package %s to %s", packagePath, localRef)
	}

	return commitHash, newPackageTreeHash, nil
}

type packageRefs struct {
	draft, proposed, final plumbing.ReferenceName
}

func createPackageRefNames(name, revision string) packageRefs {
	return packageRefs{
		draft:    createDraftRefName(name, revision),
		proposed: createProposedRefName(name, revision),
		final:    createFinalRefName(name, revision),
	}
}

func setReference(storer storage.Storer, name plumbing.ReferenceName, hash plumbing.Hash) (*plumbing.Reference, error) {
	ref := plumbing.NewHashReference(name, hash)
	if err := storer.SetReference(ref); err != nil {
		return nil, err
	}
	return ref, nil
}
