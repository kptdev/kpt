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

package location

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	"github.com/GoogleContainerTools/kpt/pkg/location/extensions"
)

type Git struct {
	// Repo is the git repository the package.
	// e.g. 'https://github.com/kubernetes/examples.git'
	Repo string

	// Directory is the sub directory of the git repository.
	// e.g. 'staging/cockroachdb'
	Directory string

	// Ref can be a Git branch, tag, or a commit SHA-1.
	Ref string

	// original is the value before parsing, it is returned
	// by String() to improve round-trip accuracy
	original string
}

var _ Reference = Git{}
var _ extensions.IdentifierGetter = Git{}
var _ extensions.DefaultDirectoryNameGetter = Git{}
var _ extensions.DefaultIdentifierGetter = Git{}

type GitLock struct {
	Git

	// Commit is the SHA-1 for the last fetch of the package.
	// This is set by kpt for bookkeeping purposes.
	Commit string
}

var _ Reference = GitLock{}
var _ ReferenceLock = GitLock{}
var _ extensions.IdentifierGetter = GitLock{}
var _ extensions.LockGetter = GitLock{}
var _ extensions.DefaultDirectoryNameGetter = GitLock{}
var _ extensions.DefaultIdentifierGetter = GitLock{}

func NewGit(location string, opts ...Option) (Git, error) {
	return newGit(location, makeOptions(opts...))
}

func newGit(location string, opt options) (Git, error) {
	// args[1] is "" for commands that do not require an output path
	gitTarget, err := parse.GitParseArgs(opt.ctx, []string{location, ""})
	var zero parse.GitTarget
	if err != nil || gitTarget == zero {
		return Git{}, err
	}

	dir := gitTarget.Directory
	if strings.HasPrefix(dir, "/") {
		dir, err = filepath.Rel("/", gitTarget.Directory)
		if err != nil {
			return Git{}, err
		}
	}

	return Git{
		Repo:      gitTarget.Repo,
		Directory: dir,
		Ref:       gitTarget.Ref,
		original:  location,
	}, nil
}

func parseGit(location string, opt options) (Reference, error) {
	git, err := newGit(location, opt)
	var zero Git
	if err == nil && git != zero {
		return git, nil
	}

	return nil, err
}

// String implements location.Reference
func (ref Git) String() string {
	if ref.original != "" {
		return ref.original
	}
	return gitString(ref.Repo, ref.Directory, ref.Ref)
}

// String implements location.ReferenceLock
func (ref GitLock) String() string {
	return gitString(ref.Repo, ref.Directory, ref.Commit)
}

func gitString(repo, dir, identifier string) string {
	if dir != "" && dir != "/" && dir != "." {
		return fmt.Sprintf("%s/%s@%s", repo, dir, identifier)
	}
	return fmt.Sprintf("%s@%s", repo, identifier)
}

// Type implements location.Reference
func (ref Git) Type() string {
	return "git"
}

// Validate implements location.Reference
func (ref Git) Validate() error {
	const op errors.Op = "git.Validate"
	if len(ref.Repo) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify repo"))
	}
	if len(ref.Ref) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify ref"))
	}
	if len(ref.Directory) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify directory"))
	}
	return nil
}

// GetDefaultDirectoryName is called from location.DefaultDirectoryName
func (ref Git) GetDefaultDirectoryName() (string, bool) {
	repo := ref.Repo
	repo = strings.TrimSuffix(repo, "/")
	repo = strings.TrimSuffix(repo, ".git")
	return path.Base(path.Join(path.Clean(repo), path.Clean(ref.Directory))), true
}

func (ref Git) GetDefaultIdentifier(ctx context.Context) (string, error) {
	gur, err := gitutil.NewGitUpstreamRepo(ctx, ref.Repo)
	if err != nil {
		return "", err
	}
	b, err := gur.GetDefaultBranch(ctx)
	if err != nil {
		return "", err
	}
	return b, nil
}

func (ref Git) GetIdentifier() (string, bool) {
	return ref.Ref, true
}

// SetIdentifier is called from mutate.Identifier
func (ref Git) SetIdentifier(name string) (Reference, error) {
	return Git{
		Repo:      ref.Repo,
		Directory: ref.Directory,
		Ref:       name,
	}, nil
}

func (ref GitLock) GetLock() (string, bool) {
	return ref.Commit, true
}

// SetLock is called from mutate.Lock
func (ref Git) SetLock(lock string) (ReferenceLock, error) {
	return GitLock{
		Git:    ref,
		Commit: lock,
	}, nil
}
