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
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

var tracer = otel.Tracer("git")

const (
	DefaultMainReferenceName plumbing.ReferenceName = "refs/heads/main"
	OriginName               string                 = "origin"
)

type GitRepository interface {
	repository.Repository
	GetPackage(ctx context.Context, ref, path string) (repository.PackageRevision, kptfilev1.GitLock, error)
}

//go:generate stringer -type=MainBranchStrategy -linecomment
type MainBranchStrategy int

const (
	ErrorIfMissing   MainBranchStrategy = iota // ErrorIsMissing
	CreateIfMissing                            // CreateIfMissing
	SkipVerification                           // SkipVerification
)

type GitRepositoryOptions struct {
	CredentialResolver repository.CredentialResolver
	UserInfoProvider   repository.UserInfoProvider
	MainBranchStrategy MainBranchStrategy
}

func OpenRepository(ctx context.Context, name, namespace string, spec *configapi.GitRepository, root string, opts GitRepositoryOptions) (GitRepository, error) {
	ctx, span := tracer.Start(ctx, "OpenRepository", trace.WithAttributes())
	defer span.End()

	replace := strings.NewReplacer("/", "-", ":", "-")
	dir := filepath.Join(root, replace.Replace(spec.Repo))

	// Cleanup the cache directory in case initialization fails.
	cleanup := dir
	defer func() {
		if cleanup != "" {
			os.RemoveAll(cleanup)
		}
	}()

	var repo *git.Repository

	if fi, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		r, err := initEmptyRepository(dir)
		if err != nil {
			return nil, fmt.Errorf("error cloning git repository %q: %w", spec.Repo, err)
		}

		repo = r
	} else if !fi.IsDir() {
		// Internal error - corrupted cache. We will cleanup on the way out.
		return nil, fmt.Errorf("cannot clone git repository %q: %w", spec.Repo, err)
	} else {
		cleanup = "" // Existing directory; do not delete it.

		r, err := openRepository(dir)
		if err != nil {
			return nil, err
		}

		repo = r
	}

	// Create Remote
	if err := initializeOrigin(repo, spec.Repo); err != nil {
		return nil, fmt.Errorf("error cloning git repository %q, cannot create remote: %v", spec.Repo, err)
	}

	branch := MainBranch
	if spec.Branch != "" {
		branch = BranchName(spec.Branch)
	}

	repository := &gitRepository{
		name:               name,
		namespace:          namespace,
		repo:               repo,
		branch:             branch,
		directory:          strings.Trim(spec.Directory, "/"),
		secret:             spec.SecretRef.Name,
		credentialResolver: opts.CredentialResolver,
		userInfoProvider:   opts.UserInfoProvider,
	}

	if err := repository.fetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	if err := repository.verifyRepository(ctx, &opts); err != nil {
		return nil, err
	}

	cleanup = "" // Success. Keep the git directory.

	return repository, nil
}

type gitRepository struct {
	name               string     // Repository resource name
	namespace          string     // Repository resource namespace
	secret             string     // Name of the k8s Secret resource containing credentials
	branch             BranchName // The main branch from repository registration (defaults to 'main' if unspecified)
	directory          string     // Directory within the repository where to look for packages.
	repo               *git.Repository
	credentialResolver repository.CredentialResolver
	userInfoProvider   repository.UserInfoProvider

	// credential contains the information needed to authenticate against
	// a git repository.
	credential repository.Credential
	mutex      sync.Mutex
}

