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
	"path"
	"path/filepath"
	"strings"
	"time"

	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"k8s.io/klog/v2"
)

const (
	DefaultMainReferenceName plumbing.ReferenceName = "refs/heads/main"
	OriginName               string                 = "origin"
)

type GitRepository interface {
	repository.Repository
	GetPackage(ref, path string) (repository.PackageRevision, kptfilev1.GitLock, error)
}

type GitRepositoryOptions struct {
	CredentialResolver repository.CredentialResolver
	UserInfoProvider   repository.UserInfoProvider
}

func OpenRepository(ctx context.Context, name, namespace string, spec *configapi.GitRepository, root string, opts GitRepositoryOptions) (GitRepository, error) {
	replace := strings.NewReplacer("/", "-", ":", "-")
	dir := filepath.Join(root, replace.Replace(spec.Repo))

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
		// Internal error - corrupted cache.
		return nil, fmt.Errorf("cannot clone git repository %q: %w", spec.Repo, err)
	} else {
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
		// TODO: Validate branch name syntax (we can't check whether the branch exists;
		// the repository may be empty).
		branch = BranchName(spec.Branch)
	}

	repository := &gitRepository{
		name:               name,
		namespace:          namespace,
		repo:               repo,
		branch:             branch,
		secret:             spec.SecretRef.Name,
		credentialResolver: opts.CredentialResolver,
		userInfoProvider:   opts.UserInfoProvider,
	}

	if err := repository.fetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	return repository, nil
}

type gitRepository struct {
	name               string     // Repository resource name
	namespace          string     // Repository resource namespace
	secret             string     // Name of the k8s Secret resource containing credentials
	branch             BranchName // The main branch from repository registration (defaults to 'main' if unspecified)
	repo               *git.Repository
	cachedCredentials  transport.AuthMethod
	credentialResolver repository.CredentialResolver
	userInfoProvider   repository.UserInfoProvider
}

func (r *gitRepository) ListPackageRevisions(ctx context.Context) ([]repository.PackageRevision, error) {
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
			draft, err := r.loadDraft(ref)
			if err != nil {
				return nil, fmt.Errorf("failed to load package draft %q: %w", name.String(), err)
			}
			if draft != nil {
				drafts = append(drafts, draft)
			} else {
				klog.Warningf("no package draft found for ref %v", ref)
			}
		case isTagInLocalRepo(ref.Name()):
			tagged, err := r.loadTaggedPackages(ref)
			if err != nil {
				return nil, fmt.Errorf("failed to load packages from tag %q: %w", name, err)
			}
			result = append(result, tagged...)
		}
	}

	if main != nil {
		// TODO: ignore packages that are unchanged in main branch, compared to a tagged version?
		mainpkgs, err := r.discoverFinalizedPackages(main)
		if err != nil {
			return nil, err
		}
		result = append(result, mainpkgs...)
	}

	result = append(result, drafts...)

	return result, nil
}

func (r *gitRepository) CreatePackageRevision(ctx context.Context, obj *v1alpha1.PackageRevision) (repository.PackageDraft, error) {
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
		branch:    draft,
		commit:    base,
	}, nil
}

func (r *gitRepository) UpdatePackage(ctx context.Context, old repository.PackageRevision) (repository.PackageDraft, error) {
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

	rev, err := r.loadDraft(head)
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
		lifecycle: oldGitPackage.getPackageRevisionLifecycle(),
		updated:   rev.updated,
		base:      rev.ref,
		tree:      rev.tree,
		commit:    rev.commit,
	}, nil
}

