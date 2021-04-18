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

// Package update contains libraries for updating packages.
package update

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

type UpdateOptions struct {
	// RelPackagePath is the relative path of a subpackage to the root. If the
	// package is root, the value here will be ".".
	RelPackagePath string

	// LocalPath is the absolute path to the package on the local fork.
	LocalPath string

	// OriginPath is the absolute path to the package in the on-disk clone
	// of the origin ref of the repo.
	OriginPath string

	// UpdatedPath is the absolute path to the package in the on-disk clone
	// of the updated ref of the repo.
	UpdatedPath string

	// IsRoot is true if the package is the root, i.e. the clones of
	// updated and origin were fetched based on the information in the
	// Kptfile from this package.
	IsRoot bool

	// DryRun configures AlphaGitPatch to print a patch rather
	// than apply it
	DryRun bool

	// Verbose configures updaters to write verbose output
	Verbose bool

	// SimpleMessage is used for testing so commit messages in patches
	// don't contain the names of generated paths
	SimpleMessage bool

	Output io.Writer
}

// Updater updates a local package
type Updater interface {
	Update(options UpdateOptions) error
}

var strategies = map[kptfilev1alpha2.UpdateStrategyType]func() Updater{
	kptfilev1alpha2.FastForward:        func() Updater { return FastForwardUpdater{} },
	kptfilev1alpha2.ForceDeleteReplace: func() Updater { return ReplaceUpdater{} },
	kptfilev1alpha2.ResourceMerge:      func() Updater { return ResourceMergeUpdater{} },
}

// Command updates the contents of a local package to a different version.
type Command struct {
	// Pkg captures information about the package that should be updated.
	Pkg *pkg.Pkg

	// Ref is the ref to update to
	Ref string

	// Repo is the repo to update to
	Repo string

	// Strategy is the update strategy to use
	Strategy kptfilev1alpha2.UpdateStrategyType

	// DryRun if set will print the patch instead of applying it
	DryRun bool

	// Verbose if set will print verbose information about the commands being run
	Verbose bool

	// SimpleMessage if set will create simple git commit messages that omit values
	// generated for tests
	SimpleMessage bool

	// Output is where dry-run information is written
	Output io.Writer
}

// Run runs the Command.
func (u Command) Run(ctx context.Context) error {
	const op errors.Op = "update.Run"
	if u.Output == nil {
		u.Output = os.Stdout
	}

	if u.Pkg == nil {
		return errors.E(op, errors.MissingParam, "pkg must be provided")
	}

	// require package is checked into git before trying to update it
	g, err := gitutil.NewLocalGitRunner(u.Pkg.UniquePath.String())
	if err != nil {
		return err
	}

	rr, err := g.Run(ctx, "status", "-s")
	if err != nil {
		return errors.E(op, u.Pkg.UniquePath, err)
	}
	if strings.TrimSpace(rr.Stdout) != "" {
		return errors.E(op, u.Pkg.UniquePath, fmt.Errorf("package must be committed "+
			"to git before attempting to update"))
	}

	rootKf, err := u.Pkg.Kptfile()
	if err != nil {
		return errors.E(op, u.Pkg.UniquePath, err)
	}

	if rootKf.Upstream == nil || rootKf.Upstream.Git == nil {
		return errors.E(op, u.Pkg.UniquePath,
			fmt.Errorf("package must have an upstream reference"))
	}
	if u.Repo != "" {
		rootKf.Upstream.Git.Repo = u.Repo
	}
	if u.Ref != "" {
		rootKf.Upstream.Git.Ref = u.Ref
	}
	if u.Strategy != "" {
		rootKf.Upstream.UpdateStrategy = u.Strategy
	}
	err = kptfileutil.WriteFile(u.Pkg.UniquePath.String(), *rootKf)
	if err != nil {
		return errors.E(op, u.Pkg.UniquePath, err)
	}

	// Use stack to keep track of paths with a Kptfile that might contain
	// information about remote subpackages.
	s := stack.NewPkgStack()
	s.Push(u.Pkg)

	for s.Len() > 0 {
		p := s.Pop()

		if err := u.updateRootPackage(ctx, p); err != nil {
			return errors.E(op, p.UniquePath, err)
		}

		subPkgs, err := p.DirectSubpackages()
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}
		for _, subPkg := range subPkgs {
			s.Push(subPkg)
		}
	}
	return nil
}

