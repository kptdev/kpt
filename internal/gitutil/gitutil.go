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
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/errors"
)

// RepoCacheDirEnv is the name of the environment variable that controls the cache directory
// for remote repos.  Defaults to UserHomeDir/.kpt/repos if unspecified.
const RepoCacheDirEnv = "KPT_CACHE_DIR"

// DefaultRef returns the DefaultRef to "master" if master branch exists in
// remote repository, falls back to "main" if master branch doesn't exist
// Making it a var so that it can be overridden for local testing
var DefaultRef = func(repo string) (string, error) {
	masterRef := "master"
	mainRef := "main"
	masterExists, err := branchExists(repo, masterRef)
	if err != nil {
		return "", err
	}
	mainExists, err := branchExists(repo, mainRef)
	if err != nil {
		return "", err
	}
	if masterExists {
		return masterRef, nil
	} else if mainExists {
		return mainRef, nil
	}
	return masterRef, nil
}

// BranchExists checks if branch is present in the input repo
func branchExists(repo, branch string) (bool, error) {
	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return false, errors.Wrap(err)
	}
	stdOut := bytes.Buffer{}
	stdErr := bytes.Buffer{}
	cmd := exec.Command(gitProgram, "ls-remote", repo, branch)
	cmd.Stderr = &stdErr
	cmd.Stdout = &stdOut
	err = cmd.Run()
	if err != nil {
		// stdErr contains the error message for os related errors, git permission errors
		// and if repo doesn't exist
		return false, errors.Errorf("failed to lookup master(or main) branch %q: %s", err, strings.TrimSpace(stdErr.String()))
	}
	// stdOut contains the branch information if the branch is present in remote repo
	// stdOut is empty if the repo doesn't have the input branch
	if strings.TrimSpace(stdOut.String()) != "" {
		return true, nil
	}
	return false, nil
}

// NewUpstreamGitRunner returns a new GitRunner for an upstream package.
//
// The upstream package repo will be fetched to a local cache directory under $HOME/.kpt
// and hard reset to origin/main.
// The refs will also be fetched so they are available locally.
func NewUpstreamGitRunner(uri, dir string, required []string, optional []string) (*GitRunner, error) {
	g := &GitRunner{}

	// make sure the repo is fetched
	cacheDir, err := g.cacheRepo(uri, dir, required, optional)
	if err != nil {
		return nil, err
	}
	g.RepoDir = cacheDir
	g.Dir = filepath.Join(cacheDir, dir)
	return g, nil
}

// NewLocalGitRunner returns a new GitRunner for a local package.
func NewLocalGitRunner(pkg string) *GitRunner {
	return &GitRunner{Dir: pkg}
}

// GitRunner runs git commands in a git repo.
type GitRunner struct {
	// Dir is the directory the commands are run in.
	Dir string

	// RepoDir is the directory of the git repo containing the package
	RepoDir string

	// Stderr is where the git command Stderr is written
	Stderr *bytes.Buffer

	// Stdin is where the git command Stdin is read from
	Stdin *bytes.Buffer

	// Stdout is where the git command Stdout is written
	Stdout *bytes.Buffer

	// Verbose prints verbose command information
	Verbose bool
}

// Run runs a git command.
// Omit the 'git' part of the command.
func (g *GitRunner) Run(args ...string) error {
	p, err := exec.LookPath("git")
	if err != nil {
		return errors.WrapPrefixf(err, "no 'git' program on path")
	}

	cmd := exec.Command(p, args...)
	cmd.Dir = g.Dir
	cmd.Env = os.Environ()

	g.Stdout = &bytes.Buffer{}
	g.Stderr = &bytes.Buffer{}
	if g.Verbose {
		// print the command
		fmt.Println(cmd.Args)
		cmd.Stdout = io.MultiWriter(g.Stdout, os.Stdout)
		cmd.Stdout = io.MultiWriter(g.Stderr, os.Stderr)
	} else {
		cmd.Stdout = g.Stdout
		cmd.Stderr = g.Stderr
	}

	if g.Stdin != nil {
		cmd.Stdin = g.Stdin
	}
	return cmd.Run()
}