func (r *gitRepository) DeletePackageRevision(ctx context.Context, old repository.PackageRevision) error {
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

func (r *gitRepository) GetPackage(version, path string) (repository.PackageRevision, kptfilev1.GitLock, error) {
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

	return r.loadPackageRevision(version, path, hash)
}

func (r *gitRepository) loadPackageRevision(version, path string, hash plumbing.Hash) (repository.PackageRevision, kptfilev1.GitLock, error) {
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

	commitTree, err := commit.Tree()
	if err != nil {
		return nil, lock, fmt.Errorf("cannot resolve git reference %s (hash %s) to tree: %w", version, hash, err)
	}
	treeHash := commitTree.Hash
	if path != "" {
		te, err := commitTree.FindEntry(path)
		if err != nil {
			return nil, lock, fmt.Errorf("cannot find package %s@%s: %w", path, version, err)
		}
		if te.Mode != filemode.Dir {
			return nil, lock, fmt.Errorf("path %s@%s is not a directory", path, version)
		}
		treeHash = te.Hash
	}

	return &gitPackageRevision{
		parent:   r,
		path:     path,
		revision: version,
		updated:  commit.Author.When,
		ref:      nil, // Cannot determine ref; this package will be considered final (immutable).
		tree:     treeHash,
		commit:   hash,
	}, lock, nil
}

func (r *gitRepository) discoverFinalizedPackages(ref *plumbing.Reference) ([]repository.PackageRevision, error) {
	git := r.repo
	commit, err := git.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
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

	var result []repository.PackageRevision
	if err := discoverPackagesInTree(git, tree, "", func(dir string, tree, kptfile plumbing.Hash) error {
		result = append(result, &gitPackageRevision{
			parent:   r,
			path:     dir,
			revision: revision,
			updated:  commit.Author.When,
			ref:      ref,
			tree:     tree,
			commit:   ref.Hash(),
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

type foundPackageCallback func(dir string, tree, kptfile plumbing.Hash) error

func discoverPackagesInTree(r *git.Repository, tree *object.Tree, dir string, found foundPackageCallback) error {
	for _, e := range tree.Entries {
		if e.Mode.IsRegular() && e.Name == "Kptfile" {
			// Found a package
			klog.Infof("Found package %q with Kptfile hash %q", path.Join(dir, e.Name), e.Hash)
			if err := found(dir, tree.Hash, e.Hash); err != nil {
				return err
			}
		}
	}

	for _, e := range tree.Entries {
		if e.Mode != filemode.Dir {
			continue
		}

		dirTree, err := r.TreeObject(e.Hash)
		if err != nil {
			return fmt.Errorf("error getting git tree %v: %w", e.Hash, err)
		}

		if err := discoverPackagesInTree(r, dirTree, path.Join(dir, e.Name), found); err != nil {
			return err
		}
	}
	return nil
}

// loadDraft will load the draft package.  If the package isn't found (we now require a Kptfile), it will return (nil, nil)
func (r *gitRepository) loadDraft(ref *plumbing.Reference) (*gitPackageRevision, error) {
	name, revision, err := parseDraftName(ref)
	if err != nil {
		return nil, err
	}

	commit, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("cannot resolve draft branch to commit (corrupted repository?): %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve package commit to tree (corrupted repository?): %w", err)
	}

	dirTree, err := tree.Tree(name)
	if err != nil {
		switch err {
		case object.ErrDirectoryNotFound, object.ErrEntryNotFound:
			// empty package
			return nil, nil

		default:
			return nil, fmt.Errorf("error when looking for package in the repository: %w", err)
		}
	}

	packageTree := dirTree.Hash
	kptfileEntry, err := dirTree.FindEntry("Kptfile")
	if err != nil {
		if err == object.ErrEntryNotFound {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error finding Kptfile: %w", err)
		}
	}
	if !kptfileEntry.Mode.IsRegular() {
		return nil, fmt.Errorf("found Kptfile which is not a regular file: %s", kptfileEntry.Mode)
	}

	return &gitPackageRevision{
		parent:   r,
		path:     name,
		revision: revision,
		updated:  commit.Author.When,
		ref:      ref,
		tree:     packageTree,
		commit:   ref.Hash(),
	}, nil
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

func (r *gitRepository) loadTaggedPackages(tag *plumbing.Reference) ([]repository.PackageRevision, error) {
	name, ok := getTagNameInLocalRepo(tag.Name())
	if !ok {
		return nil, fmt.Errorf("invalid tag ref: %q", tag)
	}
	slash := strings.LastIndex(name, "/")

	if slash < 0 {
		// tag=<version>
		return r.discoverFinalizedPackages(tag)
	}

	// tag=<package path>/version
	path, revision := name[:slash], name[slash+1:]

	commit, err := r.repo.CommitObject(tag.Hash())
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tag %q to commit (corrupted repository?): %w", name, err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tag %q to tree (corrupted repository?): %w", name, err)
	}

	dirTree, err := tree.Tree(path)
	if err != nil {
		klog.Warningf("Skipping %q; cannot find %q (corrupted repository?): %w", name, path, err)
		return nil, nil
	}

	if kptfileEntry, err := dirTree.FindEntry("Kptfile"); err != nil {
		klog.Warningf("Skipping %q: Kptfile not found: %w", name, err)
		return nil, nil
	} else if !kptfileEntry.Mode.IsRegular() {
		klog.Warningf("Skippping %q: Kptfile is not a file", name)
		return nil, nil
	}

	return []repository.PackageRevision{
		&gitPackageRevision{
			parent:   r,
			path:     path,
			revision: revision,
			updated:  commit.Author.When,
			ref:      tag,
			tree:     dirTree.Hash,
			commit:   tag.Hash(),
		},
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

func resolveCredential(ctx context.Context, namespace, name string, resolver repository.CredentialResolver) (transport.AuthMethod, error) {
	cred, err := resolver.ResolveCredential(ctx, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain credential from secret %s/%s: %w", namespace, name, err)
	}

	username := cred.Data["username"]
	password := cred.Data["password"]

	return &http.BasicAuth{
		Username: string(username),
		Password: string(password),
	}, nil
}

func (r *gitRepository) getAuthMethod(ctx context.Context) (transport.AuthMethod, error) {
	if r.cachedCredentials == nil {
		if r.secret != "" {
			if auth, err := resolveCredential(ctx, r.namespace, r.secret, r.credentialResolver); err != nil {
				return nil, err
			} else {
				r.cachedCredentials = auth
			}
		}
	}

	return r.cachedCredentials, nil
}

func (r *gitRepository) getRepo() (string, error) {
	origin, err := r.repo.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("cannot determine repository origin: %w", err)
	}

	return origin.Config().URLs[0], nil
}

func (r *gitRepository) fetchRemoteRepository(ctx context.Context) error {
	auth, err := r.getAuthMethod(ctx)
	if err != nil {
		return err
	}

	// Fetch
	switch err := r.repo.Fetch(&git.FetchOptions{
		RemoteName: OriginName,
		Auth:       auth,
		Prune:      git.Prune,
	}); err {
	case nil: // OK
	case git.NoErrAlreadyUpToDate:
	case transport.ErrEmptyRemoteRepository:

	default:
		return fmt.Errorf("cannot fetch repository %s/%s: %w", r.namespace, r.name, err)
	}

	return nil
}

// Creates a commit which deletes the package from the branch, and returns its commit hash.
// If the branch doesn't exist, will return zero hash and no error.
func (r *gitRepository) createPackageDeleteCommit(ctx context.Context, branch plumbing.ReferenceName, pkg *gitPackageRevision) (plumbing.Hash, error) {
	var zero plumbing.Hash
	auth, err := r.getAuthMethod(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to obtain git credentials: %w", err)
	}

	repo := r.repo

	local, err := refInRemoteFromRefInLocal(branch)
	if err != nil {
		return zero, err
	}
	// Fetch the branch
	// TODO: Fetch only as part of conflict resolution & Retry
	switch err := repo.Fetch(&git.FetchOptions{
		RemoteName: OriginName,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("+%s:%s", local, branch))},
		Auth:       auth,
		Tags:       git.NoTags,
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
	auth, err := r.getAuthMethod(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain git credentials: %w", err)
	}

	specs, require, err := ph.BuildRefSpecs()
	if err != nil {
		return err
	}

	if err := r.repo.Push(&git.PushOptions{
		RemoteName:        OriginName,
		RefSpecs:          specs,
		Auth:              auth,
		RequireRemoteRefs: require,
	}); err != nil {
		return err
	}
	return nil
}
