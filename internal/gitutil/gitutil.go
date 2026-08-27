// Copyright 2019,2026 The kpt Authors
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

package gitutil

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/kptdev/kpt/pkg/lib/errors"
	"github.com/kptdev/kpt/pkg/printer"
)

// RepoCacheDirEnv is the name of the environment variable that controls the cache directory
// for remote repos.  Defaults to UserHomeDir/.kpt/repos if unspecified.
const RepoCacheDirEnv = "KPT_CACHE_DIR"

type GitArgKind string

const (
	GitArgRef     GitArgKind = "reference"
	GitArgCommit  GitArgKind = "commit"
	GitArgRepoURI GitArgKind = "repo URI"
)

// validateGitArg guards against git argument injection. Values such as refs,
// tags, refspecs, commit SHAs and repo URIs are frequently attacker-controlled
// (e.g. read from the upstream block of a remote sub-package's Kptfile) and are
// passed to git as positional arguments. Git never accepts a ref, refspec,
// commit, or URI that begins with a dash, so any such value would instead be
// interpreted as a command-line option (for example `--output=<file>` for
// `git show`, or `--upload-pack=<cmd>` for `git fetch`). Rejecting values that
// begin with a dash prevents these option-injection attacks without breaking
// any legitimate input.
func validateGitArg(kind GitArgKind, value string) error {
	const op errors.Op = "gitutil.validateGitArg"
	if strings.HasPrefix(value, "-") {
		return errors.E(op, errors.InvalidParam, fmt.Errorf(
			"invalid git %s %q: must not begin with '-'", kind, value))
	}
	return nil
}

// NewLocalGitRunner returns a new GitLocalRunner for a local package.
func NewLocalGitRunner(pkg string) (*GitLocalRunner, error) {
	const op errors.Op = "gitutil.NewLocalGitRunner"
	p, err := exec.LookPath("git")
	if err != nil {
		return nil, errors.E(op, errors.Git, &GitExecError{
			Type: GitExecutableNotFound,
			Err:  err,
		})
	}

	return &GitLocalRunner{
		gitPath: p,
		Dir:     pkg,
		Debug:   false,
	}, nil
}

// GitLocalRunner runs git commands in a local git repo.
type GitLocalRunner struct {
	// Path to the git executable.
	gitPath string

	// Dir is the directory the commands are run in.
	Dir string

	// Debug enables output of debug information to stderr.
	Debug bool
}

type RunResult struct {
	Stdout string
	Stderr string
}

// Run runs a git command.
// Omit the 'git' part of the command.
// The first return value contains the output to Stdout and Stderr when
// running the command.
func (g *GitLocalRunner) Run(ctx context.Context, command string, args ...string) (RunResult, error) {
	return g.run(ctx, false, command, args...)
}

// RunVerbose runs a git command.
// Omit the 'git' part of the command.
// The first return value contains the output to Stdout and Stderr when
// running the command.
func (g *GitLocalRunner) RunVerbose(ctx context.Context, command string, args ...string) (RunResult, error) {
	return g.run(ctx, true, command, args...)
}

