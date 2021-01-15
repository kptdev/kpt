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

// Package get contains libraries for fetching packages.
package get

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
)

// Command fetches a package from a git repository and copies it to a local directory.
type Command struct {
	// Git contains information about the git repo to fetch
	kptfile.Git

	// Destination is the output directory to clone the package to.  Defaults to the name of the package --
	// either the base repo name, or the base subdirectory name.
	Destination string

	// Name is the name to give the package.  Defaults to the destination.
	Name string

	// Remove directory before copying to it.
	Clean bool
}

// Run runs the Command.
func (c Command) Run() error {
	if err := (&c).DefaultValues(); err != nil {
		return err
	}

	if _, err := os.Stat(c.Destination); !c.Clean && !os.IsNotExist(err) {
		return errors.Errorf("destination directory %s already exists", c.Destination)
	}

	// normalize path to a filepath
	if !strings.HasSuffix(c.Directory, "file://") {
		c.Directory = filepath.Join(path.Split(c.Directory))
	}

	// define where we are going to clone the package from
	r := &git.RepoSpec{OrgRepo: c.Repo, Path: c.Directory, Ref: c.Ref}

	err := cloneAndCopy(r, c.Destination, c.Name, c.Clean)
	if err != nil {
		return err
	}

	if err = c.fetchRemoteSubpackages(); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

// fetchRemoteSubpackages goes through the root package and its subpackages
// and fetches any remote subpackages referenced. It will also handle situations
// where a remote subpackage references other remote subpackages.
func (c Command) fetchRemoteSubpackages() error {
	// Create a stack to keep track of all Kptfiles that needs to be checked
	// for remote subpackages.
	stack := newStack()

	paths, err := pathutil.DirsWithFile(c.Destination, kptfile.KptFileName, true)
	if err != nil {
		return err
	}
	for _, p := range paths {
		stack.push(p)
	}

	for stack.len() > 0 {
		p := stack.pop()
		kf, err := kptfileutil.ReadFile(p)
		if err != nil {
			return err
		}

		remoteSubPkgDirs := make(map[string]bool)
		for i := range kf.Subpackages {
			sp := kf.Subpackages[i]

			if _, found := remoteSubPkgDirs[sp.LocalDir]; found {
				return fmt.Errorf("multiple remote subpackages with localDir %q", sp.LocalDir)
			}
			remoteSubPkgDirs[sp.LocalDir] = true

			gitInfo := sp.Git
			localPath := filepath.Join(p, sp.LocalDir)

			_, err = os.Stat(localPath)
			// If we get an error and it is something different than that the
			// directory doesn't exist, we just return the error.
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			// Check if the folder already exist by checking if err is nil. Due
			// to the check above, err here can only be IsNotExist or nil. So
			// if err is nil it means the folder already exists.
			// If it does, we return an error with a specific error message.
			if err == nil {
				return fmt.Errorf("local subpackage in directory %q already exists. Either"+
					"rename the local subpackage or use a different directory for the remote subpackage", sp.LocalDir)
			}

			r := &git.RepoSpec{OrgRepo: gitInfo.Repo, Path: gitInfo.Directory, Ref: gitInfo.Ref}
			err := cloneAndCopy(r, localPath, sp.LocalDir, false)
			if err != nil {
				return err
			}

			subPaths, err := pathutil.DirsWithFile(localPath, kptfile.KptFileName, true)
			if err != nil {
				return err
			}
			for _, subp := range subPaths {
				if subp == localPath {
					continue
				}
				stack.push(subp)
			}
		}
	}
	return nil
}

// cloneAndCopy fetches the provided repo and copies the content into the
// directory specified by dest. The provided name is set as `metadata.name`
// of the Kptfile of the package.
func cloneAndCopy(r *git.RepoSpec, dest, name string, clean bool) error {
	defaultRef, err := gitutil.DefaultRef(r.OrgRepo)
	if err != nil {
		return err
	}

	if err := ClonerUsingGitExec(r, defaultRef); err != nil {
		return errors.Errorf("failed to clone git repo: %v", err)
	}
	defer os.RemoveAll(r.Dir)

	// delete the existing package if it exists
	if clean {
		err = os.RemoveAll(dest)
		if err != nil {
			return errors.Wrap(err)
		}
	}

	if err := copyutil.CopyDir(r.AbsPath(), dest); err != nil {
		return errors.WrapPrefixf(err, "missing subdirectory %s in repo %s at ref %s\n",
			r.Path, r.OrgRepo, r.Ref)
	}

	if err := upsertKptfile(dest, name, r); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func newStack() *stack {
	return &stack{
		slice: make([]string, 0),
	}
}

type stack struct {
	slice []string
}

func (s *stack) push(str string) {
	s.slice = append(s.slice, str)
}

func (s *stack) pop() string {
	l := len(s.slice)
	if l == 0 {
		panic(fmt.Errorf("can't pop an empty stack"))
	}
	str := s.slice[l-1]
	s.slice = s.slice[:l-1]
	return str
}

func (s *stack) len() int {
	return len(s.slice)
}

// Cloner is a function that can clone a git repo.
type Cloner func(repoSpec *git.RepoSpec) error

// ClonerUsingGitExec uses a local git install, as opposed
// to say, some remote API, to obtain a local clone of
// a remote repo.
func ClonerUsingGitExec(repoSpec *git.RepoSpec, defaultRef string) error {
	// look for a tag with the directory as a prefix for versioning
	// subdirectories independently
	originalRef := repoSpec.Ref
	if repoSpec.Path != "" && !strings.Contains(repoSpec.Ref, "refs") {
		// join the directory with the Ref (stripping the preceding '/' if it exists)
		repoSpec.Ref = path.Join(strings.TrimLeft(repoSpec.Path, "/"), repoSpec.Ref)
	}

	// clone the repo to a tmp directory.
	// delete the tmp directory later.
	err := clonerUsingGitExec(repoSpec)
	if err != nil && originalRef != repoSpec.Ref {
		repoSpec.Ref = originalRef
		err = clonerUsingGitExec(repoSpec)
	}

	if err != nil {
		if strings.HasPrefix(repoSpec.Path, "blob/") {
			return errors.Errorf("failed to clone git repo containing /blob/, "+
				"you may need to remove /blob/%s from the url:\n%v", defaultRef, err)
		}
		return errors.Errorf("failed to clone git repo: %v", err)
	}

	return nil
}

func clonerUsingGitExec(repoSpec *git.RepoSpec) error {
	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return errors.WrapPrefixf(err, "no 'git' program on path")
	}

	repoSpec.Dir, err = ioutil.TempDir("", "kpt-get-")
	if err != nil {
		return err
	}
	cmd := exec.Command(gitProgram, "init", repoSpec.Dir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing empty git repo: %s", out.String())
		return errors.WrapPrefixf(err, "trouble initializing empty git repo in %s",
			repoSpec.Dir)
	}

	cmd = exec.Command(gitProgram, "remote", "add", "origin", repoSpec.CloneSpec())
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = repoSpec.Dir
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting git remote: %s", out.String())
		return errors.WrapPrefixf(
			err,
			"trouble adding remote %s",
			repoSpec.CloneSpec())
	}
	if repoSpec.Ref == "" {
		repoSpec.Ref, err = gitutil.DefaultRef(repoSpec.Dir)
		if err != nil {
			return err
		}
	}

	err = func() error {
		cmd = exec.Command(gitProgram, "fetch", "origin", "--depth=1", repoSpec.Ref)
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		err = cmd.Run()
		if err != nil {
			return errors.WrapPrefixf(err, "trouble fetching %s, "+
				"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", repoSpec.Ref)
		}
		cmd = exec.Command(gitProgram, "reset", "--hard", "FETCH_HEAD")
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		err = cmd.Run()
		if err != nil {
			return errors.WrapPrefixf(
				err, "trouble hard resetting empty repository to %s", repoSpec.Ref)
		}
		return nil
	}()
	if err != nil {
		cmd = exec.Command(gitProgram, "fetch", "origin")
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		if err = cmd.Run(); err != nil {
			return errors.WrapPrefixf(err, "trouble fetching origin, "+
				"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials")
		}
		cmd = exec.Command(gitProgram, "reset", "--hard", repoSpec.Ref)
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = repoSpec.Dir
		if err = cmd.Run(); err != nil {
			return errors.WrapPrefixf(
				err, "trouble hard resetting empty repository to %s, "+
					"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", repoSpec.Ref)
		}
	}

	cmd = exec.Command(gitProgram, "submodule", "update", "--init", "--recursive")
	cmd.Stdout = &out
	cmd.Dir = repoSpec.Dir
	err = cmd.Run()
	if err != nil {
		return errors.WrapPrefixf(err, "trouble fetching submodules for %s, "+
			"please run 'git clone <REPO>; stat <DIR/SUBDIR>' to verify credentials", repoSpec.Ref)
	}

	return nil
}