func (r *gitRepository) ListPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter) ([]repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::ListPackageRevisions", trace.WithAttributes())
	defer span.End()

	if err := r.fetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}

	var main *plumbing.Reference
	var drafts []repository.PackageRevision
	var result []repository.PackageRevision

	mainBranch := r.branch.RefInLocal() // Looking for the registered branch

	for {
		ref, err := refs.Next()
		if err == io.EOF {
			break
		}

		switch name := ref.Name(); {
		case name == mainBranch:
			main = ref
			continue

		case isProposedBranchNameInLocal(ref.Name()), isDraftBranchNameInLocal(ref.Name()):
			draft, err := r.loadDraft(ctx, ref)
			if err != nil {
				return nil, fmt.Errorf("failed to load package draft %q: %w", name.String(), err)
			}
			if draft != nil {
				drafts = append(drafts, draft)
			} else {
				klog.Warningf("no package draft found for ref %v", ref)
			}
		case isTagInLocalRepo(ref.Name()):
			tagged, err := r.loadTaggedPackages(ctx, ref)
			if err != nil {
				// this tag is not associated with any package (e.g. could be a release tag)
				continue
			}
			for _, p := range tagged {
				if filter.Matches(p) {
					result = append(result, p)
				}
			}
		}
	}

	if main != nil {
		// TODO: ignore packages that are unchanged in main branch, compared to a tagged version?
		mainpkgs, err := r.discoverFinalizedPackages(ctx, main)
		if err != nil {
			return nil, err
		}
		for _, p := range mainpkgs {
			if filter.Matches(p) {
				result = append(result, p)
			}
		}
	}

	for _, p := range drafts {
		if filter.Matches(p) {
			result = append(result, p)
		}
	}

	return result, nil
}

func (r *gitRepository) CreatePackageRevision(ctx context.Context, obj *v1alpha1.PackageRevision) (repository.PackageDraft, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::CreatePackageRevision", trace.WithAttributes())
	defer span.End()

	if !packageInDirectory(obj.Spec.PackageName, r.directory) {
		return nil, fmt.Errorf("cannot create %s outside of repository directory %q", obj.Spec.PackageName, r.directory)
	}

	var base plumbing.Hash
	refName := r.branch.RefInLocal()
	switch main, err := r.repo.Reference(refName, true); {
	case err == nil:
		base = main.Hash()
	case err == plumbing.ErrReferenceNotFound:
		// reference not found - empty repository. Package draft has no parent commit
	default:
		return nil, fmt.Errorf("error when resolving target branch for the package: %w", err)
	}

	draft := createDraftName(obj.Spec.PackageName, obj.Spec.Revision)

	return &gitPackageDraft{
		parent:    r,
		path:      obj.Spec.PackageName,
		revision:  obj.Spec.Revision,
		lifecycle: v1alpha1.PackageRevisionLifecycleDraft,
		updated:   time.Now(),
		base:      nil, // Creating a new package
		tasks:     nil, // Creating a new package
		branch:    draft,
		commit:    base,
	}, nil
}

func (r *gitRepository) UpdatePackage(ctx context.Context, old repository.PackageRevision) (repository.PackageDraft, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::UpdatePackage", trace.WithAttributes())
	defer span.End()

	oldGitPackage, ok := old.(*gitPackageRevision)
	if !ok {
		return nil, fmt.Errorf("cannot update non-git package %T", old)
	}

	ref := oldGitPackage.ref
	if ref == nil {
		return nil, fmt.Errorf("cannot update final package")
	}

	head, err := r.repo.Reference(ref.Name(), true)
	if err != nil {
		return nil, fmt.Errorf("cannot find draft package branch %q: %w", ref.Name(), err)
	}

	rev, err := r.loadDraft(ctx, head)
	if err != nil {
		return nil, fmt.Errorf("cannot load draft package: %w", err)
	}
	if rev == nil {
		return nil, fmt.Errorf("cannot load draft package %q (package not found)", ref.Name())
	}

	return &gitPackageDraft{
		parent:    r,
		path:      oldGitPackage.path,
		revision:  oldGitPackage.revision,
		lifecycle: oldGitPackage.Lifecycle(),
		updated:   rev.updated,
		base:      rev.ref,
		tree:      rev.tree,
		commit:    rev.commit,
		tasks:     rev.tasks,
	}, nil
}

