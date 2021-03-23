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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// Command fetches a package from a git repository, copies it to a local
// directory, and expands any remote subpackages.
type Command struct {
	// Git contains information about the git repo to fetch
	Git *kptfilev1alpha2.Git

	// Destination is the output directory to clone the package to.  Defaults to the name of the package --
	// either the base repo name, or the base subdirectory name.
	Destination string

	// Name is the name to give the package.  Defaults to the destination.
	Name string
}

// Run runs the Command.
func (c Command) Run() error {
	if err := (&c).DefaultValues(); err != nil {
		return err
	}

	if _, err := os.Stat(c.Destination); !os.IsNotExist(err) {
		return errors.Errorf("destination directory %s already exists", c.Destination)
	}

	err := os.MkdirAll(c.Destination, 0700)
	if err != nil {
		return err
	}

	// normalize path to a filepath
	repoDir := c.Git.Directory
	if !strings.HasSuffix(repoDir, "file://") {
		repoDir = filepath.Join(path.Split(repoDir))
	}
	c.Git.Directory = repoDir

	kf := kptfileutil.DefaultKptfile(c.Name)
	kf.Upstream = &kptfilev1alpha2.Upstream{
		Type:           kptfilev1alpha2.GitOrigin,
		Git:            c.Git,
		UpdateStrategy: kptfilev1alpha2.ResourceMerge,
	}

	err = kptfileutil.WriteFile(c.Destination, kf)
	if err != nil {
		return err
	}

	p, err := pkg.New(c.Destination)
	if err != nil {
		return err
	}

	if err = c.fetchPackages(p); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

// fetchRemoteSubpackages goes through the root package and its subpackages
// and fetches any remote subpackages referenced. It will also handle situations
// where a remote subpackage references other remote subpackages.
func (c Command) fetchPackages(rootPkg *pkg.Pkg) error {
	// Create a stack to keep track of all Kptfiles that needs to be checked
	// for remote subpackages.
	s := stack.NewPkgStack()
	s.Push(rootPkg)

	for s.Len() > 0 {
		p := s.Pop()

		kf, err := p.Kptfile()
		if err != nil {
			return err
		}

		if kf.Upstream != nil && kf.UpstreamLock == nil {
			err := (&fetch.Command{
				Pkg: p,
			}).Run()
			if err != nil {
				return err
			}
		}

		subPkgs, err := p.DirectSubpackages()
		if err != nil {
			return err
		}
		for _, subPkg := range subPkgs {
			s.Push(subPkg)
		}
	}
	return nil
}

// DefaultValues sets values to the default values if they were unspecified
func (c *Command) DefaultValues() error {
	if c.Git == nil {
		return errors.Errorf("must specify git repo information")
	}
	g := c.Git
	if len(g.Repo) == 0 {
		return errors.Errorf("must specify repo")
	}
	if len(g.Ref) == 0 {
		return errors.Errorf("must specify ref")
	}
	if len(c.Destination) == 0 {
		return errors.Errorf("must specify destination")
	}
	if len(g.Directory) == 0 {
		return errors.Errorf("must specify directory")
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
