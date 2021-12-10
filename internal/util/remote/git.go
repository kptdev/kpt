// Copyright 2021 Google LLC
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

package remote

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/otiai10/copy"
)

type gitUpstream struct {
	git     *v1.Git
	gitLock *v1.GitLock
	origin  *v1.GitLock
}

var _ Fetcher = &gitUpstream{}

func NewGitUpstream(git *v1.Git) Fetcher {
	return &gitUpstream{
		git: git,
	}
}

func NewGitOrigin(git *v1.Git) Fetcher {
	return &gitUpstream{
		origin: &v1.GitLock{
			Repo:      git.Repo,
			Directory: git.Directory,
			Ref:       git.Ref,
		},
	}
}

func (u *gitUpstream) String() string {
	return fmt.Sprintf("%s@%s", u.git.Repo, u.git.Ref)
}

func (u *gitUpstream) LockedString() string {
	return fmt.Sprintf("%s@%s", u.gitLock.Repo, u.gitLock.Ref)
}


func (u *gitUpstream) OriginString() string {
	return fmt.Sprintf("%s@%s", u.origin.Repo, u.origin.Ref)
}


func (u *gitUpstream) Validate() error {
	const op errors.Op = "upstream.Validate"
	g := u.git
	if g != nil {
		if len(g.Repo) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify repo"))
		}
		if len(g.Ref) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify ref"))
		}
		if len(g.Directory) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify directory"))
		}
	}
	return nil
}

func (u *gitUpstream) BuildUpstream() *v1.Upstream {
	repoDir := u.git.Directory
	if !strings.HasSuffix(repoDir, "file://") {
		repoDir = filepath.Join(path.Split(repoDir))
	}
	u.git.Directory = repoDir

	return &v1.Upstream{
		Type: v1.GitOrigin,
		Git:  u.git,
	}
}

func (u *gitUpstream) BuildUpstreamLock(digest string) *v1.UpstreamLock {
	u.gitLock = &v1.GitLock{
		Repo:      u.git.Repo,
		Directory: u.git.Directory,
		Ref:       u.git.Ref,
		Commit:    digest,
	}
	return &v1.UpstreamLock{
		Type: v1.GitOrigin,
		Git:  u.gitLock,
	}
}

func (u *gitUpstream) BuildOrigin(digest string) *v1.Origin {
	return &v1.Origin{
		Type: v1.GitOrigin,
		Git:  &v1.GitLock{
			Repo:      u.origin.Repo,
			Directory: u.origin.Directory,
			Ref:       u.origin.Ref,
			Commit:    digest,
		},
	}
}

func (u *gitUpstream) FetchUpstream(ctx context.Context, dest string) (string, string, error) {
	repoSpec := &git.RepoSpec{
		OrgRepo: u.git.Repo,
		Path:    u.git.Directory,
		Ref:     u.git.Ref,
		Dir:     dest,
	}
	if err := ClonerUsingGitExec(ctx, repoSpec); err != nil {
		return "", "", err
	}
	return path.Join(repoSpec.Dir, repoSpec.Path), repoSpec.Commit, nil
}

func (u *gitUpstream) FetchUpstreamLock(ctx context.Context, dest string) (string, error) {
	repoSpec := &git.RepoSpec{
		OrgRepo: u.gitLock.Repo,
		Path:    u.gitLock.Directory,
		Ref:     u.gitLock.Commit,
		Dir:     dest,
	}
	if err := ClonerUsingGitExec(ctx, repoSpec); err != nil {
		return "", err
	}
	return path.Join(repoSpec.Dir, repoSpec.Path), nil
}

func (u *gitUpstream) FetchOrigin(ctx context.Context, dest string) (string, string, error) {
	repoSpec := &git.RepoSpec{
		OrgRepo: u.origin.Repo,
		Path:    u.origin.Directory,
		Ref:     u.origin.Ref,
	}
	if err := ClonerUsingGitExec(ctx, repoSpec); err != nil {
		return "", "", err
	}
	defer os.RemoveAll(repoSpec.Dir)
	if err := pkgutil.CopyPackage(repoSpec.AbsPath(), dest, true, pkg.All); err != nil {
		return "", "", err
	}

	return dest, repoSpec.Commit, nil
}

