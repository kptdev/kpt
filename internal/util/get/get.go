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
	"context"
	goerrors "errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/addmergecomment"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
)

// Command fetches a package from a git repository, copies it to a local
// directory, and expands any remote subpackages.
type Command struct {
	// Git contains information about the git repo to fetch
	Git *kptfilev1.Git

	// Destination is the output directory to clone the package to.  Defaults to the name of the package --
	// either the base repo name, or the base subdirectory name.
	Destination string

	// Name is the name to give the package.  Defaults to the destination.
	Name string

	// UpdateStrategy is the strategy that will be configured in the package
	// Kptfile. This determines how changes will be merged when updating the
	// package.
	UpdateStrategy kptfilev1.UpdateStrategyType
}

// Run runs the Command.
func (c Command) Run(ctx context.Context) error {
	const op errors.Op = "get.Run"
	if err := (&c).DefaultValues(); err != nil {
		return errors.E(op, err)
	}

	if _, err := os.Stat(c.Destination); !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, errors.Exist, types.UniquePath(c.Destination), fmt.Errorf("destination directory already exists"))
	}

	err := os.MkdirAll(c.Destination, 0700)
	if err != nil {
		return errors.E(op, errors.IO, types.UniquePath(c.Destination), err)
	}

	// normalize path to a filepath
	repoDir := c.Git.Directory
	if !strings.HasSuffix(repoDir, "file://") {
		repoDir = filepath.Join(path.Split(repoDir))
	}
	c.Git.Directory = repoDir

	kf := kptfileutil.DefaultKptfile(c.Name)
	kf.Upstream = &kptfilev1.Upstream{
		Type:           kptfilev1.GitOrigin,
		Git:            c.Git,
		UpdateStrategy: c.UpdateStrategy,
	}

	err = kptfileutil.WriteFile(c.Destination, kf)
	if err != nil {
		return errors.E(op, types.UniquePath(c.Destination), err)
	}

	p, err := pkg.New(c.Destination)
	if err != nil {
		return errors.E(op, types.UniquePath(c.Destination), err)
	}

	if err = c.fetchPackages(ctx, p); err != nil {
		return errors.E(op, types.UniquePath(c.Destination), err)
	}

	if err := addmergecomment.Process(c.Destination); err != nil {
		return errors.E(op, types.UniquePath(c.Destination), err)
	}
	return nil
}

// Fetches any remote subpackages referenced through the root package and its subpackages.
// It will also handle situations where a remote subpackage references other remote subpackages.
func (c Command) fetchPackages(ctx context.Context, rootPkg *pkg.Pkg) error {
	const op errors.Op = "get.fetchPackages"
	pr := printer.FromContextOrDie(ctx)
	packageCount := 0
	// Create a stack to keep track of all Kptfiles that needs to be checked
	// for remote subpackages.
	s := stack.NewPkgStack()
	s.Push(rootPkg)

	for s.Len() > 0 {
		p := s.Pop()

		kf, err := p.Kptfile()
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}

		if kf.Upstream != nil && kf.UpstreamLock == nil {
			packageCount += 1
			pr.PrintPackage(p, !(p == rootPkg))
			pr.Printf("Fetching %s@%s\n", kf.Upstream.Git.Repo, kf.Upstream.Git.Ref)
			err := (&fetch.Command{
				Pkg: p,
			}).Run(ctx)
			if err != nil {
				return errors.E(op, p.UniquePath, err)
			}
		}

		subPkgs, err := p.DirectSubpackages()
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}
		for _, subPkg := range subPkgs {
			s.Push(subPkg)
		}
	}
	pr.Printf("\nFetched %d package(s).\n", packageCount)
	return nil
}

// DefaultValues sets values to the default values if they were unspecified
func (c *Command) DefaultValues() error {
	const op errors.Op = "get.DefaultValues"
	if c.Git == nil {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify git repo information"))
	}
	g := c.Git
	if len(g.Repo) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify repo"))
	}
	if len(g.Ref) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify ref"))
	}
	if len(c.Destination) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify destination"))
	}
	if len(g.Directory) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify directory"))
	}

	if !filepath.IsAbs(c.Destination) {
		return errors.E(op, errors.InvalidParam, fmt.Errorf("destination must be an absolute path"))
	}

	// default the name to the destination name
	if len(c.Name) == 0 {
		c.Name = filepath.Base(c.Destination)
	}

	// default the update strategy to resource-merge
	if len(c.UpdateStrategy) == 0 {
		c.UpdateStrategy = kptfilev1.ResourceMerge
	}

	return nil
}
