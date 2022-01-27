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
	gogit "github.com/go-git/go-git/v5"
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
	refDraftPrefix = "refs/heads/drafts/"
)

type GitRepository interface {
	repository.Repository
	GetPackage(ref, path string) (repository.PackageRevision, kptfilev1.GitLock, error)
}

func OpenRepository(name, namespace string, spec *configapi.GitRepository, authOpts repository.AuthOptions, root string) (GitRepository, error) {
	replace := strings.NewReplacer("/", "-", ":", "-")
	dir := filepath.Join(root, replace.Replace(spec.Repo))
	auth := createAuth(authOpts)

	var repo *gogit.Repository

	if fi, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		opts := gogit.CloneOptions{
			URL:        spec.Repo,
			Auth:       auth,
			NoCheckout: true,
		}
		isBare := true
		r, err := gogit.PlainClone(dir, isBare, &opts)
		if err != nil {
			return nil, err
		}
		repo = r
	} else if !fi.IsDir() {
		// Internal error - corrupted cache.
		return nil, fmt.Errorf("cannot clone git repository: %q", spec.Repo)
	} else {
		r, err := gogit.PlainOpen(dir)
		if err != nil {
			return nil, err
		}

		remotes, err := r.Remotes()
		if err != nil {
			return nil, fmt.Errorf("cannot list remotes in %q", spec.Repo)
		}

		found := false
	outer:
		for _, remote := range remotes {
			cfg := remote.Config()
			for _, url := range cfg.URLs {
				if url == spec.Repo {
					found = true
					break outer
				}
			}
		}
		if !found {
			// TODO: add remote?
			return nil, fmt.Errorf("cannot clone git repository: %q", spec.Repo)
		}
		repo = r
	}

	return &gitRepository{
		name:      name,
		namespace: namespace,
		repo:      repo,
		auth:      auth,
	}, nil
}

func createAuth(auth repository.AuthOptions) transport.AuthMethod {
	if auth == nil {
		return nil
	}
	if username, uok := auth["username"]; uok {
		if token, tok := auth["token"]; tok {
			return &http.BasicAuth{
				Username: string(username),
				Password: string(token),
			}
		}
	}
	return nil
}

type gitRepository struct {
	name      string
	namespace string
	repo      *gogit.Repository
	auth      transport.AuthMethod
}

func (r *gitRepository) ListPackageRevisions(ctx context.Context) ([]repository.PackageRevision, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, err
	}

	var main *plumbing.Reference
	var drafts []repository.PackageRevision

	for {
		ref, err := refs.Next()
		if err == io.EOF {
			break
		}

		name := ref.Name()
		if name == refMain {
			main = ref
			continue
		} else if strings.HasPrefix(name.String(), refDraftPrefix) {
			draft, err := r.loadDraft(ref)
			if err != nil {
				return nil, fmt.Errorf("failed to load package draft %q: %w", name.String(), err)
			}
			drafts = append(drafts, draft)
		}
	}

	var result []repository.PackageRevision
	if main != nil {
		// TODO: analyze tags too
		result, err = r.discoverFinalizedPackages(main)
		if err != nil {
			return nil, err
		}
	}

	result = append(result, drafts...)

	return result, nil
}

func (r *gitRepository) CreatePackageRevision(ctx context.Context, obj *v1alpha1.PackageRevision) (repository.PackageDraft, error) {
	main, err := r.repo.Reference(refMain, true)
	if err != nil {
		return nil, err
	}
	ref := createDraftRefName(obj.Spec.PackageName, obj.Spec.Revision)
	head := plumbing.NewHashReference(ref, main.Hash())
	if err := r.repo.Storer.SetReference(head); err != nil {
		return nil, err
	}

	return &gitPackageDraft{
		gitPackageRevision: gitPackageRevision{
			parent:   r,
			path:     obj.Spec.PackageName,
			revision: obj.Spec.Revision,
			updated:  time.Now(),
			draft:    head,
			sha:      main.Hash(),
		},
	}, nil
}

func (r *gitRepository) UpdatePackage(ctx context.Context, old repository.PackageRevision) (repository.PackageDraft, error) {
	oldGitPackage, ok := old.(*gitPackageRevision)
	if !ok {
		return nil, fmt.Errorf("cannot update non-git package %T", old)
	}

	refName := createDraftRefName(oldGitPackage.path, oldGitPackage.revision)
	head, err := r.repo.Reference(refName, true)
	if err != nil {
		return nil, fmt.Errorf("cannot find draft package branch %q: %w", refName, err)
	}

	rev, err := r.loadDraft(head)
	if err != nil {
		return nil, fmt.Errorf("cannot load draft package: %w", err)
	}

	return &gitPackageDraft{
		gitPackageRevision: gitPackageRevision{
			parent:   r,
			path:     oldGitPackage.path,
			revision: oldGitPackage.revision,
			updated:  rev.updated,
			draft:    rev.draft,
			tree:     rev.tree,
			sha:      rev.sha,
		},
	}, nil
}

