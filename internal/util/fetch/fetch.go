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
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// Command fetches a package from a git repository and copies it to a local
// directory.
type Command struct {
	RepoSpec *git.RepoSpec

	Destination string

	Name string

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
	if !strings.HasSuffix(c.RepoSpec.Dir, "file://") {
		c.RepoSpec.Dir = filepath.Join(path.Split(c.RepoSpec.Dir))
	}

	err := cloneAndCopy(c.RepoSpec, c.Destination, c.Name, c.Clean)
	if err != nil {
		return err
	}
	return nil
}

// cloneAndCopy fetches the provided repo and copies the content into the
// directory specified by dest. The provided name is set as `metadata.name`
// of the Kptfile of the package.
func cloneAndCopy(r *git.RepoSpec, dest, name string, clean bool) error {
	if err := ClonerUsingGitExec(r); err != nil {
		return errors.Errorf("failed to clone git repo: %v", err)
	}
	defer os.RemoveAll(r.Dir)

	// delete the existing package if it exists
	if clean {
		if err := os.RemoveAll(dest); err != nil {
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

// ClonerUsingGitExec uses a local git install, as opposed
// to say, some remote API, to obtain a local clone of
// a remote repo. It looks for tags with the directory as a prefix to allow
// for versioning multiple kpt packages in a single repo independently. It
// relies on the private clonerUsingGitExec function to try fetching different
// refs.
func ClonerUsingGitExec(repoSpec *git.RepoSpec) error {
	// look for a tag with the directory as a prefix for versioning
	// subdirectories independently
	originalRef := repoSpec.Ref
	if repoSpec.Path != "" && !strings.Contains(repoSpec.Ref, "refs") {
		// join the directory with the Ref (stripping the preceding '/' if it exists)
		repoSpec.Ref = path.Join(strings.TrimLeft(repoSpec.Path, "/"), repoSpec.Ref)
	}

	defaultRef, err := gitutil.DefaultRef(repoSpec.OrgRepo)
	if err != nil {
		return err
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
			return errors.Errorf("failed to clone git repo containing /blob/, "+
				"you may need to remove /blob/%s from the url:\n%v", defaultRef, err)
		}
		return errors.Errorf("failed to clone git repo: %v", err)
	}

	return nil
}

// clonerUsingGitExec is the implementation for cloning a repo from git into
// a local temp directory. This is used by the public ClonerUsingGitExec
// function to allow trying multiple different refs.
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
	if c.RepoSpec == nil {
		return errors.Errorf("must specify a repoSpec")
	}
	r := c.RepoSpec
	if len(r.OrgRepo) == 0 {
		return errors.Errorf("must specify repo")
	}
	if len(r.Ref) == 0 {
		return errors.Errorf("must specify ref")
	}
	if len(c.Destination) == 0 {
		return errors.Errorf("must specify destination")
	}
	if len(r.Path) == 0 {
		return errors.Errorf("must specify path")
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