func (r *gitRepository) DeletePackageRevision(ctx context.Context, old repository.PackageRevision) error {
	ctx, span := tracer.Start(ctx, "gitRepository::DeletePackageRevision", trace.WithAttributes())
	defer span.End()

	oldGit, ok := old.(*gitPackageRevision)
	if !ok {
		return fmt.Errorf("cannot delete non-git package: %T", old)
	}

	ref := oldGit.ref
	if ref == nil {
		// This is an internal error. In some rare cases (see GetPackage below) we create
		// package revisions without refs. They should never be returned via the API though.
		return fmt.Errorf("cannot delete package with no ref: %s", oldGit.path)
	}

	// We can only delete packages which have their own ref. Refs that are shared with other packages
	// (main branch, tag that doesn't contain package path in its name, ...) cannot be deleted.

	refSpecs := newPushRefSpecBuilder()

	switch rn := ref.Name(); {
	case rn.IsTag():
		// Delete tag only if it is package-specific.
		name := createFinalTagNameInLocal(oldGit.path, oldGit.revision)
		if rn != name {
			return fmt.Errorf("cannot delete package tagged with a tag that is not specific to the package: %s", rn)
		}

		// Delete the tag
		refSpecs.AddRefToDelete(ref)

	case isDraftBranchNameInLocal(rn), isProposedBranchNameInLocal(rn):
		// PackageRevision is proposed or draft; delete the branch directly.
		refSpecs.AddRefToDelete(ref)

	case isBranchInLocalRepo(rn):
		// Delete package from the branch
		commitHash, err := r.createPackageDeleteCommit(ctx, rn, oldGit)
		if err != nil {
			return err
		}

		// Update the reference
		refSpecs.AddRefToPush(commitHash, rn)

	default:
		return fmt.Errorf("cannot delete package with the ref name %s", rn)
	}

	// Update references
	if err := r.pushAndCleanup(ctx, refSpecs); err != nil {
		return fmt.Errorf("failed to update git references: %v", err)
	}
	return nil
}

func (r *gitRepository) GetPackage(ctx context.Context, version, path string) (repository.PackageRevision, kptfilev1.GitLock, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::GetPackage", trace.WithAttributes())
	defer span.End()

	git := r.repo

	var hash plumbing.Hash

	// Trim leading and trailing slashes
	path = strings.Trim(path, "/")

	// Versions map to git tags in one of two ways:
	//
	// * directly (tag=version)- but then this means that all packages in the repo must be versioned together.
	// * prefixed (tag=<packageDir/<version>) - solving the co-versioning problem.
	//
	// We have to check both forms when looking up a version.
	refNames := []string{}
	if path != "" {
		refNames = append(refNames, path+"/"+version)
		// HACK: Is this always refs/remotes/origin ?  Is it ever not (i.e. do we need both forms?)
		refNames = append(refNames, "refs/remotes/origin/"+path+"/"+version)
	}
	refNames = append(refNames, version)
	// HACK: Is this always refs/remotes/origin ?  Is it ever not (i.e. do we need both forms?)
	refNames = append(refNames, "refs/remotes/origin/"+version)

	for _, ref := range refNames {
		if resolved, err := git.ResolveRevision(plumbing.Revision(ref)); err != nil {
			if errors.Is(err, plumbing.ErrReferenceNotFound) {
				continue
			}
			return nil, kptfilev1.GitLock{}, fmt.Errorf("error resolving git reference %q: %w", ref, err)
		} else {
			hash = *resolved
			break
		}
	}

	if hash.IsZero() {
		r.dumpAllRefs()

		return nil, kptfilev1.GitLock{}, fmt.Errorf("cannot find git reference (tried %v)", refNames)
	}

	return r.loadPackageRevision(ctx, version, path, hash)
}

