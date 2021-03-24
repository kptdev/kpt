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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
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
func (c Command) Run() error {
	const op errors.Op = "fetch.Run"
	kf, err := c.Pkg.Kptfile()
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
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
	err = cloneAndCopy(repoSpec, c.Pkg.UniquePath.String())
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	return nil
}

// validate makes sure the Kptfile has the necessary information to fetch
// the package.
func (c Command) validate(kf *kptfilev1alpha2.KptFile) error {
	const op errors.Op = "fetch.validate"
	if kf.Upstream == nil {
		return errors.E(op, c.Pkg.UniquePath, errors.MissingParam,
			"kptfile doesn't contain upstream information")
	}

	if kf.Upstream.Git == nil {
		return errors.E(op, c.Pkg.UniquePath, errors.MissingParam,
			"kptfile upstream doesn't have git information")
	}

	g := kf.Upstream.Git
	if len(g.Repo) == 0 {
		return errors.E(op, c.Pkg.UniquePath, errors.MissingParam,
			"must specify repo")
	}
	if len(g.Ref) == 0 {
		return errors.E(op, c.Pkg.UniquePath, errors.MissingParam,
			"must specify ref")
	}
	if len(g.Directory) == 0 {
		return errors.E(op, c.Pkg.UniquePath, errors.MissingParam,
			"must specify directory")
	}
	return nil
}

// cloneAndCopy fetches the provided repo and copies the content into the
// directory specified by dest. The provided name is set as `metadata.name`
// of the Kptfile of the package.
func cloneAndCopy(r *git.RepoSpec, dest string) error {
	const op errors.Op = "fetch.cloneAndCopy"
	if err := ClonerUsingGitExec(r); err != nil {
		return errors.E(op, err)
	}
	defer os.RemoveAll(r.Dir)

	if err := pkgutil.CopyPackageWithSubpackages(r.AbsPath(), dest); err != nil {
		return errors.E(op, err)
	}

	if err := UpsertKptfile(dest, r); err != nil {
		return errors.E(op, "failed to update Kptfile")
	}
	return nil
}

// ClonerUsingGitExec uses a local git install, as opposed
// to say, some remote API, to obtain a local clone of
// a remote repo. It looks for tags with the directory as a prefix to allow
// for versioning multiple kpt packages in a single repo independently. It
// relies on the private clonerUsingGitExec function to try fetching different
// refs.
func ClonerUsingGitExec(repoSpec *git.RepoSpec) error {
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
	err = clonerUsingGitExec(repoSpec)
	if err != nil && originalRef != repoSpec.Ref {
		repoSpec.Ref = originalRef
		err = clonerUsingGitExec(repoSpec)
	}

	if err != nil {
		if strings.HasPrefix(repoSpec.Path, "blob/") {
			return errors.E(op, errors.Git,
				fmt.Errorf("failed to clone git repo containing /blob/, "+
					"you may need to remove /blob/%s from the url: %w", defaultRef, err))
		}
		return errors.E(op, errors.Git, err)
	}

	return nil
}

