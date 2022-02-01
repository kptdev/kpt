package location

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
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
}

var _ Reference = Git{}
var _ DirectoryNameDefaulter = Git{}

type GitLock struct {
	Git

	// Commit is the SHA-1 for the last fetch of the package.
	// This is set by kpt for bookkeeping purposes.
	Commit string
}

var _ ReferenceLock = GitLock{}

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
	}, nil
}

func parseGit(location string, opt options) (Reference, error) {
	git, gitErr := newGit(location, opt)
	var zero Git
	if gitErr == nil && git != zero {
		return git, nil
	}

	// TODO - figure out which gitErr must be returned, and which simply mean "it's not a git path"

	return nil, nil
}

// String implements location.Reference
func (ref Git) String() string {
	return fmt.Sprintf("type:git repo:%q ref:%q directory:%q", ref.Repo, ref.Ref, ref.Directory)
}

// String implements location.ReferenceLock
func (ref GitLock) String() string {
	return fmt.Sprintf("%v commit:%q", ref.Git, ref.Commit)
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

// SetIdentifier is called from mutate.Identifier
func (ref Git) SetIdentifier(name string) (Reference, error) {
	return Git{
		Repo:      ref.Repo,
		Directory: ref.Directory,
		Ref:       name,
	}, nil
}

// SetLock is called from mutate.Lock
func (ref Git) SetLock(lock string) (ReferenceLock, error) {
	return GitLock{
		Git:    ref,
		Commit: lock,
	}, nil
}