func (r *gitRepository) loadPackageRevision(ctx context.Context, version, path string, hash plumbing.Hash) (repository.PackageRevision, kptfilev1.GitLock, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::loadPackageRevision", trace.WithAttributes())
	defer span.End()

	if !packageInDirectory(path, r.directory) {
		return nil, kptfilev1.GitLock{}, fmt.Errorf("cannot find package %s@%s; package is not under the Repository.spec.directory", path, version)
	}

	git := r.repo

	origin, err := git.Remote("origin")
	if err != nil {
		return nil, kptfilev1.GitLock{}, fmt.Errorf("cannot determine repository origin: %w", err)
	}

	lock := kptfilev1.GitLock{
		Repo:      origin.Config().URLs[0],
		Directory: path,
		Ref:       version,
	}

	commit, err := git.CommitObject(hash)
	if err != nil {
		return nil, lock, fmt.Errorf("cannot resolve git reference %s (hash: %s) to commit: %w", version, hash, err)
	}
	lock.Commit = commit.Hash.String()

	krmPackage, err := r.FindPackage(commit, path)
	if err != nil {
		return nil, lock, err
	}

	if krmPackage == nil {
		return nil, lock, fmt.Errorf("cannot find package %s@%s", path, version)
	}

	var ref *plumbing.Reference = nil // Cannot determine ref; this package will be considered final (immutable).

	packageRevision, err := krmPackage.buildGitPackageRevision(ctx, version, ref)
	if err != nil {
		return nil, lock, err
	}
	return packageRevision, lock, nil
}

func (r *gitRepository) discoverFinalizedPackages(ctx context.Context, ref *plumbing.Reference) ([]repository.PackageRevision, error) {
	ctx, span := tracer.Start(ctx, "gitRepository::discoverFinalizedPackages", trace.WithAttributes())
	defer span.End()

	git := r.repo
	commit, err := git.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	var revision string
	if rev, ok := getBranchNameInLocalRepo(ref.Name()); ok {
		revision = rev
	} else if rev, ok = getTagNameInLocalRepo(ref.Name()); ok {
		revision = rev
	} else {
		// TODO: ignore the ref instead?
		return nil, fmt.Errorf("cannot determine revision from ref: %q", rev)
	}

	krmPackages, err := r.DiscoverPackagesInTree(commit, DiscoverPackagesOptions{FilterPrefix: r.directory, Recurse: true})
	if err != nil {
		return nil, err
	}

	var result []repository.PackageRevision
	for _, krmPackage := range krmPackages.packages {
		packageRevision, err := krmPackage.buildGitPackageRevision(ctx, revision, ref)
		if err != nil {
			return nil, err
		}
		result = append(result, packageRevision)
	}
	return result, nil
}

// loadDraft will load the draft package.  If the package isn't found (we now require a Kptfile), it will return (nil, nil)
func (r *gitRepository) loadDraft(ctx context.Context, ref *plumbing.Reference) (*gitPackageRevision, error) {
	name, revision, err := parseDraftName(ref)
	if err != nil {
		return nil, err
	}

	// Only load drafts in the directory specified at repository registration.
	if !packageInDirectory(name, r.directory) {
		return nil, nil
	}

	commit, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("cannot resolve draft branch to commit (corrupted repository?): %w", err)
	}

	krmPackage, err := r.FindPackage(commit, name)
	if err != nil {
		return nil, err
	}

	if krmPackage == nil {
		klog.Warningf("draft package %q was not found", name)
		return nil, nil
	}

	packageRevision, err := krmPackage.buildGitPackageRevision(ctx, revision, ref)
	if err != nil {
		return nil, err
	}

	return packageRevision, nil
}

func parseDraftName(draft *plumbing.Reference) (name, revision string, err error) {
	refName := draft.Name()
	var suffix string
	if b, ok := getDraftBranchNameInLocal(refName); ok {
		suffix = string(b)
	} else if b, ok = getProposedBranchNameInLocal(refName); ok {
		suffix = string(b)
	} else {
		return "", "", fmt.Errorf("invalid draft ref name: %q", refName)
	}

	revIndex := strings.LastIndex(suffix, "/")
	if revIndex <= 0 {
		return "", "", fmt.Errorf("invalid draft ref name; missing revision suffix: %q", refName)
	}
	name, revision = suffix[:revIndex], suffix[revIndex+1:]
	return name, revision, nil
}

