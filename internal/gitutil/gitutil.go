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

package gitutil

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base32"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
)

// RepoCacheDirEnv is the name of the environment variable that controls the cache directory
// for remote repos.  Defaults to UserHomeDir/.kpt/repos if unspecified.
const RepoCacheDirEnv = "KPT_CACHE_DIR"

// NewLocalGitRunner returns a new GitLocalRunner for a local package.
func NewLocalGitRunner(pkg string) (*GitLocalRunner, error) {
	const op errors.Op = "gitutil.NewLocalGitRunner"
	p, err := exec.LookPath("git")
	if err != nil {
		return nil, errors.E(op, errors.Git,
			fmt.Errorf("no 'git' program on path: %w", err))
	}

	return &GitLocalRunner{
		gitPath: p,
		Dir:     pkg,
	}, nil
}

// GitLocalRunner runs git commands in a local git repo.
type GitLocalRunner struct {
	// Path to the git executable.
	gitPath string

	// Dir is the directory the commands are run in.
	Dir string
}

type RunResult struct {
	Stdout string
	Stderr string
}

// Run runs a git command.
// Omit the 'git' part of the command.
// The first return value contains the output to Stdout and Stderr when
// running the command.
func (g *GitLocalRunner) Run(ctx context.Context, args ...string) (RunResult, error) {
	return g.run(ctx, false, args...)
}

// RunVerbose runs a git command.
// Omit the 'git' part of the command.
// The first return value contains the output to Stdout and Stderr when
// running the command.
func (g *GitLocalRunner) RunVerbose(ctx context.Context, args ...string) (RunResult, error) {
	return g.run(ctx, true, args...)
}

// run runs a git command.
// Omit the 'git' part of the command.
// The first return value contains the output to Stdout and Stderr when
// running the command.
func (g *GitLocalRunner) run(ctx context.Context, verbose bool, args ...string) (RunResult, error) {
	const op errors.Op = "gitutil.run"

	cmd := exec.CommandContext(ctx, g.gitPath, args...)
	cmd.Dir = g.Dir
	cmd.Env = os.Environ()

	cmdStdout := &bytes.Buffer{}
	cmdStderr := &bytes.Buffer{}
	if verbose {
		cmd.Stdout = io.MultiWriter(cmdStdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(cmdStderr, os.Stderr)
	} else {
		cmd.Stdout = cmdStdout
		cmd.Stderr = cmdStderr
	}

	err := cmd.Run()
	if err != nil {
		return RunResult{}, errors.E(op, errors.Git, &GitExecError{
			Args:   args,
			Err:    err,
			StdOut: cmdStdout.String(),
			StdErr: cmdStderr.String(),
		})
	}
	return RunResult{
		Stdout: cmdStdout.String(),
		Stderr: cmdStderr.String(),
	}, nil
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

// NewGitUpstreamRepo returns a new GitUpstreamRepo for an upstream package.
func NewGitUpstreamRepo(ctx context.Context, uri string) (*GitUpstreamRepo, error) {
	const op errors.Op = "gitutil.NewGitUpstreamRepo"

	g := &GitUpstreamRepo{
		URI: uri,
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
	if strings.HasPrefix(branch, "refs/heads/") {
		branch = strings.TrimPrefix(branch, "refs/heads/")
	}
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
	if strings.HasPrefix(tag, "refs/tags/") {
		tag = strings.TrimPrefix(tag, "refs/tags/")
	}
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
// This takes the md5 hash of the repo uri and then base32 encodes it to make
// sure it doesn't contain characters that isn't legal in directory names.
func (gur *GitUpstreamRepo) getRepoDir(uri string) string {
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
			return "", errors.E(op, errors.Git, fmt.Errorf("error running `git init`: %w", err))
		}
		gitRunner.Dir = repoCacheDir
		if _, err = gitRunner.Run(ctx, "remote", "add", "origin", uri); err != nil {
			return "", errors.E(op, errors.Git, fmt.Errorf("error adding origin remote: %w", err))
		}
	} else {
		gitRunner.Dir = repoCacheDir
	}

	// fetch the specified refs
	triedFallback := false
	for _, s := range requiredRefs {
		if _, err := gitRunner.Run(ctx, "fetch", "origin", "--depth=1", s); err != nil {
			if !triedFallback { // only fallback to fetch origin once
				// fallback on fetching the origin. If the user provided a short sha,
				// we need to fetch all objects in order to resolve it into the full
				// sha.
				// TODO: See if there is a way to resolve a short sha into a complete
				// sha without fetching. Haven't found one so far...
				if _, retryErr := gitRunner.Run(ctx, "fetch", "origin"); retryErr != nil {
					// We are using the original error here.
					return "", errors.E(op, errors.Git, fmt.Errorf(
						"error running `git fetch` for origin: %w", err))
				}
				triedFallback = true
			}
			// verify we got the commit
			if _, err = gitRunner.Run(ctx, "show", s); err != nil {
				return "", errors.E(op, errors.Git, fmt.Errorf(
					"error verifying results from fetch: %w", err))
			}
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
