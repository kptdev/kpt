// Copyright 2019 The kpt Authors
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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
)

// RepoCacheDirEnv is the name of the environment variable that controls the cache directory
// for remote repos.  Defaults to UserHomeDir/.kpt/repos if unspecified.
const RepoCacheDirEnv = "KPT_CACHE_DIR"

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

type NewGitUpstreamRepoOption func(*GitUpstreamRepo)

func WithFetchedRefs(a map[string]bool) NewGitUpstreamRepoOption {
	return func(g *GitUpstreamRepo) {
		g.fetchedRefs = a
	}
}

// NewGitUpstreamRepo returns a new GitUpstreamRepo for an upstream package.
func NewGitUpstreamRepo(ctx context.Context, uri string, opts ...NewGitUpstreamRepoOption) (*GitUpstreamRepo, error) {
	const op errors.Op = "gitutil.NewGitUpstreamRepo"
	g := &GitUpstreamRepo{
		URI: uri,
	}
	for _, opt := range opts {
		opt(g)
	}
	if g.fetchedRefs == nil {
		g.fetchedRefs = map[string]bool{}
	}
	if err := g.updateRefs(ctx); err != nil {
		return nil, errors.E(op, errors.Repo(uri), err)
	}
	return g, nil
}

// GitUpstreamRepo runs git commands in a local git repo.
type GitUpstreamRepo struct {
	URI string

	// Heads contains all head refs in the upstream repo as well as the
	// each of the are referencing.
	Heads map[string]string

	// Tags contains all tag refs in the upstream repo as well as the
	// each of the are referencing.
	Tags map[string]string

	// fetchedRefs keeps track of refs already fetched from remote
	fetchedRefs map[string]bool
}

func (gur *GitUpstreamRepo) GetFetchedRefs() []string {
	fetchedRefs := make([]string, 0, len(gur.fetchedRefs))
	for ref := range gur.fetchedRefs {
		fetchedRefs = append(fetchedRefs, ref)
	}
	return fetchedRefs
}