// run runs a git command.
// Omit the 'git' part of the command.
// The first return value contains the output to Stdout and Stderr when
// running the command.
func (g *GitLocalRunner) run(ctx context.Context, verbose bool, command string, args ...string) (RunResult, error) {
	const op errors.Op = "gitutil.run"

	fullArgs := append([]string{command}, args...)
	cmd := exec.CommandContext(ctx, g.gitPath, fullArgs...)
	cmd.Dir = g.Dir
	// Disable git prompting the user for credentials.
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0")
	pr := printer.FromContextOrDie(ctx)
	cmdStdout := &bytes.Buffer{}
	cmdStderr := &bytes.Buffer{}
	if verbose {
		cmd.Stdout = io.MultiWriter(cmdStdout, pr.OutStream())
		cmd.Stderr = io.MultiWriter(cmdStderr, pr.ErrStream())
	} else {
		cmd.Stdout = cmdStdout
		cmd.Stderr = cmdStderr
	}

	if g.Debug {
		_, _ = fmt.Fprintf(os.Stderr, "[git -C %s %s]\n", g.Dir, strings.Join(fullArgs, " "))
	}
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	if g.Debug {
		_, _ = fmt.Fprintf(os.Stderr, "duration: %v\n", duration)
	}
	if err != nil {
		return RunResult{}, errors.E(op, errors.Git, &GitExecError{
			Type:    determineErrorType(cmdStderr.String()),
			Args:    args,
			Command: command,
			Err:     err,
			StdOut:  cmdStdout.String(),
			StdErr:  cmdStderr.String(),
		})
	}
	return RunResult{
		Stdout: cmdStdout.String(),
		Stderr: cmdStderr.String(),
	}, nil
}

// NewGitUpstreamRepo returns a new GitUpstreamRepo for an upstream package.
func NewGitUpstreamRepo(ctx context.Context, uri string) (GitUpstreamRepo, error) {
	const op errors.Op = "gitutil.NewGitUpstreamRepo"
	if err := validateGitArg(GitArgRepoURI, uri); err != nil {
		return nil, errors.E(op, errors.Repo(uri), err)
	}
	g := &gitUpstreamRepoBroker{
		uri:         uri,
		fetchedRefs: map[string]bool{},
	}
	if err := g.updateRefs(ctx); err != nil {
		return nil, errors.E(op, errors.Repo(uri), err)
	}
	return g, nil
}

type GitUpstreamRepo interface {
	Uri() string
	Heads() []string
	Tags() []string
	GetFetchedRefs() []string
	GetRepo(ctx context.Context, refs []string) (string, error)
	GetDefaultBranch(ctx context.Context) (string, error)
	ResolveBranch(branch string) (string, bool)
	ResolveTag(tag string) (string, bool)
	ResolveRef(ref string) string
}

// gitUpstreamRepoBroker runs git commands in a local git repo.
type gitUpstreamRepoBroker struct {
	uri string

	// commitByHead contains all head refs in the upstream repo
	commitByHead map[string]string

	// commitByTag contains all tag refs in the upstream repo
	commitByTag map[string]string

	// fetchedRefs keeps track of the commits already fetched from remote. It is
	// keyed by the resolved commit SHA (not the ref name) so that a branch or
	// tag that has moved to a new commit is fetched again rather than skipped.
	fetchedRefs map[string]bool
}

func (gur *gitUpstreamRepoBroker) Uri() string {
	return gur.uri
}

func (gur *gitUpstreamRepoBroker) Heads() []string {
	heads := slices.Sorted(maps.Keys(gur.commitByHead))
	if heads == nil {
		return []string{}
	}
	return heads
}

func (gur *gitUpstreamRepoBroker) Tags() []string {
	tags := slices.Sorted(maps.Keys(gur.commitByTag))
	if tags == nil {
		return []string{}
	}
	return tags
}