// repoClone is an interface that represents a clone of a repo on the local
// disk.
type repoClone interface {
	AbsPath() string
}

// newNilRepoClone creates a new nilRepoClone that implements the repoClone
// interface
func newNilRepoClone() (*nilRepoClone, error) {
	const op errors.Op = "update.newNilRepoClone"
	dir, err := ioutil.TempDir("", "kpt-empty-")
	if err != nil {
		return nil, errors.E(op, errors.IO, fmt.Errorf("errors creating a temporary directory: %w", err))
	}
	return &nilRepoClone{
		dir: dir,
	}, nil
}

// nilRepoClone is an implementation of the repoClone interface, but that
// just represents an empty directory. This simplifies the logic for update
// since we don't have to special case situations where we don't have
// upstream and/or origin.
type nilRepoClone struct {
	dir string
}

// AbsPath returns the absolute path to the local directory for the repo. For
// the nilRepoClone, this will always be an empty directory.
func (nrc *nilRepoClone) AbsPath() string {
	return nrc.dir
}

// updateRootPackage updates a local package. It will use the information
// about upstream in the Kptfile to fetch upstream and origin, and then
// recursively traverse the hierarchy to add/update/delete packages.
func (u Command) updateRootPackage(ctx context.Context, p *pkg.Pkg) error {
	const op errors.Op = "update.updateRootPackage"
	kf, err := p.Kptfile()
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}

	if kf.Upstream == nil || kf.Upstream.Git == nil {
		return nil
	}

	g := kf.Upstream.Git

	pr := printer.FromContextOrDie(ctx)
	pr.Printf("updating package %s from %s/%s@%s\n",
		filepath.Base(p.UniquePath.String()), g.Repo, g.Directory, g.Ref)

	updated := &git.RepoSpec{OrgRepo: g.Repo, Path: g.Directory, Ref: g.Ref}
	if err := fetch.ClonerUsingGitExec(ctx, updated); err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	defer os.RemoveAll(updated.AbsPath())

	var origin repoClone
	if kf.UpstreamLock != nil {
		gLock := kf.UpstreamLock.GitLock
		originRepoSpec := &git.RepoSpec{OrgRepo: gLock.Repo, Path: gLock.Directory, Ref: gLock.Commit}
		if err := fetch.ClonerUsingGitExec(ctx, originRepoSpec); err != nil {
			return errors.E(op, p.UniquePath, err)
		}
		origin = originRepoSpec
	} else {
		origin, err = newNilRepoClone()
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}
	}
	defer os.RemoveAll(origin.AbsPath())

	s := stack.New()
	s.Push(".")

	for s.Len() > 0 {
		relPath := s.Pop()
		localPath := filepath.Join(p.UniquePath.String(), relPath)
		updatedPath := filepath.Join(updated.AbsPath(), relPath)
		originPath := filepath.Join(origin.AbsPath(), relPath)

		isRoot := false
		if relPath == "." {
			isRoot = true
		} else {
			pr.Printf("updating subpackage %s\n", relPath)
		}

		if err := u.updatePackage(relPath, localPath, updatedPath, originPath, isRoot); err != nil {
			return errors.E(op, p.UniquePath, err)
		}

		paths, err := pkgutil.FindSubpackagesForPaths(pkg.Remote, false,
			localPath, updatedPath, originPath)
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}
		for _, path := range paths {
			s.Push(filepath.Join(relPath, path))
		}
	}

	if err := kptfileutil.UpdateUpstreamLockFromGit(p.UniquePath.String(), updated); err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	return nil
}