// updateRefs fetches all refs from the upstream git repo, parses the results
// and caches all refs and the commit they reference. Not that this doesn't
// download any objects, only refs.
func (gur *GitUpstreamRepo) updateRefs(ctx context.Context) error {
	const op errors.Op = "gitutil.updateRefs"
	repoCacheDir, err := gur.cacheRepo(ctx, gur.URI, []string{}, []string{})
	if err != nil {
		return errors.E(op, errors.Repo(gur.URI), err)
	}

	gitRunner, err := NewLocalGitRunner(repoCacheDir)
	if err != nil {
		return errors.E(op, errors.Repo(gur.URI), err)
	}

	rr, err := gitRunner.Run(ctx, "ls-remote", "--heads", "--tags", "--refs", "origin")
	if err != nil {
		AmendGitExecError(err, func(e *GitExecError) {
			e.Repo = gur.URI
		})
		// TODO: This should only fail if we can't connect to the repo. We should
		// consider exposing the error message from git to the user here.
		return errors.E(op, errors.Repo(gur.URI), err)
	}

	heads := make(map[string]string)
	tags := make(map[string]string)

	re := regexp.MustCompile(`^([a-z0-9]+)\s+refs/(heads|tags)/(.+)$`)
	scanner := bufio.NewScanner(bytes.NewBufferString(rr.Stdout))
	for scanner.Scan() {
		txt := scanner.Text()
		res := re.FindStringSubmatch(txt)
		if len(res) == 0 {
			continue
		}
		switch res[2] {
		case "heads":
			heads[res[3]] = res[1]
		case "tags":
			tags[res[3]] = res[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.E(op, errors.Repo(gur.URI), errors.Git,
			fmt.Errorf("error parsing response from git: %w", err))
	}
	gur.Heads = heads
	gur.Tags = tags
	return nil
}

// GetRepo fetches all the provided refs and the objects. It will fetch it
// to the cache repo and returns the path to the local git clone in the cache
// directory.
func (gur *GitUpstreamRepo) GetRepo(ctx context.Context, refs []string) (string, error) {
	const op errors.Op = "gitutil.GetRepo"
	dir, err := gur.cacheRepo(ctx, gur.URI, refs, []string{})
	if err != nil {
		return "", errors.E(op, errors.Repo(gur.URI), err)
	}
	return dir, nil
}

// GetDefaultBranch returns the name of the branch pointed to by the
// HEAD symref. This is the default branch of the repository.
func (gur *GitUpstreamRepo) GetDefaultBranch(ctx context.Context) (string, error) {
	const op errors.Op = "gitutil.GetDefaultBranch"
	cacheRepo, err := gur.cacheRepo(ctx, gur.URI, []string{}, []string{})
	if err != nil {
		return "", errors.E(op, errors.Repo(gur.URI), err)
	}

	gitRunner, err := NewLocalGitRunner(cacheRepo)
	if err != nil {
		return "", errors.E(op, errors.Repo(gur.URI), err)
	}

	rr, err := gitRunner.Run(ctx, "ls-remote", "--symref", "origin", "HEAD")
	if err != nil {
		AmendGitExecError(err, func(e *GitExecError) {
			e.Repo = gur.URI
		})
		return "", errors.E(op, errors.Repo(gur.URI), err)
	}
	if rr.Stdout == "" {
		return "", errors.E(op, errors.Repo(gur.URI),
			fmt.Errorf("unable to detect default branch in repo"))
	}

	re := regexp.MustCompile(`ref: refs/heads/([^\s/]+)\s*HEAD`)
	match := re.FindStringSubmatch(rr.Stdout)
	if len(match) != 2 {
		return "", errors.E(op, errors.Repo(gur.URI), errors.Git,
			fmt.Errorf("unexpected response from git when determining default branch: %s", rr.Stdout))
	}
	return match[1], nil
}

// ResolveBranch resolves the branch to a commit SHA. This happens based on the
// cached information about refs in the upstream repo. If the branch doesn't exist
// in the upstream repo, the last return value will be false.
func (gur *GitUpstreamRepo) ResolveBranch(branch string) (string, bool) {
	branch = strings.TrimPrefix(branch, "refs/heads/")
	for head, commit := range gur.Heads {
		if head == branch {
			return commit, true
		}
	}
	return "", false
}

// ResolveTag resolves the tag to a commit SHA. This happens based on the
// cached information about refs in the upstream repo. If the tag doesn't exist
// in the upstream repo, the last return value will be false.
func (gur *GitUpstreamRepo) ResolveTag(tag string) (string, bool) {
	tag = strings.TrimPrefix(tag, "refs/tags/")
	for t, commit := range gur.Tags {
		if t == tag {
			return commit, true
		}
	}
	return "", false
}

// ResolveRef resolves the ref (either tag or branch) to a commit SHA. If the
// ref doesn't exist in the upstream repo, the last return value will be false.
func (gur *GitUpstreamRepo) ResolveRef(ref string) (string, bool) {
	commit, found := gur.ResolveBranch(ref)
	if found {
		return commit, true
	}
	return gur.ResolveTag(ref)
}

// getRepoDir returns the cache directory name for a remote repo
// This takes the md5 hash of the repo uri and then base32 (or hex for Windows to shorten dir)
// encodes it to make sure it doesn't contain characters that isn't legal in directory names.
func (gur *GitUpstreamRepo) getRepoDir(uri string) string {
	if runtime.GOOS == "windows" {
		var hash = md5.Sum([]byte(uri))
		return strings.ToLower(hex.EncodeToString(hash[:]))	
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(md5.New().Sum([]byte(uri))))
}

// getRepoCacheDir
func (gur *GitUpstreamRepo) getRepoCacheDir() (string, error) {
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

// cacheRepo fetches a remote repo to a cache location, and fetches the provided refs.
func (gur *GitUpstreamRepo) cacheRepo(ctx context.Context, uri string, requiredRefs []string, optionalRefs []string) (string, error) {
	const op errors.Op = "gitutil.cacheRepo"
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
		return "", errors.E(op, errors.Repo(uri), err)
	}
	uriSha := gur.getRepoDir(uri)
	repoCacheDir := filepath.Join(kptCacheDir, uriSha)
	if _, err := os.Stat(repoCacheDir); os.IsNotExist(err) {
		if _, err := gitRunner.Run(ctx, "init", uriSha); err != nil {
			AmendGitExecError(err, func(e *GitExecError) {
				e.Repo = uri
			})
			return "", errors.E(op, errors.Git, fmt.Errorf("error running `git init`: %w", err))
		}
		gitRunner.Dir = repoCacheDir
		if _, err = gitRunner.Run(ctx, "remote", "add", "origin", uri); err != nil {
			AmendGitExecError(err, func(e *GitExecError) {
				e.Repo = uri
			})
			return "", errors.E(op, errors.Git, fmt.Errorf("error adding origin remote: %w", err))
		}
	} else {
		gitRunner.Dir = repoCacheDir
	}

loop:
	for i := range requiredRefs {
		s := requiredRefs[i]
		// Check if we can verify the ref. This will output a full commit sha if
		// either the ref (short commit, tag, branch) can be resolved to a full
		// commit sha, or if the provided ref is already a valid full commit sha (note
		// that this will happen even if the commit doesn't exist in the local repo).
		// We ignore the error here since an error just means the ref didn't exist,
		// which we detect by checking the output to stdout.
		rr, _ := gitRunner.Run(ctx, "rev-parse", "--verify", "-q", s)
		// If the output is the same as the ref, then the ref was already a full
		// commit sha.
		validFullSha := s == strings.TrimSpace(rr.Stdout)
		_, resolved := gur.ResolveRef(s)
		// check if ref was previously fetched
		// we use the ref s as the cache key
		_, fetched := gur.fetchedRefs[s]
		switch {
		case fetched:
			// skip refetching if previously fetched
			break
		case resolved || validFullSha:
			// If the ref references a branch or a tag, or is a valid commit
			// sha and has not already been fetched, we can fetch just a single commit.
			if _, err := gitRunner.RunVerbose(ctx, "fetch", "origin", "--depth=1", s); err != nil {
				AmendGitExecError(err, func(e *GitExecError) {
					e.Repo = uri
					e.Command = "fetch"
					e.Ref = s
				})
				return "", errors.E(op, errors.Git, fmt.Errorf(
					"error running `git fetch` for ref %q: %w", s, err))
			}
			gur.fetchedRefs[s] = true
		default:
			// In other situations (like a short commit sha), we have to do
			// a full fetch from the remote.
			if _, err := gitRunner.RunVerbose(ctx, "fetch", "origin"); err != nil {
				AmendGitExecError(err, func(e *GitExecError) {
					e.Repo = uri
					e.Command = "fetch"
				})
				return "", errors.E(op, errors.Git, fmt.Errorf(
					"error running `git fetch` for origin: %w", err))
			}
			if _, err = gitRunner.Run(ctx, "show", s); err != nil {
				AmendGitExecError(err, func(e *GitExecError) {
					e.Repo = uri
					e.Ref = s
				})
				return "", errors.E(op, errors.Git, fmt.Errorf(
					"error verifying results from fetch: %w", err))
			}
			gur.fetchedRefs[s] = true
			// If we did a full fetch, we already have all refs, so we can just
			// exit the loop.
			break loop
		}
	}

	var found bool
	for _, s := range optionalRefs {
		if _, err := gitRunner.Run(ctx, "fetch", "origin", s); err == nil {
			found = true
		}
	}
	if !found && len(optionalRefs) > 0 {
		return "", errors.E(op, errors.Git, fmt.Errorf("unable to find any refs %s",
			strings.Join(optionalRefs, ",")))
	}
	return repoCacheDir, nil
}
