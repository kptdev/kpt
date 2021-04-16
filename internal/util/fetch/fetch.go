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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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
	if err := ClonerUsingGitExec(ctx, r); err != nil {
		return errors.E(op, errors.Git, types.UniquePath(dest), err)
	}
	defer os.RemoveAll(r.Dir)

	p.Printf("copying %q to %s\n", r.Path, dest)
	if err := pkgutil.CopyPackageWithSubpackages(r.AbsPath(), dest); err != nil {
		return errors.E(op, types.UniquePath(dest), err)
	}

	if err := kptfileutil.UpdateKptfileWithoutOrigin(dest, r.AbsPath(), false); err != nil {
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
	// look for a tag with the directory as a prefix for versioning
	// subdirectories independently
	originalRef := repoSpec.Ref
	if repoSpec.Path != "" && !strings.Contains(repoSpec.Ref, "refs") {
		// join the directory with the Ref (stripping the preceding '/' if it exists)
		repoSpec.Ref = path.Join(strings.TrimLeft(repoSpec.Path, "/"), repoSpec.Ref)
	}

	defaultRef, err := gitutil.DefaultRef(repoSpec.OrgRepo)
	if err != nil {
		return errors.E(op, errors.Git, err)
	}

	// clone the repo to a tmp directory.
	// delete the tmp directory later.
	err = clonerUsingGitExec(ctx, repoSpec)
	if err != nil && originalRef != repoSpec.Ref {
		repoSpec.Ref = originalRef
		err = clonerUsingGitExec(ctx, repoSpec)
	}

	if err != nil {
		if strings.HasPrefix(repoSpec.Path, "blob/") {
			p := printer.FromContextOrDie(ctx)
			p.Printf("git repo contains /blob/, you may need to remove /blob/%s", defaultRef)
			return errors.E(op, errors.Git, err)
		}
		return errors.E(op, errors.Git, err)
	}

	return nil
}

// clonerUsingGitExec is the implementation for cloning a repo from git into
// a local temp directory. This is used by the public ClonerUsingGitExec
// function to allow trying multiple different refs.
func clonerUsingGitExec(ctx context.Context, repoSpec *git.RepoSpec) error {
	const op errors.Op = "fetch.clonerUsingGitExec"
	var err error
	repoSpec.Dir, err = ioutil.TempDir("", "kpt-get-")
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
	}
	err = runGitExec(ctx, repoSpec.Dir, "init", repoSpec.Dir)
	if err != nil {
		return errors.E(op, errors.Git, fmt.Errorf("trouble initializing empty git repo in %s: %w",
			repoSpec.Dir, err))
	}

	err = runGitExec(ctx, repoSpec.Dir, "remote", "add", "origin", repoSpec.CloneSpec())
	if err != nil {
		return errors.E(op, errors.Git, fmt.Errorf("error adding remote %s: %w", repoSpec.CloneSpec(), err))
	}
	if repoSpec.Ref == "" {
		repoSpec.Ref, err = gitutil.DefaultRef(repoSpec.Dir)
		if err != nil {
			return errors.E(op, errors.Git, fmt.Errorf("error looking up default branch for repo: %w", err))
		}
	}

	err = func() error {
		err = runGitExec(ctx, repoSpec.Dir, "fetch", "origin", "--depth=1", repoSpec.Ref)
		if err != nil {
			return errors.E(op, errors.Git, fmt.Errorf("trouble fetching %s, "+
				"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", repoSpec.Ref, err))
		}

		err = runGitExec(ctx, repoSpec.Dir, "reset", "--hard", "FETCH_HEAD")
		if err != nil {
			return errors.E(op, errors.Git,
				fmt.Errorf("trouble hard resetting empty repository to %s: %w", repoSpec.Ref, err))
		}
		return nil
	}()
	if err != nil {
		err := runGitExec(ctx, repoSpec.Dir, "fetch", "origin")
		if err != nil {
			return errors.E(op, errors.Git, fmt.Errorf("trouble fetching origin, "+
				"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", err))
		}

		err = runGitExec(ctx, repoSpec.Dir, "reset", "--hard", repoSpec.Ref)
		if err != nil {
			return errors.E(op, errors.Git, fmt.Errorf("trouble hard resetting empty repository to %s, "+
				"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", repoSpec.Ref, err))
		}
	}

	err = runGitExec(ctx, repoSpec.Dir, "submodule", "update", "--init", "--recursive")
	if err != nil {
		return errors.E(op, errors.Git, fmt.Errorf("trouble fetching submodules for %s, "+
			"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials: %w", repoSpec.Ref, err))
	}

	return nil
}

func runGitExec(ctx context.Context, dir string, args ...string) error {
	const op errors.Op = "fetch.runGitExec"
	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return errors.E(op, errors.Git,
			fmt.Errorf("no 'git' program on path: %w", err))
	}

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	cmd := exec.CommandContext(ctx, gitProgram, args...)
	cmd.Dir = dir
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	if err != nil {
		return &GitExecError{
			Args:   args,
			Err:    err,
			StdErr: errBuf.String(),
			StdOut: outBuf.String(),
		}
	}
	return nil
}

type GitExecError struct {
	Args   []string
	Err    error
	StdErr string
	StdOut string
}

func (e *GitExecError) Error() string {
	b := new(strings.Builder)
	b.WriteString(e.Err.Error())
	b.WriteString(": ")
	b.WriteString(e.StdErr)
	return b.String()
}
