package location

import (
	"fmt"
	"path/filepath"
	"strings"

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

type GitLock struct {
	Git

	// Commit is the SHA-1 for the last fetch of the package.
	// This is set by kpt for bookkeeping purposes.
	Commit string
}

var _ ReferenceLock = GitLock{}

func NewGit(location string, opts ...Option) (Git, error) {
	opt := makeOptions(opts...)

	// args[1] is "" for commands that do not require an output path
	gitTarget, err := parse.GitParseArgs(opt.ctx, []string{location, ""})
	if err != nil {
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

func (ref Git) String() string {
	return fmt.Sprintf("type:git repo:%q ref:%q directory:%q", ref.Repo, ref.Ref, ref.Directory)
}

func (ref GitLock) String() string {
	return fmt.Sprintf("%v commit:%q", ref.Git, ref.Commit)
}

func (ref Git) SetIdentifier(name string) (Reference, error) {
	return Git{
		Repo:      ref.Repo,
		Directory: ref.Directory,
		Ref:       name,
	}, nil
}

func (ref Git) SetLock(lock string) (ReferenceLock, error) {
	return GitLock{
		Git:    ref,
		Commit: lock,
	}, nil
}