// getRepoDir returns the cache directory name for a remote repo
func (g *GitRunner) getRepoDir(uri string) string {
	return base64.URLEncoding.EncodeToString(sha256.New().Sum([]byte(uri)))[:32]
}

func (g *GitRunner) getRepoCacheDir() (string, error) {
	var err error
	dir := os.Getenv(RepoCacheDirEnv)
	if dir != "" {
		return dir, nil
	}

	// cache location unspecified, use UserHomeDir/.kpt/repos
	dir, err = os.UserHomeDir()
	if err != nil {
		return "", errors.Errorf(
			"failed to clone repo: trouble resolving cache directory: %v", err)
	}
	return filepath.Join(dir, ".kpt", "repos"), nil
}

// cacheRepo fetches a remote repo to a cache location, and fetches the provided refs.
func (g *GitRunner) cacheRepo(uri, dir string,
	requiredRefs []string, optionalRefs []string) (string, error) {
	kptCacheDir, err := g.getRepoCacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(kptCacheDir, 0700); err != nil {
		return "", errors.Errorf(
			"failed to clone repo: trouble creating cache directory: %v", err)
	}

	// create the repo directory if it doesn't exist yet
	gitRunner := GitRunner{Dir: kptCacheDir}
	uriSha := g.getRepoDir(uri)
	repoCacheDir := filepath.Join(kptCacheDir, uriSha)
	if _, err := os.Stat(repoCacheDir); os.IsNotExist(err) {
		if err := gitRunner.Run("init", uriSha); err != nil {
			return "", errors.Errorf("failed to clone repo: trouble running init: %v", err)
		}
		gitRunner.Dir = repoCacheDir
		if err = gitRunner.Run("remote", "add", "origin", uri); err != nil {
			return "", errors.Errorf("failed to clone repo: trouble adding origin: %v", err)
		}
	} else {
		gitRunner.Dir = repoCacheDir
	}

	// fetch the specified refs
	triedFallback := false
	for _, s := range requiredRefs {
		if err = gitRunner.Run("fetch", "origin", s); err != nil {
			if !triedFallback { // only fallback to fetch origin once
				// fallback on fetching the origin -- some versions of git have an issue
				// with fetching the first commit by sha.
				if err = gitRunner.Run("fetch", "origin"); err != nil {
					return "", errors.Errorf(
						"failed to clone git repo: trouble fetching origin %v, "+
							"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", err)
				}
				triedFallback = true
			}
			// verify we got the commit
			if err = gitRunner.Run("show", s); err != nil {
				return "", errors.Errorf(
					"failed to clone git repo: trouble fetching origin %q: %v, "+
						"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", s, err)
			}
		}
	}

	var found bool
	for _, s := range optionalRefs {
		if err := gitRunner.Run("fetch", "origin", s); err == nil {
			found = true
		}
	}
	if !found {
		return "", errors.Errorf("failed to clone git repo: unable to find any matching refs: %s, "+
			"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials",
			strings.Join(optionalRefs, ","))
	}

	if err = gitRunner.Run("fetch", "origin"); err != nil {
		return "", errors.Errorf("failed to clone git repo: trouble fetching origin: %v, "+
			"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", err)
	}

	defaultRef, err := DefaultRef(uri)
	if err != nil {
		return "", errors.Errorf("%v, please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", err)
	}

	// reset the repo state
	if err = gitRunner.Run("checkout", defaultRef); err != nil {
		return "", errors.Errorf("failed to clone repo: trouble checking out %s: %v, "+
			"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", defaultRef, err)
	}

	// TODO: make this safe for concurrent operations
	if err = gitRunner.Run("reset", "--hard", "origin/"+defaultRef); err != nil {
		return "", errors.Errorf("failed to clone repo: trouble reset to %s: %v, "+
			"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", defaultRef, err)
	}
	gitRunner.Dir = filepath.Join(repoCacheDir, dir)
	return repoCacheDir, nil
}
