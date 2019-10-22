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

	"github.com/pkg/errors"
)

// RepoCacheDirEnv is the name of the environment variable that controls the cache directory
// for remote repos.  Defaults to UserHomeDir/.kpt/repos if unspecified.
const RepoCacheDirEnv = "KPT_CACHE_DIR"

// NewUpstreamGitRunner returns a new GitRunner for an upstream package.
//
// The upstream package repo will be fetched to a local cache directory under $HOME/.kpt
// and hard reset to origin/master.
// The refs will also be fetched so they are available locally.
func NewUpstreamGitRunner(uri, dir string, refs ...string) (*GitRunner, error) {
	g := &GitRunner{}

	// make sure the repo is fetched
	cacheDir, err := g.cacheRepo(uri, dir, refs)
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
		return errors.Wrap(err, "no 'git' program on path")
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
		return "", fmt.Errorf(
			"failed to clone repo: trouble resolving cache directory: %v", err)
	}
	return filepath.Join(dir, ".kpt", "repos"), nil
}

// cacheRepo fetches a remote repo to a cache location, and fetches the provided refs.
func (g *GitRunner) cacheRepo(uri, dir string, refs []string) (string, error) {
	kptCacheDir, err := g.getRepoCacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(kptCacheDir, 0700); err != nil {
		return "", fmt.Errorf(
			"failed to clone repo: trouble creating cache directory: %v", err)
	}

	// create the repo directory if it doesn't exist yet
	gitRunner := GitRunner{Dir: kptCacheDir}
	uriSha := g.getRepoDir(uri)
	repoCacheDir := filepath.Join(kptCacheDir, uriSha)
	if _, err := os.Stat(repoCacheDir); os.IsNotExist(err) {
		if err := gitRunner.Run("init", uriSha); err != nil {
			return "", fmt.Errorf("failed to clone repo: trouble running init: %v", err)
		}
		gitRunner.Dir = repoCacheDir
		if err = gitRunner.Run("remote", "add", "origin", uri); err != nil {
			return "", fmt.Errorf("failed to clone repo: trouble adding origin: %v", err)
		}
	} else {
		gitRunner.Dir = repoCacheDir
	}

	// fetch the specified refs
	for _, s := range refs {
		if err = gitRunner.Run("fetch", "origin", s); err != nil {
			return "", fmt.Errorf(
				"failed to clone git repo: trouble fetching origin %s: %v", s, err)
		}
	}
	if err = gitRunner.Run("fetch", "origin"); err != nil {
		return "", fmt.Errorf("failed to clone git repo: trouble fetching origin: %v", err)
	}

	// reset the repo state
	if err = gitRunner.Run("checkout", "master"); err != nil {
		return "", fmt.Errorf("failed to clone repo: trouble checking out master: %v", err)
	}

	// TODO: make this safe for concurrent operations
	if err = gitRunner.Run("reset", "--hard", "origin/master"); err != nil {
		return "", fmt.Errorf("failed to clone repo: trouble reset to master: %v", err)
	}
	gitRunner.Dir = filepath.Join(repoCacheDir, dir)
	return repoCacheDir, nil
}
