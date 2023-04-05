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

// Package get contains libraries for fetching packages.
package get

import (
	"context"
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/hook"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/addmergecomment"
	"github.com/GoogleContainerTools/kpt/internal/util/attribution"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
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

	// IsDeploymentInstance indicates if the package is forked for deployment.
	// If forked package has defined deploy hooks, those will be executed post fork.
	IsDeploymentInstance bool

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
		// Convert from separator to slash and back.
		// This ensures all separators are compatible with the local OS.
		repoDir = filepath.FromSlash(filepath.ToSlash(repoDir))
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
		return cleanUpDirAndError(c.Destination, err)
	}

	absDestPath, _, err := pathutil.ResolveAbsAndRelPaths(c.Destination)
	if err != nil {
		return err
	}
	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, absDestPath)
	if err != nil {
		return cleanUpDirAndError(c.Destination, err)
	}

	if err = c.fetchPackages(ctx, p); err != nil {
		return cleanUpDirAndError(c.Destination, err)
	}

	inout := &kio.LocalPackageReadWriter{PackagePath: c.Destination, PreserveSeqIndent: true, WrapBareSeqNode: true}
	amc := &addmergecomment.AddMergeComment{}
	at := &attribution.Attributor{PackagePaths: []string{c.Destination}, CmdGroup: "pkg"}
	// do not error out as this is best effort
	_ = kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: []kio.Filter{kio.FilterAll(amc), kio.FilterAll(at)},
		Outputs: []kio.Writer{inout},
	}.Execute()

	if c.IsDeploymentInstance {
		pr := printer.FromContextOrDie(ctx)
		pr.Printf("\nCustomizing package for deployment.\n")
		hookCmd := hook.Executor{}
		hookCmd.RunnerOptions.InitDefaults()
		hookCmd.PkgPath = c.Destination

		builtinHooks := []kptfilev1.Function{
			{
				Image: fnruntime.FuncGenPkgContext,
			},
		}
		if err := hookCmd.Execute(ctx, builtinHooks); err != nil {
			return err
		}
		pr.Printf("\nCustomized package for deployment.\n")
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
			packageCount++
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

func cleanUpDirAndError(destination string, err error) error {
	const op errors.Op = "get.Run"
	rmErr := os.RemoveAll(destination)
	if rmErr != nil {
		return errors.E(op, types.UniquePath(destination), err, rmErr)
	}
	return errors.E(op, types.UniquePath(destination), err)
}
