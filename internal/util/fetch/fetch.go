// Copyright 2019 Google LLC
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

package fetch

import (
	"context"
	"fmt"
	"io/fs"
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
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

// Command takes the upstream information in the Kptfile at the path for the
// provided package, and fetches the package referenced if it isn't already
// there.
type Command struct {
	Pkg *pkg.Pkg
}

// Run runs the Command.
func (c Command) Run(ctx context.Context) error {
	const op errors.Op = "fetch.Run"
	kf, err := c.Pkg.Kptfile()
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("no Kptfile found"))
	}

	if err := c.validate(kf); err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	g := kf.Upstream.Git
	repoSpec := &git.RepoSpec{
		OrgRepo: g.Repo,
		Path:    g.Directory,
		Ref:     g.Ref,
	}
	err = cloneAndCopy(ctx, repoSpec, c.Pkg.UniquePath.String())
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	return nil
}

// validate makes sure the Kptfile has the necessary information to fetch
// the package.
func (c Command) validate(kf *kptfilev1.KptFile) error {
	const op errors.Op = "validate"
	if kf.Upstream == nil {
		return errors.E(op, errors.MissingParam, fmt.Errorf("kptfile doesn't contain upstream information"))
	}

	if kf.Upstream.Git == nil {
		return errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream doesn't have git information"))
	}

	g := kf.Upstream.Git
	if len(g.Repo) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify repo"))
	}
	if len(g.Ref) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify ref"))
	}
	if len(g.Directory) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify directory"))
	}
	return nil
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
		if _, isPathError := err.(*fs.PathError); isPathError {
			return errors.E(op, "Missing path at package", errors.Repo(r.RepoRef()), err)
		}
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

	// We need to create a temp directory where we can copy the content of the repo.
	// During update, we need to checkout multiple versions of the same repo, so
	// we can't do merges directly from the cache.
	repoSpec.Dir, err = ioutil.TempDir("", "kpt-get-")
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
	}
	repoSpec.Commit = commit

	// Copy the content of the repo into the temp directory.
	// TODO: See if we can avoid copying everything in the repo if the
	// repoSpec.Path property is a subdirectory of the repo.
	err = copyutil.CopyDir(dir, repoSpec.Dir)
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error copying package: %w", err))
	}

	// Verify that if a Kptfile exists in the package, it contains the correct
	// version of the Kptfile.
	pkgPath := filepath.Join(dir, repoSpec.Path)
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