// updatePackage takes care of updating a single package. The absolute paths to
// the local, updated and origin packages are provided, as well as the path to the
// package relative to the root.
// The last parameter tells if this package is the root, i.e. the package
// from which we got the information about upstream and origin.
//nolint:gocyclo
func (u Command) updatePackage(subPkgPath, localPath, updatedPath, originPath string, isRootPkg bool) error {
	const op errors.Op = "update.updatePackage"
	localExists, err := pkg.IsPackageDir(localPath)
	if err != nil {
		return errors.E(op, types.UniquePath(localPath), err)
	}

	// We need to handle the root package special here, since the copies
	// from updated and origin might not have a Kptfile at the root.
	updatedExists := isRootPkg
	if !isRootPkg {
		updatedExists, err = pkg.IsPackageDir(updatedPath)
		if err != nil {
			return errors.E(op, types.UniquePath(localPath), err)
		}
	}

	originExists := isRootPkg
	if !isRootPkg {
		originExists, err = pkg.IsPackageDir(originPath)
		if err != nil {
			return errors.E(op, types.UniquePath(localPath), err)
		}
	}

	switch {
	case !originExists && !localExists && !updatedExists:
		break
	// Check if subpackage has been added both in upstream and in local. We
	// can't make a sane merge here, so we treat it as an error.
	case !originExists && localExists && updatedExists:
		return errors.E(op, types.UniquePath(localPath),
			fmt.Errorf("subpackage %q added in both upstream and local", subPkgPath))

	// Package added in upstream. We just copy the package. If the package
	// contains any unfetched subpackages, those will be handled when we traverse
	// the package hierarchy and that package is the root.
	case !originExists && !localExists && updatedExists:
		if err := pkgutil.CopyPackage(updatedPath, localPath, !isRootPkg); err != nil {
			return errors.E(op, types.UniquePath(localPath), err)
		}

	// Package added locally, so no action needed.
	case !originExists && localExists && !updatedExists:
		break

	// Package deleted from both upstream and local, so no action needed.
	case originExists && !localExists && !updatedExists:
		break

	// Package deleted from local
	// In this case we assume the user knows what they are doing, so
	// we don't re-add the updated package from upstream.
	case originExists && !localExists && updatedExists:
		break
	// Package deleted from upstream
	case originExists && localExists && !updatedExists:
		// Check the diff. If there are local changes, we keep the subpackage.
		diff, err := copyutil.Diff(originPath, localPath)
		if err != nil {
			return errors.E(op, types.UniquePath(localPath), err)
		}
		if diff.Len() == 0 {
			if err := os.RemoveAll(localPath); err != nil {
				return errors.E(op, types.UniquePath(localPath), err)
			}
		}
	default:
		if err := u.mergePackage(localPath, updatedPath, originPath, subPkgPath, isRootPkg); err != nil {
			return errors.E(op, types.UniquePath(localPath), err)
		}
	}
	return nil
}

func (u Command) mergePackage(localPath, updatedPath, originPath, relPath string, isRootPkg bool) error {
	const op errors.Op = "update.mergePackage"
	updatedUnfetched, err := pkg.IsPackageUnfetched(updatedPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) || !isRootPkg {
			return errors.E(op, types.UniquePath(localPath), err)
		}
		// For root packages, there might not be a Kptfile in the upstream repo.
		updatedUnfetched = false
	}

	originUnfetched, err := pkg.IsPackageUnfetched(originPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) || !isRootPkg {
			return errors.E(op, types.UniquePath(localPath), err)
		}
		// For root packages, there might not be a Kptfile in origin.
		originUnfetched = false
	}

	switch {
	case updatedUnfetched && originUnfetched:
		fallthrough
	case updatedUnfetched && !originUnfetched:
		// updated is unfetched, so can't have changes except for Kptfile.
		// we can just merge that one.
		return kptfileutil.UpdateKptfile(localPath, updatedPath, originPath, true)
	case !updatedUnfetched && originUnfetched:
		// This means that the package was unfetched when local forked from upstream,
		// so the local fork and upstream might have fetched different versions of
		// the package. We just return an error here.
		// We might be able to compare the commit SHAs from local and updated
		// to determine if they share the common upstream and then fetch origin
		// using the common commit SHA. But this is a very advanced scenario,
		// so we just return the error for now.
		return errors.E(op, types.UniquePath(localPath), fmt.Errorf("no origin available for package"))
	default:
		// Both exists, so just go ahead as normal.
	}

	pkgKf, err := kptfileutil.ReadFile(localPath)
	if err != nil {
		return errors.E(op, types.UniquePath(localPath), err)
	}
	updater, found := strategies[pkgKf.Upstream.UpdateStrategy]
	if !found {
		return errors.E(op, types.UniquePath(localPath),
			fmt.Errorf("unrecognized update strategy %s", u.Strategy))
	}
	if err := updater().Update(UpdateOptions{
		RelPackagePath: relPath,
		LocalPath:      localPath,
		UpdatedPath:    updatedPath,
		OriginPath:     originPath,
		IsRoot:         isRootPkg,
		DryRun:         u.DryRun,
		Verbose:        u.Verbose,
		SimpleMessage:  u.SimpleMessage,
		Output:         u.Output,
	}); err != nil {
		return errors.E(op, types.UniquePath(localPath), err)
	}
	return nil
}