func (u *gitUpstream) CloneUpstream(ctx context.Context, dest string) error {
	repoSpec := &git.RepoSpec{
		OrgRepo: u.git.Repo,
		Path:    u.git.Directory,
		Ref:     u.git.Ref,
	}
	return cloneAndCopy(ctx, repoSpec, dest)
}

func (u *gitUpstream) PushOrigin(ctx context.Context, dest string, kptfile *kptfilev1.KptFile) (digest string, err error) {
	return "", fmt.Errorf("git push not implemented")
}

func (u *gitUpstream) Ref() (string, error) {
	return u.git.Ref, nil
}

func (u *gitUpstream) SetRef(ref string) error {
	u.git.Ref = ref
	return nil
}

func (u *gitUpstream) OriginRef() (string, error) {
	return u.origin.Ref, nil
}

func (u *gitUpstream) SetOriginRef(ref string) error {
	u.origin.Ref = ref
	return nil
}

// shouldUpdateSubPkgRef checks if subpkg ref should be updated.
// This is true if pkg has the same upstream repo, upstream directory is within or equal to root pkg directory and original root pkg ref matches the subpkg ref.
func (u *gitUpstream) ShouldUpdateSubPkgRef(rootUpstream Fetcher, originalRootKfRef string) bool {
	root, ok := rootUpstream.(*gitUpstream)
	return ok &&
		u.git.Repo == root.git.Repo &&
		u.git.Ref == originalRootKfRef &&
		strings.HasPrefix(path.Clean(u.git.Directory), path.Clean(root.git.Directory))
}

// cloneAndCopy fetches the provided repo and copies the content into the
// directory specified by dest. The provided name is set as `metadata.name`
// of the Kptfile of the package.
func cloneAndCopy(ctx context.Context, r *git.RepoSpec, dest string) error {
	const op errors.Op = "fetch.cloneAndCopy"
	pr := printer.FromContextOrDie(ctx)

	err := ClonerUsingGitExec(ctx, r)
	if err != nil {
		return errors.E(op, errors.Git, types.UniquePath(dest), err)
	}
	defer os.RemoveAll(r.Dir)

	sourcePath := filepath.Join(r.Dir, r.Path)
	pr.Printf("Adding package %q.\n", strings.TrimPrefix(r.Path, "/"))
	if err := pkgutil.CopyPackage(sourcePath, dest, true, pkg.All); err != nil {
		return errors.E(op, types.UniquePath(dest), err)
	}

	if err := kptfileutil.UpdateKptfileWithoutOrigin(dest, sourcePath, false); err != nil {
		return errors.E(op, types.UniquePath(dest), err)
	}

	if err := kptfileutil.UpdateUpstreamLockFromGit(dest, r); err != nil {
		return errors.E(op, errors.Git, types.UniquePath(dest), err)
	}
	return nil
}