func (r *gitRepository) loadTaggedPackages(ctx context.Context, tag *plumbing.Reference) ([]repository.PackageRevision, error) {
	name, ok := getTagNameInLocalRepo(tag.Name())
	if !ok {
		return nil, fmt.Errorf("invalid tag ref: %q", tag)
	}
	slash := strings.LastIndex(name, "/")

	if slash < 0 {
		// tag=<version>
		// could be a release tag or something else, we ignore these types of tags
		return nil, nil

	}

	// tag=<package path>/version
	path, revision := name[:slash], name[slash+1:]

	if !packageInDirectory(path, r.directory) {
		return nil, nil
	}

	commit, err := r.repo.CommitObject(tag.Hash())
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tag %q to commit (corrupted repository?): %w", name, err)
	}

	krmPackage, err := r.FindPackage(commit, path)
	if err != nil {
		klog.Warningf("Skipping %q; cannot find %q (corrupted repository?): %w", name, path, err)
		return nil, nil
	}

	if krmPackage == nil {
		klog.Warningf("Skipping %q: Kptfile not found: %w", name, err)
		return nil, nil
	}

	packageRevision, err := krmPackage.buildGitPackageRevision(ctx, revision, tag)
	if err != nil {
		return nil, err
	}

	return []repository.PackageRevision{
		packageRevision,
	}, nil

}

func (r *gitRepository) dumpAllRefs() {
	refs, err := r.repo.References()
	if err != nil {
		klog.Warningf("failed to get references: %v", err)
	} else {
		for {
			ref, err := refs.Next()
			if err != nil {
				if err != io.EOF {
					klog.Warningf("failed to get next reference: %v", err)
				}
				break
			}
			klog.Infof("ref %#v", ref.Name())
		}
	}

	branches, err := r.repo.Branches()
	if err != nil {
		klog.Warningf("failed to get branches: %v", err)
	} else {
		for {
			branch, err := branches.Next()
			if err != nil {
				if err != io.EOF {
					klog.Warningf("failed to get next branch: %v", err)
				}
				break
			}
			klog.Infof("branch %#v", branch.Name())
		}
	}
}

// getAuthMethod fetches the credentials for authenticating to git. It caches the
// credentials between calls and refresh credentials when the tokens have expired.
func (r *gitRepository) getAuthMethod(ctx context.Context, forceRefresh bool) (transport.AuthMethod, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// If no secret is provided, we try without any auth.
	if r.secret == "" {
		return nil, nil
	}

	if r.credential == nil || !r.credential.Valid() || forceRefresh {
		if cred, err := r.credentialResolver.ResolveCredential(ctx, r.namespace, r.secret); err != nil {
			return nil, fmt.Errorf("failed to obtain credential from secret %s/%s: %w", r.namespace, r.secret, err)
		} else {
			r.credential = cred
		}
	}

	return r.credential.ToAuthMethod(), nil
}

func (r *gitRepository) getRepo() (string, error) {
	origin, err := r.repo.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("cannot determine repository origin: %w", err)
	}

	return origin.Config().URLs[0], nil
}

func (r *gitRepository) fetchRemoteRepository(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "gitRepository::fetchRemoteRepository", trace.WithAttributes())
	defer span.End()

	// Fetch
	switch err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return r.repo.Fetch(&git.FetchOptions{
			RemoteName: OriginName,
			Auth:       auth,
			Prune:      git.Prune,
		})
	}); err {
	case nil: // OK
	case git.NoErrAlreadyUpToDate:
	case transport.ErrEmptyRemoteRepository:

	default:
		return fmt.Errorf("cannot fetch repository %s/%s: %w", r.namespace, r.name, err)
	}

	return nil
}

// Verifies repository. Repository must be fetched already.
func (r *gitRepository) verifyRepository(ctx context.Context, opts *GitRepositoryOptions) error {
	// When opening a temporary repository, such as for cloning a package
	// from unregistered upstream, we won't be pushing into the remote so
	// we don't need to verify presence of the main branch.
	if opts.MainBranchStrategy == SkipVerification {
		return nil
	}

	if _, err := r.repo.Reference(r.branch.RefInLocal(), false); err != nil {
		switch opts.MainBranchStrategy {
		case ErrorIfMissing:
			return fmt.Errorf("branch %q doesn't exist: %v", r.branch, err)
		case CreateIfMissing:
			klog.Info("Creating branch %s in repository %s", r.branch, r.name)
			if err := r.createBranch(ctx, r.branch); err != nil {
				return fmt.Errorf("error creating main branch %q: %v", r.branch, err)
			}
		default:
			return fmt.Errorf("unknown main branch strategy %q", opts.MainBranchStrategy.String())
		}
	}
	return nil
}

