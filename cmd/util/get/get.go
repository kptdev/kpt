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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"lib.kpt.dev/copyutil"
	"lib.kpt.dev/kptfile"
	"lib.kpt.dev/kptfile/kptfileutil"
	"lib.kpt.dev/yaml"
	"sigs.k8s.io/kustomize/v3/pkg/git"
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
		return fmt.Errorf("destination directory %s already exists", c.Destination)
	}

	// normalize path to a filepath
	if !strings.HasSuffix(c.Directory, "file://") {
		c.Directory = filepath.Join(path.Split(c.Directory))
	}

	// define where we are going to clone the package from
	r := &git.RepoSpec{
		OrgRepo: c.Repo,
		Path:    c.Directory,
		Ref:     c.Ref,
	}

	// clone the repo to a tmp directory.
	// delete the tmp directory later.
	err := git.ClonerUsingGitExec(r)
	if err != nil {
		return fmt.Errorf("failed to clone git repo: %v", err)
	}
	defer os.RemoveAll(r.AbsPath())

	// delete the existing package if it exists
	if c.Clean {
		err = os.RemoveAll(c.Destination)
		if err != nil {
			return err
		}
	}

	// copy the git sub directory to the destination
	err = copyutil.CopyDir(r.AbsPath(), c.Destination)
	if err != nil {
		return err
	}

	// create or update the KptFile with the values from git
	if err = (&c).upsertKptfile(r); err != nil {
		return err
	}
	return nil
}

// DefaultValues sets values to the default values if they were unspecified
func (c *Command) DefaultValues() error {
	if len(c.Repo) == 0 {
		return fmt.Errorf("must specify repo")
	}
	if len(c.Ref) == 0 {
		return fmt.Errorf("must specify ref")
	}
	if len(c.Destination) == 0 {
		return fmt.Errorf("must specify destination")
	}
	if len(c.Directory) == 0 {
		return fmt.Errorf("must specify remote subdirectory")
	}

	// default the name to the destination name
	if len(c.Name) == 0 {
		c.Name = filepath.Base(c.Destination)
	}

	return nil
}

// upsertKptfile populates the KptFile values, merging any cloned KptFile and the
// cloneFrom values.
func (c *Command) upsertKptfile(spec *git.RepoSpec) error {
	// read KptFile cloned with the package if it exists
	kpgfile, err := kptfileutil.ReadFile(c.Destination)
	if err != nil {
		// no KptFile present, create a default
		kpgfile = kptfile.KptFile{
			ResourceMeta: yaml.ResourceMeta{
				ApiVersion: kptfile.TypeMeta.ApiVersion,
				Kind:       kptfile.TypeMeta.Kind,
				ObjectMeta: yaml.ObjectMeta{Name: c.Name},
			},
		}
	}

	// find the git commit sha that we cloned the package at so we can write it to the KptFile
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = spec.AbsPath()
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return err
	}
	commit := strings.TrimSpace(string(b))

	// populate the cloneFrom values so we know where the package came from
	kpgfile.Upstream = kptfile.Upstream{
		Type: kptfile.GitOrigin,
		Git:  c.Git,
	}
	kpgfile.Upstream.Git.Commit = commit

	// update the KptFile with the new values
	contents, err := yaml.Marshal(kpgfile)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(c.Destination, kptfile.KptFileName), contents, 0600)
}