// ClonerUsingGitExec uses a local git install, as opposed
// to say, some remote API, to obtain a local clone of
// a remote repo. It looks for tags with the directory as a prefix to allow
// for versioning multiple kpt packages in a single repo independently. It
// relies on the private clonerUsingGitExec function to try fetching different
// refs.
func ClonerUsingGitExec(ctx context.Context, repoSpec *git.RepoSpec) error {
	const op errors.Op = "fetch.ClonerUsingGitExec"

	// Create a local representation of the upstream repo. This will initialize
	// the cache for the specified repo uri if it isn't already there. It also
	// fetches and caches all tag and branch refs from the upstream repo.
	upstreamRepo, err := gitutil.NewGitUpstreamRepo(ctx, repoSpec.CloneSpec())
	if err != nil {
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	// Check if we have a ref in the upstream that matches the package-specific
	// reference. If we do, we use that reference.
	ps := strings.Split(repoSpec.Path, "/")
	for len(ps) != 0 {
		p := path.Join(ps...)
		packageRef := path.Join(strings.TrimLeft(p, "/"), repoSpec.Ref)
		if _, found := upstreamRepo.ResolveTag(packageRef); found {
			repoSpec.Ref = packageRef
			break
		}
		ps = ps[:len(ps)-1]
	}

	// Pull the required ref into the repo git cache.
	dir, err := upstreamRepo.GetRepo(ctx, []string{repoSpec.Ref})
	if err != nil {
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	gitRunner, err := gitutil.NewLocalGitRunner(dir)
	if err != nil {
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	// Find the commit SHA for the ref that was just fetched. We need the SHA
	// rather than the ref to be able to do a hard reset of the cache repo.
	commit, found := upstreamRepo.ResolveRef(repoSpec.Ref)
	if !found {
		commit = repoSpec.Ref
	}

	// Reset the local repo to the commit we need. Doing a hard reset instead of
	// a checkout means we don't create any local branches so we don't need to
	// worry about fast-forwarding them with changes from upstream. It also makes
	// sure that any changes in the local worktree are cleaned out.
	_, err = gitRunner.Run(ctx, "reset", "--hard", commit)
	if err != nil {
		gitutil.AmendGitExecError(err, func(e *gitutil.GitExecError) {
			e.Repo = repoSpec.CloneSpec()
			e.Ref = commit
		})
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	if repoSpec.Dir == "" {
		// We need to create a temp directory where we can copy the content of the repo.
		// During update, we need to checkout multiple versions of the same repo, so
		// we can't do merges directly from the cache.
		repoSpec.Dir, err = ioutil.TempDir("", "kpt-get-")
		if err != nil {
			return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
		}
	}
	repoSpec.Commit = commit

	pkgPath := filepath.Join(dir, repoSpec.Path)
	// Verify that the requested path exists in the repo.
	_, err = os.Stat(pkgPath)
	if os.IsNotExist(err) {
		return errors.E(op,
			errors.Internal,
			err,
			fmt.Errorf("path %q does not exist in repo %q", repoSpec.Path, repoSpec.OrgRepo))
	}

	// Copy the content of the pkg into the temp directory.
	// Note that we skip the content outside the package directory.
	err = copyDir(ctx, pkgPath, repoSpec.AbsPath())
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error copying package: %w", err))
	}

	// Verify that if a Kptfile exists in the package, it contains the correct
	// version of the Kptfile.
	_, err = pkg.ReadKptfile(pkgPath)
	if err != nil {
		// A Kptfile isn't required, so it is fine if there is no Kptfile.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		// If the error is of type KptfileError, we replace it with a
		// RemoteKptfileError. This allows us to provide information about the
		// git source of the Kptfile instead of the path to some random
		// temporary directory.
		var kfError *pkg.KptfileError
		if errors.As(err, &kfError) {
			return &pkg.RemoteKptfileError{
				RepoSpec: repoSpec,
				Err:      kfError.Err,
			}
		}
	}
	return nil
}

// copyDir copies a src directory to a dst directory.
// copyDir skips copying the .git directory from the src and ignores symlinks.
func copyDir(ctx context.Context, srcDir string, dstDir string) error {
	pr := printer.FromContextOrDie(ctx)
	opts := copy.Options{
		Skip: func(src string) (bool, error) {
			return strings.HasSuffix(src, ".git"), nil
		},
		OnSymlink: func(src string) copy.SymlinkAction {
			// try to print relative path of symlink
			// if we can, else absolute path which is not
			// pretty because it contains path to temporary repo dir
			displayPath, err := filepath.Rel(srcDir, src)
			if err != nil {
				displayPath = src
			}
			pr.Printf("[Warn] Ignoring symlink %q \n", displayPath)
			return copy.Skip
		},
	}
	return copy.Copy(srcDir, dstDir, opts)
}