const (
	fileContent   = "Created by porch"
	fileName      = "README.md"
	commitMessage = "Initial commit for main branch by porch"
)

// createBranch creates the provided branch by creating a commit containing
// a README.md file on the root of the repo and then pushing it to the branch.
func (r *gitRepository) createBranch(ctx context.Context, branch BranchName) error {
	fileHash, err := storeBlob(r.repo.Storer, fileContent)
	if err != nil {
		return err
	}

	tree := &object.Tree{}
	tree.Entries = append(tree.Entries, object.TreeEntry{
		Name: fileName,
		Mode: filemode.Regular,
		Hash: fileHash,
	})

	treeEo := r.repo.Storer.NewEncodedObject()
	if err := tree.Encode(treeEo); err != nil {
		return err
	}

	treeHash, err := r.repo.Storer.SetEncodedObject(treeEo)
	if err != nil {
		return err
	}

	now := time.Now()
	commit := &object.Commit{
		Author: object.Signature{
			Name:  porchSignatureName,
			Email: porchSignatureEmail,
			When:  now,
		},
		Committer: object.Signature{
			Name:  porchSignatureName,
			Email: porchSignatureEmail,
			When:  now,
		},
		Message:  commitMessage,
		TreeHash: treeHash,
	}
	commitHash, err := storeCommit(r.repo.Storer, plumbing.ZeroHash, commit)
	if err != nil {
		return err
	}

	refSpecs := newPushRefSpecBuilder()
	refSpecs.AddRefToPush(commitHash, branch.RefInLocal())
	return r.pushAndCleanup(ctx, refSpecs)
}

// Creates a commit which deletes the package from the branch, and returns its commit hash.
// If the branch doesn't exist, will return zero hash and no error.
func (r *gitRepository) createPackageDeleteCommit(ctx context.Context, branch plumbing.ReferenceName, pkg *gitPackageRevision) (plumbing.Hash, error) {
	var zero plumbing.Hash
	repo := r.repo

	local, err := refInRemoteFromRefInLocal(branch)
	if err != nil {
		return zero, err
	}
	// Fetch the branch
	// TODO: Fetch only as part of conflict resolution & Retry
	switch err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return repo.Fetch(&git.FetchOptions{
			RemoteName: OriginName,
			RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("+%s:%s", local, branch))},
			Auth:       auth,
			Tags:       git.NoTags,
		})
	}); err {
	case nil, git.NoErrAlreadyUpToDate:
		// ok
	default:
		return zero, fmt.Errorf("failed to fetch remote repository: %w", err)
	}

	// find the branch
	ref, err := repo.Reference(branch, true)
	if err != nil {
		// branch doesn't exist, and therefore package doesn't exist either.
		klog.Infof("Branch %q no longer exist, deleting a package from it is unnecessary", branch)
		return zero, nil
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return zero, fmt.Errorf("failed to resolve main branch to commit: %w", err)
	}
	root, err := commit.Tree()
	if err != nil {
		return zero, fmt.Errorf("failed to find commit tree for %s: %w", ref, err)
	}

	packagePath := pkg.path

	// Find the package in the tree
	switch _, err := root.FindEntry(packagePath); err {
	case object.ErrEntryNotFound:
		// Package doesn't exist; no need to delete it
		return zero, nil
	case nil:
		// found
	default:
		return zero, fmt.Errorf("failed to find package %q in the repositrory ref %q: %w,", packagePath, ref, err)
	}

	// Create commit helper. Use zero hash for the initial package tree. Commit helper will initialize trees
	// without TreeEntry for this package present - the package is deleted.
	ch, err := newCommitHelper(repo, r.userInfoProvider, commit.Hash, packagePath, zero)
	if err != nil {
		return zero, fmt.Errorf("failed to initialize commit of package %q to %q: %w", packagePath, ref, err)
	}

	message := fmt.Sprintf("Delete %s", packagePath)
	commitHash, _, err := ch.commit(ctx, message, packagePath)
	if err != nil {
		return zero, fmt.Errorf("failed to commit package %q to %q: %w", packagePath, ref, err)
	}
	return commitHash, nil
}

