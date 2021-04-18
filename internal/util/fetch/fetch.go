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
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
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
func (c Command) validate(kf *kptfilev1alpha2.KptFile) error {
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
	p := printer.FromContextOrDie(ctx)
	p.Printf("cloning %s@%s\n", r.OrgRepo, r.Ref)
	err := ClonerUsingGitExec(ctx, r)
	if err != nil {
		return errors.E(op, errors.Git, types.UniquePath(dest), err)
	}
	defer os.RemoveAll(r.Dir)

	sourcePath := filepath.Join(r.Dir, r.Path)
	p.Printf("copying %q to %s\n", r.Path, dest)
	if err := pkgutil.CopyPackageWithSubpackages(sourcePath, dest); err != nil {
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

	upGitRunner, err := gitutil.NewGitUpstreamRepo(ctx, repoSpec.CloneSpec())
	if err != nil {
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	packageRef := path.Join(strings.TrimLeft(repoSpec.Path, "/"), repoSpec.Ref)
	if _, found := upGitRunner.ResolveTag(packageRef); found {
		repoSpec.Ref = packageRef
	}

	dir, err := upGitRunner.GetRepo(ctx, []string{repoSpec.Ref})
	if err != nil {
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	gitRunner, err := gitutil.NewLocalGitRunner(dir)
	if err != nil {
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	commit, found := upGitRunner.ResolveRef(repoSpec.Ref)
	if !found {
		commit = repoSpec.Ref
	}

	_, err = gitRunner.Run(ctx, "reset", "--hard", commit)
	if err != nil {
		return errors.E(op, errors.Git, errors.Repo(repoSpec.CloneSpec()), err)
	}

	repoSpec.Dir, err = ioutil.TempDir("", "kpt-get-")
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
	}
	repoSpec.Commit = commit

	err = copyutil.CopyDir(dir, repoSpec.Dir)
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error copying package: %w", err))
	}
	return nil
}
