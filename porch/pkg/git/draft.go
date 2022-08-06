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
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
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
	tasks     []v1alpha1.Task
}

var _ repository.PackageDraft = &gitPackageDraft{}

func (d *gitPackageDraft) UpdateResources(ctx context.Context, new *v1alpha1.PackageRevisionResources, change *v1alpha1.Task) error {
	ctx, span := tracer.Start(ctx, "gitPackageDraft::UpdateResources", trace.WithAttributes())
	defer span.End()

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

	annotation := &gitAnnotation{
		PackagePath: d.path,
		Revision:    d.revision,
		Task:        change,
	}
	message := "Intermediate commit"
	if change != nil {
		message += fmt.Sprintf(": %s", change.Type)
		d.tasks = append(d.tasks, *change)
	}
	message += "\n"

	message, err = AnnotateCommitMessage(message, annotation)
	if err != nil {
		return err
	}

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
	ctx, span := tracer.Start(ctx, "gitPackageDraft::Close", trace.WithAttributes())
	defer span.End()

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
		repo:     d.parent,
		path:     d.path,
		revision: d.revision,
		updated:  d.updated,
		ref:      newRef,
		tree:     d.tree,
		commit:   newRef.Hash(),
		tasks:    d.tasks,
	}, nil
}

// doGitWithAuth fetches auth information for git and provides it
// to the provided function which performs the operation against a git repo.
func (r *gitRepository) doGitWithAuth(ctx context.Context, op func(transport.AuthMethod) error) error {
	auth, err := r.getAuthMethod(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to obtain git credentials: %w", err)
	}
	err = op(auth)
	if err != nil {
		if !errors.Is(err, transport.ErrAuthenticationRequired) {
			return err
		}
		klog.Infof("Authentication failed. Trying to refresh credentials")
		// TODO: Consider having some kind of backoff here.
		auth, err := r.getAuthMethod(ctx, true)
		if err != nil {
			return fmt.Errorf("failed to obtain git credentials: %w", err)
		}
		return op(auth)
	}
	return nil
}

func (r *gitRepository) commitPackageToMain(ctx context.Context, d *gitPackageDraft) (commitHash, newPackageTreeHash plumbing.Hash, base *plumbing.Reference, err error) {
	branch := r.branch
	localRef := branch.RefInLocal()

	var zero plumbing.Hash

	repo := r.repo

	// Fetch main
	switch err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return repo.Fetch(&git.FetchOptions{
			RemoteName: OriginName,
			RefSpecs:   []config.RefSpec{branch.ForceFetchSpec()},
			Auth:       auth,
		})
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
	packageRevision := d.revision

	// Fetch the commits belonging to this package revision.
	commits, err := r.loadPackageCommits(d.commit, packagePath, packageRevision)
	if err != nil {
		return zero, zero, nil, fmt.Errorf("failed to load commits for package %s: %w", packagePath, err)
	}
	reverseCommitSlice(commits)

	// TODO: Check for out-of-band update of the package in main branch
	// (compare package tree in target branch and common base)
	var ch *commitHelper
	if len(commits) == 0 {
		// If we can't find any commits, the draft might have been created outside of porch. We
		// just add the changes to the main branch in a single commit.
		ch, err = newCommitHelper(repo, r.userInfoProvider, headCommit.Hash, packagePath, d.tree)
		if err != nil {
			return zero, zero, nil, fmt.Errorf("failed to initialize commit of package %s to %s", packagePath, localRef)
		}
	} else {
		// If we have commits, we reproduce the same commits on the main branch to keep
		// the history from the draft.
		ch, err = newCommitHelper(repo, r.userInfoProvider, headCommit.Hash, packagePath, plumbing.ZeroHash)
		if err != nil {
			return zero, zero, nil, fmt.Errorf("failed to initialize commit of package %s to %s", packagePath, localRef)
		}

		for _, commit := range commits {
			// Look up the tree for the package in the commit.
			t, err := commit.Tree()
			if err != nil {
				return zero, zero, nil, fmt.Errorf("cannot resolve commit's (%s) tree (%s) to tree object: %w", t.Hash, commit.TreeHash, err)
			}
			te, err := t.FindEntry(packagePath)
			if err != nil {
				return zero, zero, nil, err
			}

			err = ch.storeTree(packagePath, te.Hash)
			if err != nil {
				return zero, zero, nil, err
			}

			_, _, err = ch.commit(ctx, commit.Message, packagePath)
			if err != nil {
				return zero, zero, nil, fmt.Errorf("failed to commit package %s to %s", packagePath, localRef)
			}
		}
	}

	// Add a commit without changes to mark that the package revision is approved.
	message := fmt.Sprintf("Approve %s/%s", packagePath, d.revision)
	commitHash, newPackageTreeHash, err = ch.commit(ctx, message, packagePath)
	if err != nil {
		return zero, zero, nil, fmt.Errorf("failed to commit package %s to %s", packagePath, localRef)
	}

	return commitHash, newPackageTreeHash, localTarget, nil
}

// TODO: This is an almost direct copy of
// https://github.com/GoogleContainerTools/kpt/blob/3c3288af0c4c4a7e07ffeb6fe473d32afd81137b/porch/pkg/git/git.go#L860.
// Fix it when we can use generics.
func reverseCommitSlice(s []*object.Commit) {
	first := 0
	last := len(s) - 1
	for first < last {
		s[first], s[last] = s[last], s[first]
		first++
		last--
	}
}

// loadPackageCommits looks through the commit log starting at the provided
// commitHash and fetches all commits with a matching packagePath and revision.
func (r *gitRepository) loadPackageCommits(commitHash plumbing.Hash, packagePath, revision string) ([]*object.Commit, error) {
	var logOptions = git.LogOptions{
		From:  commitHash,
		Order: git.LogOrderCommitterTime,
	}

	commits, err := r.repo.Log(&logOptions)
	if err != nil {
		return nil, fmt.Errorf("error walking commits: %w", err)
	}

	var packageRevCommits []*object.Commit

	visitCommit := func(commit *object.Commit) error {
		gitAnnotations, err := ExtractGitAnnotations(commit)
		if err != nil {
			return err
		}

		for _, gitAnnotation := range gitAnnotations {
			if gitAnnotation.PackagePath == packagePath && gitAnnotation.Revision == revision {
				packageRevCommits = append(packageRevCommits, commit)
			}
		}
		return nil
	}

	if err := commits.ForEach(visitCommit); err != nil {
		return nil, fmt.Errorf("error visiting commits: %w", err)
	}
	return packageRevCommits, nil
}