func (r *gitRepository) pushAndCleanup(ctx context.Context, ph *pushRefSpecBuilder) error {
	specs, require, err := ph.BuildRefSpecs()
	if err != nil {
		return err
	}

	klog.Infof("pushing refs: %v", specs)

	if err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return r.repo.Push(&git.PushOptions{
			RemoteName:        OriginName,
			RefSpecs:          specs,
			Auth:              auth,
			RequireRemoteRefs: require,
			// TODO(justinsb): Need to ensure this is a compare-and-swap
			Force: true,
		})
	}); err != nil {
		return err
	}
	return nil
}

func (r *gitRepository) loadTasks(ctx context.Context, startCommit *object.Commit, packagePath, revision string) ([]v1alpha1.Task, error) {
	var logOptions = git.LogOptions{
		From:  startCommit.Hash,
		Order: git.LogOrderCommitterTime,
	}

	// NOTE: We don't prune the commits with the filepath; this is because it's a relatively expensive operation,
	// as we have to visit the whole trees.  Visiting the commits is comparatively fast.
	// // Prune the commits we visit a bit - though the actual gate is on the gitAnnotation
	// if packagePath != "" {
	// 	if !strings.HasSuffix(packagePath, "/") {
	// 		packagePath += "/"
	// 	}
	// 	pathFilter := func(p string) bool {
	// 		matchesPackage := strings.HasPrefix(p, packagePath)
	// 		return matchesPackage
	// 	}
	// 	logOptions.PathFilter = pathFilter
	// }

	commits, err := r.repo.Log(&logOptions)
	if err != nil {
		return nil, fmt.Errorf("error walking commits: %w", err)
	}

	var tasks []v1alpha1.Task

	visitCommit := func(commit *object.Commit) error {
		gitAnnotations, err := ExtractGitAnnotations(commit)
		if err != nil {
			return err
		}

		for _, gitAnnotation := range gitAnnotations {
			if gitAnnotation.PackagePath == packagePath && gitAnnotation.Revision == revision {
				// We are iterating through the commits in reverse order.
				// Tasks that are read from separate commits will be recorded in
				// reverse order.
				// The entire `tasks` slice will get reversed later, which will give us the
				// tasks in chronological order.
				if gitAnnotation.Task != nil {
					tasks = append(tasks, *gitAnnotation.Task)
				}

				lastTask := tasks[len(tasks)-1]
				if lastTask.Type == v1alpha1.TaskTypeClone || lastTask.Type == v1alpha1.TaskTypeInit {
					// we have reached the beginning of this package revision and don't need to
					// continue further
					break
				}
			}
		}

		// TODO: If a commit has no annotations defined, we should treat it like a patch.
		// This will allow direct manipulation of the git repo.
		// We should also probably _not_ record an annotation for a patch task, so we
		// can allow direct editing.
		return nil
	}

	if err := commits.ForEach(visitCommit); err != nil {
		return nil, fmt.Errorf("error visiting commits: %w", err)
	}

	// We need to reverse the tasks so they appear in chronological order
	reverseSlice(tasks)

	return tasks, nil
}

// See https://eli.thegreenplace.net/2021/generic-functions-on-slices-with-go-type-parameters/
// func ReverseSlice[T any](s []T) { // Ready for generics!
func reverseSlice(s []v1alpha1.Task) {
	first := 0
	last := len(s) - 1
	for first < last {
		s[first], s[last] = s[last], s[first]
		first++
		last--
	}
}