// clonerUsingGitExec is the implementation for cloning a repo from git into
// a local temp directory. This is used by the public ClonerUsingGitExec
// function to allow trying multiple different refs.
func clonerUsingGitExec(repoSpec *git.RepoSpec) error {
	const op errors.Op = "fetch.clonerUsingGitExec"
	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return errors.E(op, errors.Git, "no 'git' program on path", err)
	}

	repoSpec.Dir, err = ioutil.TempDir("", "kpt-get-")
	if err != nil {
		return errors.E(op, errors.Internal, err)
	}
	cmd := exec.Command(gitProgram, "init", repoSpec.Dir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		return errors.E(op, errors.Git, &execError{
			Msg:    "failed to initialize empty git repo",
			Err:    err,
			Output: out.String(),
		})
	}

	cmd = exec.Command(gitProgram, "remote", "add", "origin", repoSpec.CloneSpec())
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = repoSpec.Dir
	err = cmd.Run()
	if err != nil {
		return errors.E(op, errors.Git, &execError{
			Msg:    fmt.Sprintf("failed to set git remote %s", repoSpec.CloneSpec()),
			Err:    err,
			Output: out.String(),
		})
	}
	if repoSpec.Ref == "" {
		repoSpec.Ref, err = gitutil.DefaultRef(repoSpec.Dir)
		if err != nil {
			return errors.E(op, errors.Git,
				fmt.Errorf("failed to look up default ref for git clone"))
		}
	}

	err = func() error {
		cmd = exec.Command(gitProgram, "fetch", "origin", "--depth=1", repoSpec.Ref)
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		err = cmd.Run()
		if err != nil {
			return errors.E(op, errors.Git, &execError{
				Msg: fmt.Sprintf("failed to fetch %q, "+
					"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", repoSpec.Ref),
				Err:    err,
				Output: out.String(),
			})
		}
		cmd = exec.Command(gitProgram, "reset", "--hard", "FETCH_HEAD")
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		err = cmd.Run()
		if err != nil {
			return errors.E(op, errors.Git, &execError{
				Msg:    fmt.Sprintf("failed to hard reset empty repository to %q", repoSpec.Ref),
				Err:    err,
				Output: out.String(),
			})
		}
		return nil
	}()
	if err != nil {
		cmd = exec.Command(gitProgram, "fetch", "origin")
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		if err = cmd.Run(); err != nil {
			return errors.E(op, errors.Git, &execError{
				Msg: fmt.Sprintf("failed to fetch origin, " +
					"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials"),
				Err:    err,
				Output: out.String(),
			})
		}
		cmd = exec.Command(gitProgram, "reset", "--hard", repoSpec.Ref)
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		if err = cmd.Run(); err != nil {
			return errors.E(op, errors.Git, &execError{
				Msg: fmt.Sprintf("failed to hard reset empty repository to %q, "+
					"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", repoSpec.Ref),
				Err:    err,
				Output: out.String(),
			})
		}
	}

	cmd = exec.Command(gitProgram, "submodule", "update", "--init", "--recursive")
	cmd.Stdout = &out
	cmd.Dir = repoSpec.Dir
	err = cmd.Run()
	if err != nil {
		return errors.E(op, errors.Git, &execError{
			Msg: fmt.Sprintf("failed to fetch submodules for %q, "+
				"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", repoSpec.Ref),
			Err:    err,
			Output: out.String(),
		})
	}

	return nil
}

type execError struct {
	Msg    string
	Output string
	Err    error
}

func (e *execError) Error() string {
	b := new(strings.Builder)
	b.WriteString(e.Msg)
	b.WriteString(": ")
	b.WriteString(e.Err.Error())
	// TODO: Consider if we should print the stdout/stderr output here as well.
	return b.String()
}

// UpsertKptfile populates the KptFile values, merging any cloned KptFile and the
// cloneFrom values.
func UpsertKptfile(path string, spec *git.RepoSpec) error {
	// read KptFile cloned with the package if it exists
	kpgfile, err := kptfileutil.ReadFile(path)
	if err != nil {
		return err
	}

	// find the git commit sha that we cloned the package at so we can write it to the KptFile
	commit, err := git.LookupCommit(spec.AbsPath())
	if err != nil {
		return err
	}

	// populate the cloneFrom values so we know where the package came from
	kpgfile.UpstreamLock = &kptfilev1alpha2.UpstreamLock{
		Type: kptfilev1alpha2.GitOrigin,
		GitLock: &kptfilev1alpha2.GitLock{
			Repo:      spec.OrgRepo,
			Directory: spec.Path,
			Ref:       spec.Ref,
			Commit:    commit,
		},
	}
	return kptfileutil.WriteFile(path, kpgfile)
}
