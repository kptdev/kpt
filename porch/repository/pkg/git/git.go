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
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
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
	refMain plumbing.ReferenceName = "refs/heads/main"
	// TODO: support customizable pattern of draft branches.
	refDraftPrefix                               = "refs/heads/drafts/"
	refProposedPrefix                            = "refs/heads/proposed/"
	refTagsPrefix                                = "refs/tags/"
	refHeadsPrefix                               = "refs/heads/"
	refRemoteBranchPrefix                        = "refs/remotes/origin/"
	refOriginMain         plumbing.ReferenceName = refRemoteBranchPrefix + "main"
	originName                                   = "origin"
)

type GitRepository interface {
	repository.Repository
	GetPackage(ref, path string) (repository.PackageRevision, kptfilev1.GitLock, error)
}

func OpenRepository(ctx context.Context, name, namespace string, spec *configapi.GitRepository, resolver repository.CredentialResolver, root string) (GitRepository, error) {
	replace := strings.NewReplacer("/", "-", ":", "-")
	dir := filepath.Join(root, replace.Replace(spec.Repo))

	var repo *git.Repository

	if fi, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		isBare := true
		r, err := git.PlainInit(dir, isBare)
		if err != nil {
			return nil, fmt.Errorf("error cloning git repository %q: %w", spec.Repo, err)
		}

		// Create Remote
		if _, err = r.CreateRemote(&config.RemoteConfig{
			Name: originName,
			URLs: []string{spec.Repo},
			Fetch: []config.RefSpec{
				config.RefSpec(fmt.Sprintf(config.DefaultFetchRefSpec, originName)),
			},
		}); err != nil {
			return nil, fmt.Errorf("error cloning git repository %q, cannot create remote: %v", spec.Repo, err)
		}

		repo = r
	} else if !fi.IsDir() {
		// Internal error - corrupted cache.
		return nil, fmt.Errorf("cannot clone git repository %q: %w", spec.Repo, err)
	} else {
		r, err := git.PlainOpen(dir)
		if err != nil {
			return nil, err
		}

		repo = r
	}

	repository := &gitRepository{
		name:               name,
		namespace:          namespace,
		repo:               repo,
		secret:             spec.SecretRef.Name,
		credentialResolver: resolver,
	}

	if err := repository.update(ctx); err != nil {
		return nil, err
	}

	return repository, nil
}

type gitRepository struct {
	name               string
	namespace          string
	secret             string
	repo               *git.Repository
	cachedCredentials  transport.AuthMethod
	credentialResolver repository.CredentialResolver
}