func (r *gitRepository) DeletePackageRevision(ctx context.Context, old repository.PackageRevision) error {
	return fmt.Errorf("gitRepository::DeletePackageRevision not implemented")
}

func (r *gitRepository) GetPackage(version, path string) (repository.PackageRevision, kptfilev1.GitLock, error) {
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
	var hash plumbing.Hash

	// Versions map to git tags in one of two ways:
	//
	// * directly (tag=version)- but then this means that all packages in the repo must be versioned together.
	// * prefixed (tag=<packageDir/<version>) - solving the co-versioning problem.
	//
	// We have to check both forms when looking up a version.
	refNames := []string{
		path + "/" + version,
		version,
	}
	for _, ref := range refNames {
		if resolved, err := git.ResolveRevision(plumbing.Revision(ref)); err != nil {
			if errors.Is(err, plumbing.ErrReferenceNotFound) {
				continue
			}
			return nil, lock, fmt.Errorf("error resolving git reference %q: %w", ref, err)
		} else {
			hash = *resolved
			break
		}
	}

	if hash.IsZero() {
		return nil, lock, fmt.Errorf("cannot find git reference (tried %v)", refNames)
	}

	commit, err := git.CommitObject(hash)
	if err != nil {
		return nil, lock, fmt.Errorf("cannot resolve git reference %s (hash: %s) to commit: %w", version, hash, err)
	}
	lock.Commit = commit.Hash.String()

	ctree, err := commit.Tree()
	if err != nil {
		return nil, lock, fmt.Errorf("cannot resolve git reference %s (hash %s) to tree: %w", version, hash, err)
	}
	te, err := ctree.FindEntry(path)
	if err != nil {
		return nil, lock, fmt.Errorf("cannot find package %s@%s: %w", path, version, err)
	}
	if te.Mode != filemode.Dir {
		return nil, lock, fmt.Errorf("path %s@%s is not a directory", path, version)
	}

	return &gitPackageRevision{
		parent:   r,
		path:     path,
		revision: version,
		updated:  commit.Author.When,
		tree:     te.Hash,
		sha:      hash,
	}, lock, nil
}

func (r *gitRepository) discoverFinalizedPackages(main *plumbing.Reference) ([]repository.PackageRevision, error) {
	git := r.repo
	commit, err := git.CommitObject(main.Hash())
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var result []repository.PackageRevision
	if err := discoverPackagesInTree(git, tree, "", func(dir string, tree, kptfile plumbing.Hash) error {
		result = append(result, &gitPackageRevision{
			parent:   r,
			path:     dir,
			revision: "",
			updated:  commit.Author.When, // TODO: this is inaccurate, pointing at the last commit in 'main'; rather we need time of the package's last commit/tag
			draft:    nil,
			tree:     tree,
			sha:      main.Hash(),
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

type foundPackageCallback func(dir string, tree, kptfile plumbing.Hash) error

func discoverPackagesInTree(r *gogit.Repository, tree *object.Tree, dir string, found foundPackageCallback) error {
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

func (r *gitRepository) loadDraft(draft *plumbing.Reference) (*gitPackageRevision, error) {
	name, revision, err := parseDraftName(draft)
	if err != nil {
		return nil, err
	}

	commit, err := r.repo.CommitObject(draft.Hash())
	if err != nil {
		return nil, fmt.Errorf("cannot resolve draft branch to commit (corrupted repository?): %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve package commit to tree (corrupted repository?): %w", err)
	}

	var packageTree plumbing.Hash

	if dirTree, err := tree.Tree(name); err != nil {
		if err != object.ErrEntryNotFound {
			return nil, fmt.Errorf("draft package is not a directory (corrupted repository?): %w", err)
		}
	} else {
		packageTree = dirTree.Hash
		if kptfileEntry, err := dirTree.FindEntry("Kptfile"); err == nil {
			if !kptfileEntry.Mode.IsRegular() {
				return nil, fmt.Errorf("found Kptfile which is not a regular file: %s", kptfileEntry.Mode)
			}
		}
	}

	return &gitPackageRevision{
		parent:   r,
		path:     name,
		revision: revision,
		updated:  commit.Author.When,
		draft:    draft,
		tree:     packageTree,
		sha:      draft.Hash(),
	}, nil
}

func parseDraftName(draft *plumbing.Reference) (name, revision string, err error) {
	draftBranch := draft.Name().String()
	if !strings.HasPrefix(draftBranch, refDraftPrefix) {
		return "", "", fmt.Errorf("invalid draft ref name: %q; expected prefix %q", draftBranch, refDraftPrefix)
	}

	suffix := draftBranch[len(refDraftPrefix):]
	revIndex := strings.LastIndex(suffix, "/")
	if revIndex <= 0 {
		return "", "", fmt.Errorf("invalid draft ref name; missing revision suffix: %q", draftBranch)
	}
	name, revision = suffix[:revIndex], suffix[revIndex+1:]
	return name, revision, nil
}

func createDraftRefName(name, revision string) plumbing.ReferenceName {
	refName := fmt.Sprintf("refs/heads/drafts/%s/%s", name, revision)
	return plumbing.ReferenceName(refName)
}