// DefaultValues sets values to the default values if they were unspecified
func (c *Command) DefaultValues() error {
	if len(c.Repo) == 0 {
		return errors.Errorf("must specify repo")
	}
	if len(c.Ref) == 0 {
		return errors.Errorf("must specify ref")
	}
	if len(c.Destination) == 0 {
		return errors.Errorf("must specify destination")
	}
	if len(c.Directory) == 0 {
		return errors.Errorf("must specify remote subdirectory")
	}

	if !filepath.IsAbs(c.Destination) {
		return errors.Errorf("destination must be an absolute path")
	}

	// default the name to the destination name
	if len(c.Name) == 0 {
		c.Name = filepath.Base(c.Destination)
	}

	return nil
}

// upsertKptfile populates the KptFile values, merging any cloned KptFile and the
// cloneFrom values.
func upsertKptfile(path, name string, spec *git.RepoSpec) error {
	// read KptFile cloned with the package if it exists
	kpgfile, err := kptfileutil.ReadFile(path)
	if err != nil {
		// no KptFile present, create a default
		kpgfile = kptfileutil.DefaultKptfile(name)
	}
	kpgfile.Name = name

	// find the git commit sha that we cloned the package at so we can write it to the KptFile
	commit, err := git.LookupCommit(spec.AbsPath())
	if err != nil {
		return err
	}

	// populate the cloneFrom values so we know where the package came from
	kpgfile.Upstream = kptfile.Upstream{
		Type: kptfile.GitOrigin,
		Git: kptfile.Git{
			Repo:      spec.OrgRepo,
			Directory: spec.Path,
			Ref:       spec.Ref,
		},
	}
	kpgfile.Upstream.Git.Commit = commit
	return kptfileutil.WriteFile(path, kpgfile)
}