func (gur *gitUpstreamRepoBroker) GetFetchedRefs() []string {
	refs := make([]string, 0, len(gur.fetchedRefs))
	for ref := range gur.fetchedRefs {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

// updateRefs fetches all refs from the upstream git repo, parses the results
// and caches all refs and the commit they reference. Note that this doesn't
// download any objects, only refs.
func (gur *gitUpstreamRepoBroker) updateRefs(ctx context.Context) error {
	const op errors.Op = "gitutil.updateRefs"
	repoCacheDir, err := gur.cacheRepo(ctx, nil)
	if err != nil {
		return errors.E(op, errors.Repo(gur.uri), err)
	}

	gitRunner, err := NewLocalGitRunner(repoCacheDir)
	if err != nil {
		return errors.E(op, errors.Repo(gur.uri), err)
	}

	rr, err := gitRunner.Run(ctx, "ls-remote", "--heads", "--tags", "--refs", "origin")
	if err != nil {
		AmendGitExecError(err, func(e *GitExecError) {
			e.Repo = gur.uri
		})
		// TODO: This should only fail if we can't connect to the repo. We should
		// consider exposing the error message from git to the user here.
		return errors.E(op, errors.Repo(gur.uri), err)
	}

	commitByHead := make(map[string]string)
	commitByTag := make(map[string]string)

	re := regexp.MustCompile(`^([a-z0-9]+)\s+refs/(heads|tags)/(.+)$`)
	scanner := bufio.NewScanner(bytes.NewBufferString(rr.Stdout))
	for scanner.Scan() {
		txt := scanner.Text()
		res := re.FindStringSubmatch(txt)
		if len(res) == 0 {
			continue
		}
		commit := res[1]
		kind := res[2]
		name := res[3]
		switch kind {
		case "heads":
			if err := validateGitArg(GitArgRef, name); err != nil {
				return errors.E(op, errors.Repo(gur.uri), err)
			}
			if err := validateGitArg(GitArgCommit, commit); err != nil {
				return errors.E(op, errors.Repo(gur.uri), err)
			}
			commitByHead[name] = commit
		case "tags":
			if err := validateGitArg(GitArgRef, name); err != nil {
				return errors.E(op, errors.Repo(gur.uri), err)
			}
			if err := validateGitArg(GitArgCommit, commit); err != nil {
				return errors.E(op, errors.Repo(gur.uri), err)
			}
			commitByTag[name] = commit
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.E(op, errors.Repo(gur.uri), errors.Git,
			fmt.Errorf("error parsing response from git: %w", err))
	}
	gur.commitByHead = commitByHead
	gur.commitByTag = commitByTag
	return nil
}

// GetRepo fetches all the provided refs and the objects. It will fetch it
// to the cache repo and returns the path to the local git clone in the cache
// directory.
func (gur *gitUpstreamRepoBroker) GetRepo(ctx context.Context, refs []string) (string, error) {
	const op errors.Op = "gitutil.GetRepo"
	for _, ref := range refs {
		if err := validateGitArg(GitArgRef, ref); err != nil {
			return "", errors.E(op, errors.Repo(gur.uri), err)
		}
	}
	// Refresh our view of the upstream refs so that a branch (or tag) that has
	// moved to a new commit since this broker was created resolves to its
	// current commit. Together with keying fetchedRefs on the resolved commit
	// in cacheRepo, this ensures a moved ref is fetched again instead of being
	// served the stale commit cached at construction time.
	if err := gur.updateRefs(ctx); err != nil {
		return "", errors.E(op, errors.Repo(gur.uri), err)
	}
	dir, err := gur.cacheRepo(ctx, refs)
	if err != nil {
		return "", errors.E(op, errors.Repo(gur.uri), err)
	}
	return dir, nil
}

// GetDefaultBranch returns the name of the branch pointed to by the
// HEAD symref. This is the default branch of the repository.
func (gur *gitUpstreamRepoBroker) GetDefaultBranch(ctx context.Context) (string, error) {
	const op errors.Op = "gitutil.GetDefaultBranch"
	cacheRepo, err := gur.cacheRepo(ctx, nil)
	if err != nil {
		return "", errors.E(op, errors.Repo(gur.uri), err)
	}

	gitRunner, err := NewLocalGitRunner(cacheRepo)
	if err != nil {
		return "", errors.E(op, errors.Repo(gur.uri), err)
	}

	rr, err := gitRunner.Run(ctx, "ls-remote", "--symref", "origin", "HEAD")
	if err != nil {
		AmendGitExecError(err, func(e *GitExecError) {
			e.Repo = gur.uri
		})
		return "", errors.E(op, errors.Repo(gur.uri), err)
	}
	if rr.Stdout == "" {
		return "", errors.E(op, errors.Repo(gur.uri),
			fmt.Errorf("unable to detect default branch in repo"))
	}

	re := regexp.MustCompile(`ref: refs/heads/([^\s/]+)\s*HEAD`)
	match := re.FindStringSubmatch(rr.Stdout)
	if len(match) != 2 {
		return "", errors.E(op, errors.Repo(gur.uri), errors.Git,
			fmt.Errorf("unexpected response from git when determining default branch: %s", rr.Stdout))
	}
	return match[1], nil
}

// ResolveBranch resolves the branch to a commit SHA. This happens based on the
// cached information about refs in the upstream repo. If the branch doesn't exist
// in the upstream repo, the last return value will be false.
func (gur *gitUpstreamRepoBroker) ResolveBranch(branch string) (string, bool) {
	branch = strings.TrimPrefix(branch, "refs/heads/")
	commit, ok := gur.commitByHead[branch]
	return commit, ok
}

// ResolveTag resolves the tag to a commit SHA. This happens based on the
// cached information about refs in the upstream repo. If the tag doesn't exist
// in the upstream repo, the last return value will be false.
func (gur *gitUpstreamRepoBroker) ResolveTag(tag string) (string, bool) {
	tag = strings.TrimPrefix(tag, "refs/tags/")
	commit, ok := gur.commitByTag[tag]
	return commit, ok
}

// ResolveRef resolves the ref (either tag or branch) to a commit SHA. If the
// ref doesn't exist in the upstream repo, the last return value will be false.
func (gur *gitUpstreamRepoBroker) ResolveRef(ref string) string {
	if commit, found := gur.ResolveBranch(ref); found {
		return commit
	}
	if commit, found := gur.ResolveTag(ref); found {
		return commit
	}

	return ref
}

// getRepoDir returns the cache directory name for a remote repo
// This takes the md5 hash of the repo uri and then base32 (or hex for Windows to shorten dir)
// encodes it to make sure it doesn't contain characters that isn't legal in directory names.
func (gur *gitUpstreamRepoBroker) getRepoDir(uri string) string {
	if runtime.GOOS == "windows" {
		var hash = md5.Sum([]byte(uri))
		return strings.ToLower(hex.EncodeToString(hash[:]))
	}
	sum := md5.Sum([]byte(uri))
	return strings.ToLower(base32.StdEncoding.EncodeToString(sum[:]))
}

// getRepoCacheDir
func (gur *gitUpstreamRepoBroker) getRepoCacheDir() (string, error) {
	const op errors.Op = "gitutil.getRepoCacheDir"
	var err error
	dir := os.Getenv(RepoCacheDirEnv)
	if dir != "" {
		return dir, nil
	}

	// cache location unspecified, use UserHomeDir/.kpt/repos
	dir, err = os.UserHomeDir()
	if err != nil {
		return "", errors.E(op, errors.IO, fmt.Errorf(
			"error looking up user home dir: %w", err))
	}
	return filepath.Join(dir, ".kpt", "repos"), nil
}

// cacheRepo fetches the remote repo, and fetches the provided refs.
func (gur *gitUpstreamRepoBroker) cacheRepo(ctx context.Context, requiredRefs []string) (string, error) {
	const op errors.Op = "gitutil.cacheRepo"
	// preventing argument injection.
	for _, ref := range requiredRefs {
		if err := validateGitArg(GitArgRef, ref); err != nil {
			return "", errors.E(op, errors.Repo(gur.uri), err)
		}
	}
	kptCacheDir, err := gur.getRepoCacheDir()
	if err != nil {
		return "", errors.E(op, err)
	}
	if err := os.MkdirAll(kptCacheDir, 0700); err != nil {
		return "", errors.E(op, errors.IO, fmt.Errorf(
			"error creating cache directory for repo: %w", err))
	}

	// create the repo directory if it doesn't exist yet
	gitRunner, err := NewLocalGitRunner(kptCacheDir)
	if err != nil {
		return "", errors.E(op, errors.Repo(gur.uri), err)
	}
	uriSha := gur.getRepoDir(gur.uri)
	repoCacheDir := filepath.Join(kptCacheDir, uriSha)
	if _, err := os.Stat(repoCacheDir); os.IsNotExist(err) {
		if _, err := gitRunner.Run(ctx, "init", uriSha); err != nil {
			AmendGitExecError(err, func(e *GitExecError) {
				e.Repo = gur.uri
			})
			return "", errors.E(op, errors.Git, fmt.Errorf("error running `git init`: %w", err))
		}
		gitRunner.Dir = repoCacheDir
		if _, err = gitRunner.Run(ctx, "remote", "add", "origin", gur.uri); err != nil {
			AmendGitExecError(err, func(e *GitExecError) {
				e.Repo = gur.uri
			})
			return "", errors.E(op, errors.Git, fmt.Errorf("error adding origin remote: %w", err))
		}
	} else {
		gitRunner.Dir = repoCacheDir
	}

	for i := range requiredRefs {
		requiredRef := requiredRefs[i]
		// Check if we can verify the ref. This will output a full commit sha if
		// either the ref (short commit, tag, branch) can be resolved to a full
		// commit sha, or if the provided ref is already a valid full commit sha (note
		// that this will happen even if the commit doesn't exist in the local repo).
		// We ignore the error here since an error just means the ref didn't exist,
		// which we detect by checking the output to stdout.
		rr, _ := gitRunner.Run(ctx, "rev-parse", "--verify", "-q", requiredRef)
		// If the output is the same as the ref, then the ref was already a full
		// commit sha.
		validFullSha := requiredRef == strings.TrimSpace(rr.Stdout)
		resolvedCommit := gur.ResolveRef(requiredRef)
		resolved := resolvedCommit != requiredRef
		// Use the resolved commit SHA as the cache key rather than the ref name.
		// Branches and tags are mutable and may point at a different commit than
		// they did on a previous fetch; keying on the immutable commit ensures a
		// moved ref is fetched again instead of being skipped as already-fetched.
		cacheKey := requiredRef
		if resolved {
			cacheKey = resolvedCommit
		}
		_, fetched := gur.fetchedRefs[cacheKey]
		switch {
		case fetched:
			// skip refetching if previously fetched
			break
		case resolved || validFullSha:
			// If the ref references a branch or a tag, or is a valid commit
			// sha and has not already been fetched, we can fetch just a single commit.
			if _, err := gitRunner.RunVerbose(ctx, "fetch", "origin", "--depth=1", requiredRef); err != nil {
				AmendGitExecError(err, func(e *GitExecError) {
					e.Repo = gur.uri
					e.Command = "fetch"
					e.Ref = requiredRef
				})
				return "", errors.E(op, errors.Git, fmt.Errorf(
					"error running `git fetch` for ref %q: %w", requiredRef, err))
			}
			gur.fetchedRefs[cacheKey] = true
		default:
			// In other situations (like a short commit sha), we have to do
			// a full fetch from the remote.
			if _, err := gitRunner.RunVerbose(ctx, "fetch", "origin"); err != nil {
				AmendGitExecError(err, func(e *GitExecError) {
					e.Repo = gur.uri
					e.Command = "fetch"
				})
				return "", errors.E(op, errors.Git, fmt.Errorf(
					"error running `git fetch` for origin: %w", err))
			}
			if _, err = gitRunner.Run(ctx, "show", requiredRef); err != nil {
				AmendGitExecError(err, func(e *GitExecError) {
					e.Repo = gur.uri
					e.Ref = requiredRef
				})
				return "", errors.E(op, errors.Git, fmt.Errorf(
					"error verifying results from fetch: %w", err))
			}
			gur.fetchedRefs[cacheKey] = true
			return repoCacheDir, nil
		}
	}

	return repoCacheDir, nil
}