func (r *gitRepository) ListPackageRevisions(ctx context.Context) ([]repository.PackageRevision, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}

	var main *plumbing.Reference
	var drafts []repository.PackageRevision
	var result []repository.PackageRevision

	for {
		ref, err := refs.Next()
		if err == io.EOF {
			break
		}

		switch name := ref.Name(); {
		case name == refMain:
			main = ref
			continue

		case strings.HasPrefix(name.String(), refProposedPrefix):
			fallthrough
		case strings.HasPrefix(name.String(), refDraftPrefix):
			draft, err := r.loadDraft(ref)
			if err != nil {
				return nil, fmt.Errorf("failed to load package draft %q: %w", name.String(), err)
			}
			drafts = append(drafts, draft)

		case strings.HasPrefix(name.String(), refTagsPrefix):
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
	switch main, err := r.repo.Reference(refMain, true); {
	case err == nil:
		base = main.Hash()
	case err == plumbing.ErrReferenceNotFound:
		// reference not found - empty repository. Package draft has no parent commit
	default:
		return nil, fmt.Errorf("error when resolving target branch for the package: %w", err)
	}
	ref := createDraftRefName(obj.Spec.PackageName, obj.Spec.Revision)
	head := plumbing.NewHashReference(ref, base)
	if err := r.repo.Storer.SetReference(head); err != nil {
		return nil, err
	}

	return &gitPackageDraft{
		lifecycle: v1alpha1.PackageRevisionLifecycleDraft,
		parent:    r,
		path:      obj.Spec.PackageName,
		revision:  obj.Spec.Revision,
		updated:   time.Now(),
		ref:       head,
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

	return &gitPackageDraft{
		parent:    r,
		path:      oldGitPackage.path,
		revision:  oldGitPackage.revision,
		lifecycle: oldGitPackage.getPackageRevisionLifecycle(),
		updated:   rev.updated,
		ref:       rev.ref,
		tree:      rev.tree,
		commit:    rev.commit,
	}, nil
}

func (r *gitRepository) ApprovePackageRevision(ctx context.Context, path, revision string) (repository.PackageRevision, error) {
	refName := createDraftRefName(path, revision)
	oldRef, err := r.repo.Reference(refName, true)
	if err != nil {
		return nil, fmt.Errorf("cannot find draft package branch %q: %w", refName, err)
	}

	auth, err := r.getAuthMethod(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain git credentials: %w", err)
	}

	approvedName := createApprovedRefName(path, revision)

	newRef := plumbing.NewHashReference(approvedName, oldRef.Hash())

	options := &git.PushOptions{
		RemoteName:        "origin",
		RefSpecs:          []config.RefSpec{},
		Auth:              auth,
		RequireRemoteRefs: []config.RefSpec{},
	}

	options.RefSpecs = append(options.RefSpecs, config.RefSpec(fmt.Sprintf("%s:%s", oldRef.Hash(), newRef.Name())))

	currentNewRefValue, err := r.repo.Storer.Reference(newRef.Name())
	if err == nil {
		options.RequireRemoteRefs = append(options.RequireRemoteRefs, config.RefSpec(fmt.Sprintf("%s:%s", currentNewRefValue.Hash(), newRef.Name())))
	} else if err == plumbing.ErrReferenceNotFound {
		// TODO: Should we push with 000000 ?
	} else {
		return nil, fmt.Errorf("error getting reference %q: %w", newRef.Name(), err)
	}

	klog.Infof("pushing with options %v", options)

	// Note that we push and _then_ we set the local reference to avoid drift
	if err := r.repo.Push(options); err != nil {
		return nil, fmt.Errorf("failed to push to git %#v: %w", options, err)
	}

	if err := r.repo.Storer.SetReference(newRef); err != nil {
		return nil, fmt.Errorf("error storing git reference %v: %w", newRef, err)
	}

	approved, _, err := r.loadPackageRevision(revision, path, newRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("cannot load approved package: %w", err)
	}
	return approved, nil
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

	refsPush := []config.RefSpec{}
	refsDelete := map[plumbing.ReferenceName]bool{}

	switch rn := ref.Name(); {
	case rn.IsTag():
		// Delete tag only if it is package-specific.
		name := createFinalRefName(oldGit.path, oldGit.revision)
		if rn != name {
			return fmt.Errorf("cannot delete package tagged with a tag that is not specific to the package: %s", rn)
		}

		// Delete the tag
		refsDelete[name] = true

	case strings.HasPrefix(rn.String(), refDraftPrefix), strings.HasPrefix(rn.String(), refProposedPrefix):
		// PackageRevision is proposed or draft; delete the branch directly.
		refsDelete[rn] = true

	case rn.IsBranch():
		// Delete package from the branch
		commitHash, err := r.createPackageDeleteCommit(ctx, rn, oldGit)
		if err != nil {
			return err
		}

		// Update the reference
		// TODO: consider collecting all updates and applying them all at once.
		setReference(r.repo.Storer, rn, commitHash)

		refsPush = append(
			refsPush,
			config.RefSpec(fmt.Sprintf("%s:%s", commitHash, rn.String())),
		)

	default:
		return fmt.Errorf("cannot delete package with the ref name %s", rn)
	}

	// Update references
	if err := r.pushAndCleanupRefs(ctx, refsPush, refsDelete); err != nil {
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
	rev := ref.Name().String()
	switch {
	case strings.HasPrefix(rev, refTagsPrefix):
		revision = strings.TrimPrefix(rev, refTagsPrefix)
	case strings.HasPrefix(rev, refHeadsPrefix):
		revision = strings.TrimPrefix(rev, refHeadsPrefix)
	default:
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
			return found(dir, tree.Hash, e.Hash)
		}
	}

	for _, e := range tree.Entries {
		if e.Mode != filemode.Dir {
			continue
		}

		dirTree, err := r.TreeObject(e.Hash)
		if err != nil {
			return err
		}

		discoverPackagesInTree(r, dirTree, path.Join(dir, e.Name), found)
	}
	return nil
}

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

	var packageTree plumbing.Hash

	switch dirTree, err := tree.Tree(name); err {
	case nil:
		packageTree = dirTree.Hash
		if kptfileEntry, err := dirTree.FindEntry("Kptfile"); err == nil {
			if !kptfileEntry.Mode.IsRegular() {
				return nil, fmt.Errorf("found Kptfile which is not a regular file: %s", kptfileEntry.Mode)
			}
		}

	case object.ErrDirectoryNotFound:
	case object.ErrEntryNotFound:
		// ok; empty package

	default:
		return nil, fmt.Errorf("error when looking for package in the repository: %w", err)
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
	refName := draft.Name().String()
	var suffix string
	switch {
	case strings.HasPrefix(refName, refDraftPrefix):
		suffix = strings.TrimPrefix(refName, refDraftPrefix)
	case strings.HasPrefix(refName, refProposedPrefix):
		suffix = strings.TrimPrefix(refName, refProposedPrefix)
	default:
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
	name := tag.Name().String()
	if !strings.HasPrefix(name, refTagsPrefix) {
		return nil, fmt.Errorf("invalid tag ref name: %q", name)
	}
	name = strings.TrimPrefix(name, refTagsPrefix)
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

func createDraftRefName(name, revision string) plumbing.ReferenceName {
	refName := fmt.Sprintf("refs/heads/drafts/%s/%s", name, revision)
	return plumbing.ReferenceName(refName)
}

func createProposedRefName(name, revision string) plumbing.ReferenceName {
	return plumbing.ReferenceName(refProposedPrefix + name + "/" + revision)
}

func createFinalRefName(name, revision string) plumbing.ReferenceName {
	return plumbing.ReferenceName(refTagsPrefix + name + "/" + revision)
}

func createApprovedRefName(name, revision string) plumbing.ReferenceName {
	// TODO: use createFinalRefName
	refName := fmt.Sprintf("refs/heads/%s/%s", name, revision)
	return plumbing.ReferenceName(refName)
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

func (r *gitRepository) update(ctx context.Context) error {
	auth, err := r.getAuthMethod(ctx)
	if err != nil {
		return err
	}

	// Fetch
	switch err := r.repo.Fetch(&git.FetchOptions{
		RemoteName: originName,
		Auth:       auth,
		Tags:       git.AllTags,
	}); err {
	case nil: // OK
	case git.NoErrAlreadyUpToDate:
	case transport.ErrEmptyRemoteRepository:

	default:
		return fmt.Errorf("cannot fetch repository %s/%s: %w", r.namespace, r.name, err)
	}

	// Create tracking branches for remotes
	refs, err := r.repo.References()
	if err != nil {
		return fmt.Errorf("cannot identify repository remote references in %s/%s: %w", r.namespace, r.name, err)
	}

	// Collect remote and local branches. Both maps are indexed by the local reference name.
	// remote references (refs/remotes/origin/...) are transformed to `refs/heads/...` to have
	// both maps use matching keys.
	remoteBranches := map[string]*plumbing.Reference{}
	localBranches := map[string]*plumbing.Reference{}
	for {
		ref, err := refs.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			klog.Warningf("Skipping reference during iteration on error: %v", err)
			continue
		}

		name := ref.Name().String()
		switch {
		case strings.HasPrefix(name, refRemoteBranchPrefix):
			branch := strings.TrimPrefix(name, refRemoteBranchPrefix)
			remoteBranches[refHeadsPrefix+branch] = ref

		case strings.HasPrefix(name, refHeadsPrefix):
			localBranches[name] = ref
		}
	}

	// TODO: Explore use of automatic branch tracking to do this.
	// There may be risks involved such as automated rebase which may lead to incorrect results on the package.
	// One possibility is to replay package history ... something to consider in the future.
	for name, remoteRef := range remoteBranches {
		localRef, ok := localBranches[name]

		if !ok {
			localRef = plumbing.NewHashReference(plumbing.ReferenceName(name), remoteRef.Hash())
			// local branch doesn't exist. create it
			if err := r.repo.Storer.SetReference(localRef); err != nil {
				return fmt.Errorf("failed creating reference %q: %v", localRef, err)
			}
		} else if remoteRef.Hash() != localRef.Hash() {
			remoteCommit, err := r.repo.CommitObject(remoteRef.Hash())
			if err != nil {
				return fmt.Errorf("failed to resolve remote reference %s: %w", remoteRef, err)
			}
			localCommit, err := r.repo.CommitObject(localRef.Hash())
			if err != nil {
				klog.Warningf("Overwriting unresolvable local reference %s: %v", localRef, err)
				new := plumbing.NewHashReference(localRef.Name(), remoteCommit.Hash)
				if err := r.repo.Storer.SetReference(new); err != nil {
					return fmt.Errorf("failed to set local reference: %s to %s", new.Name(), new.Hash())
				}
				continue
			}

			// If local commit is ancestor of remote, fast-forward local commit to match
			ancestor, err := localCommit.IsAncestor(remoteCommit)
			if err != nil {
				klog.Warningf("Failed to determine whether %s is ancestor of %s: %v", localRef, remoteRef, err)
			}

			// TODO: Better conflict resolution policy
			if !ancestor {
				klog.Warningf("Refusing to fast-forward %s which is not an ancestor of %s", localRef, remoteRef)
				continue
			}

			klog.Infof("Fast-forwarding local branch %s to %s", localRef, remoteRef)
			new := plumbing.NewHashReference(localRef.Name(), remoteCommit.Hash)
			if err := r.repo.Storer.SetReference(new); err != nil {
				klog.Errorf("Failed to fast-forward %s to %s, localRef, remoteRef")
			}
		}
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

	// Fetch the branch
	switch err := repo.Fetch(&git.FetchOptions{
		RemoteName: originName,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("+%s:%s", branch, branch))},
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
	ch, err := newCommitHelper(repo.Storer, commit.Hash, packagePath, zero)
	if err != nil {
		return zero, fmt.Errorf("failed to initialize commit of package %q to %q: %w", packagePath, ref, err)
	}

	message := fmt.Sprintf("Delete %s", packagePath)
	commitHash, _, err := ch.commit(message, packagePath)
	if err != nil {
		return zero, fmt.Errorf("failed to commit package %q to %q: %w", packagePath, ref, err)
	}
	return commitHash, nil
}

func (r *gitRepository) pushAndCleanupRefs(ctx context.Context, updates []config.RefSpec, cleanup map[plumbing.ReferenceName]bool) error {
	auth, err := r.getAuthMethod(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain git credentials: %w", err)
	}

	repo := r.repo

	if len(updates) > 0 {
		switch err := repo.Push(&git.PushOptions{
			RemoteName: originName,
			RefSpecs:   updates,
			Auth:       auth,
		}); err {
		case nil, git.NoErrAlreadyUpToDate:
			// ok
		default:
			return fmt.Errorf("failed to push to git: %w", err)
		}

	}

	deleteRemotes := []config.RefSpec{}
	// Cleanup local and remote branches
	for rn := range cleanup {
		switch err := repo.Storer.RemoveReference(rn); err {
		case nil, plumbing.ErrReferenceNotFound:
			// These are OK.

		default:
			return err
		}

		deleteRemotes = append(deleteRemotes, config.RefSpec(fmt.Sprintf(":%s", rn.String())))
	}

	// TODO: Combine this Push with the one above? Will need to reconcile the temporary
	// use of `force` in the previous push where draft package contents are updated.
	if len(deleteRemotes) > 0 {
		switch err := repo.Push(&git.PushOptions{
			RemoteName: originName,
			RefSpecs:   deleteRemotes,
			Auth:       auth,
		}); err {
		case nil, git.NoErrAlreadyUpToDate:
			// ok
		default:
			klog.Errorf("Failed to clean up remote branches: %v", err)
		}
	}

	return nil
}
