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
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/addmergecomment"
	"github.com/GoogleContainerTools/kpt/internal/util/copyutil"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	"github.com/GoogleContainerTools/kpt/internal/util/stack"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/content/open"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/GoogleContainerTools/kpt/pkg/location/mutate"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/GoogleContainerTools/kpt/internal/migration/os"
	"github.com/GoogleContainerTools/kpt/internal/migration/path/filepath"
)

// PkgNotGitRepoError is the error type returned if the package being updated is not inside
// a git repository.
type PkgNotGitRepoError struct {
	Path types.UniquePath
}

func (p *PkgNotGitRepoError) Error() string {
	return fmt.Sprintf("package %q is not a git repository", p.Path.String())
}

// PkgRepoDirtyError is the error type returned if the package being updated contains
// uncommitted changes.
type PkgRepoDirtyError struct {
	Path types.UniquePath
}

func (p *PkgRepoDirtyError) Error() string {
	return fmt.Sprintf("package %q contains uncommitted changes", p.Path.String())
}

type UpdateOptions struct {
	// RelPackagePath is the relative path of a subpackage to the root. If the
	// package is root, the value here will be ".".
	RelPackagePath string

	// LocalPath is the absolute path to the package on the local fork.
	LocalPath types.FileSystemPath

	// OriginPath is the absolute path to the package in the on-disk clone
	// of the origin ref of the repo.
	OriginPath types.FileSystemPath

	// UpdatedPath is the absolute path to the package in the on-disk clone
	// of the updated ref of the repo.
	UpdatedPath types.FileSystemPath

	// IsRoot is true if the package is the root, i.e. the clones of
	// updated and origin were fetched based on the information in the
	// Kptfile from this package.
	IsRoot bool
}

// Updater updates a local package
type Updater interface {
	Update(options UpdateOptions) error
}

var strategies = map[kptfilev1.UpdateStrategyType]func() Updater{
	kptfilev1.FastForward:        func() Updater { return FastForwardUpdater{} },
	kptfilev1.ForceDeleteReplace: func() Updater { return ReplaceUpdater{} },
	kptfilev1.ResourceMerge:      func() Updater { return ResourceMergeUpdater{} },
}

// Command updates the contents of a local package to a different version.
type Command struct {
	// Pkg captures information about the package that should be updated.
	Pkg *pkg.Pkg

	// Ref is the ref to update to
	Ref string

	// Strategy is the update strategy to use
	Strategy kptfilev1.UpdateStrategyType
}

// Run runs the Command.
func (u Command) Run(ctx context.Context) error {
	const op errors.Op = "update.Run"
	pr := printer.FromContextOrDie(ctx)

	if u.Pkg == nil {
		return errors.E(op, errors.MissingParam, "pkg must be provided")
	}

	rootKf, err := u.Pkg.Kptfile()
	if err != nil {
		return errors.E(op, u.Pkg.UniquePath, err)
	}

	rootUpstream, err := kptfileutil.NewReferenceFromUpstream(rootKf)
	if err != nil {
		return errors.E(op, u.Pkg.UniquePath, fmt.Errorf("package must have an upstream reference: %v", err))
	}

	originalRootUpstream := rootUpstream
	if u.Ref != "" {
		rootUpstream, err = mutate.Identifier(rootUpstream, u.Ref)
		if err != nil {
			return errors.E(op, u.Pkg.UniquePath, err)
		}
	}
	strategy := u.Strategy
	if strategy == "" {
		strategy = rootKf.Upstream.UpdateStrategy
	}

	rootKf.Upstream, err = kptfileutil.NewUpstreamFromReference(rootUpstream, strategy)
	if err != nil {
		return errors.E(op, u.Pkg.UniquePath, err)
	}

	err = kptfileutil.WriteFile(u.Pkg.UniquePath.String(), rootKf)
	if err != nil {
		return errors.E(op, u.Pkg.UniquePath, err)
	}

	packageCount := 0

	// Use stack to keep track of paths with a Kptfile that might contain
	// information about remote subpackages.
	s := stack.NewPkgStack()
	s.Push(u.Pkg)

	for s.Len() > 0 {
		p := s.Pop()
		packageCount += 1

		if err := u.updateRootPackage(ctx, p); err != nil {
			return errors.E(op, p.UniquePath, err)
		}

		subPkgs, err := p.DirectSubpackages()
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}
		for _, subPkg := range subPkgs {
			subKf, err := subPkg.Kptfile()
			if err != nil {
				return errors.E(op, p.UniquePath, err)
			}

			if subUps, err := kptfileutil.NewReferenceFromUpstream(subKf); err == nil {
				// update subpackage kf ref/strategy if current pkg is a subpkg of root pkg or is root pkg
				// and if original root pkg ref matches the subpkg ref
				if shouldUpdateSubPkgRef(originalRootUpstream, subUps) {
					if err := updateSubKf(subKf, subUps, u.Ref, u.Strategy); err != nil {
						return errors.E(op, subPkg.UniquePath, err)
					}
					if err = kptfileutil.WriteFile(subPkg.UniquePath.String(), subKf); err != nil {
						return errors.E(op, subPkg.UniquePath, err)
					}
				}
				s.Push(subPkg)
			}
		}
	}
	pr.Printf("\nUpdated %d package(s).\n", packageCount)

	// finally, make sure that the merge comments are added to all resources in the updated package
	if err := addmergecomment.ProcessObsolete(string(u.Pkg.UniquePath)); err != nil {
		return errors.E(op, u.Pkg.UniquePath, err)
	}
	return nil
}

func shouldUpdateSubPkgRef(rootPkgUpstream, subPkgUpstream location.Reference) bool {
	rel, err := location.Rel(rootPkgUpstream, subPkgUpstream)
	if err != nil {
		// locations are unrelated, e.g. different type, repo, ref, etc.
		return false
	}
	// should update if sub-package points to directory of root location
	return !strings.HasPrefix(rel, "../")
}

// updateSubKf updates subpackage with given ref and update strategy
func updateSubKf(subKf *kptfilev1.KptFile, subUpstream location.Reference, ref string, strategy kptfilev1.UpdateStrategyType) error {
	// check if explicit ref provided
	if ref != "" {
		new, err := mutate.Identifier(subUpstream, ref)
		if err != nil {
			return err
		}
		subUpstream = new
	}

	// keep existing strategy if not provided
	if strategy == "" {
		strategy = subKf.Upstream.UpdateStrategy
	}

	// create and assign
	new, err := kptfileutil.NewUpstreamFromReference(subUpstream, strategy)
	if err != nil {
		return err
	}
	subKf.Upstream = new
	return nil
}

// updateRootPackage updates a local package. It will use the information
// about upstream in the Kptfile to fetch upstream and origin, and then
// recursively traverse the hierarchy to add/update/delete packages.
func (u Command) updateRootPackage(ctx context.Context, p *pkg.Pkg) error {
	const op errors.Op = "update.updateRootPackage"
	opts := open.Options(open.WithContext(ctx))

	kf, err := p.Kptfile()
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}

	var localAbsPath, updatedAbsPath, originAbsPath types.FileSystemPath

	pr := printer.FromContextOrDie(ctx)
	pr.PrintPackage(p, !(p == u.Pkg))

	ref, err := kptfileutil.NewReferenceFromUpstream(kf)
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}

	localAbsPath = types.DiskPath(p.UniquePath.String())

	pr.Printf("Fetching upstream from %s\n", ref)
	updated, err := open.FileSystem(ref, opts)
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	defer updated.Close()
	updatedAbsPath = updated.FileSystemPath

	if kf.UpstreamLock != nil {
		lock, err := kptfileutil.NewReferenceLockFromUpstreamLock(kf)
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}

		pr.Printf("Fetching origin from %s\n", lock)
		origin, err := open.FileSystem(lock, opts)
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}
		defer origin.Close()

		originAbsPath = origin.FileSystemPath
	} else {
		// point to empty virtual path in a way that
		// supports filesystem operations
		originAbsPath = types.FileSystemPath{
			FileSystem: filesys.MakeFsInMemory(),
			Path:       "/nil",
		} 
		if err := os.MkdirAll(originAbsPath, os.ModePerm); err != nil {
			return errors.E(op, p.UniquePath, err)
		}
	}

	s := stack.New()
	s.Push(".")

	for s.Len() > 0 {
		relPath := s.Pop()
		localPath := types.Join(localAbsPath, relPath)
		updatedPath := types.Join(updatedAbsPath, relPath)
		originPath := types.Join(originAbsPath, relPath)

		isRoot := false
		if relPath == "." {
			isRoot = true
		}

		if err := u.updatePackage(ctx, relPath, localPath, updatedPath, originPath, isRoot); err != nil {
			return errors.E(op, p.UniquePath, err)
		}

		paths, err := pkgutil.FindSubpackagesForPaths(pkg.Remote, false,
			localPath, updatedPath, originPath)
		if err != nil {
			return errors.E(op, p.UniquePath, err)
		}
		for _, path := range paths {
			s.Push(filepath.JoinRel(relPath, path))
		}
	}

	if err := kptfileutil.UpdateUpstreamLock(localAbsPath, updated.ReferenceLock); err != nil {
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
func (u Command) updatePackage(ctx context.Context, subPkgPath string, localPath, updatedPath, originPath types.FileSystemPath, isRootPkg bool) error {
	const op errors.Op = "update.updatePackage"
	pr := printer.FromContextOrDie(ctx)

	localExists, err := pkg.IsPackageDir(localPath.FileSystem, localPath.Path)
	if err != nil {
		return errors.E(op, types.AsUniquePath(localPath), err)
	}

	// We need to handle the root package special here, since the copies
	// from updated and origin might not have a Kptfile at the root.
	updatedExists := isRootPkg
	if !isRootPkg {
		updatedExists, err = pkg.IsPackageDir(updatedPath.FileSystem, updatedPath.Path)
		if err != nil {
			return errors.E(op, types.UniquePath(updatedPath.String()), err)
		}
	}

	originExists := isRootPkg
	if !isRootPkg {
		originExists, err = pkg.IsPackageDir(originPath.FileSystem, originPath.Path)
		if err != nil {
			return errors.E(op, types.UniquePath(originPath.String()), err)
		}
	}

	switch {
	case !originExists && !localExists && !updatedExists:
		break
	// Check if subpackage has been added both in upstream and in local. We
	// can't make a sane merge here, so we treat it as an error.
	case !originExists && localExists && updatedExists:
		pr.Printf("Package %q added in both local and upstream.\n", packageName(localPath))
		return errors.E(op, types.AsUniquePath(localPath),
			fmt.Errorf("subpackage %q added in both upstream and local", subPkgPath))

	// Package added in upstream. We just copy the package. If the package
	// contains any unfetched subpackages, those will be handled when we traverse
	// the package hierarchy and that package is the root.
	case !originExists && !localExists && updatedExists:
		pr.Printf("Adding package %q from upstream.\n", packageName(localPath))
		if err := pkgutil.CopyPackage(updatedPath, localPath, !isRootPkg, pkg.None); err != nil {
			return errors.E(op, types.AsUniquePath(localPath), err)
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
		pr.Printf("Ignoring package %q in upstream since it is deleted from local.\n", packageName(localPath))

	// Package deleted from upstream
	case originExists && localExists && !updatedExists:
		// Check the diff. If there are local changes, we keep the subpackage.
		diff, err := copyutil.Diff(originPath, localPath)
		if err != nil {
			return errors.E(op, types.AsUniquePath(localPath), err)
		}
		if diff.Len() == 0 {
			pr.Printf("Deleting package %q from local since it is removed in upstream.\n", packageName(localPath))
			if err := os.RemoveAll(localPath); err != nil {
				return errors.E(op, types.AsUniquePath(localPath), err)
			}
		} else {
			pr.Printf("Package %q deleted from upstream, but keeping local since it has changes.\n", packageName(localPath))
		}
	default:
		if err := u.mergePackage(ctx, localPath, updatedPath, originPath, subPkgPath, isRootPkg); err != nil {
			return errors.E(op, types.AsUniquePath(localPath), err)
		}
	}
	return nil
}

func (u Command) mergePackage(ctx context.Context, localPath, updatedPath, originPath types.FileSystemPath, relPath string, isRootPkg bool) error {
	const op errors.Op = "update.mergePackage"
	pr := printer.FromContextOrDie(ctx)
	// at this point, the localPath, updatedPath and originPath exists and are about to be merged
	// make sure that the merge comments are added to all of them so that they are merged accurately
	if err := addmergecomment.Process(localPath, updatedPath, originPath); err != nil {
		return errors.E(op, types.AsUniquePath(localPath),
			fmt.Errorf("failed to add merge comments %q", err.Error()))
	}
	updatedUnfetched, err := pkg.IsPackageUnfetched(updatedPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) || !isRootPkg {
			return errors.E(op, types.AsUniquePath(localPath), err)
		}
		// For root packages, there might not be a Kptfile in the upstream repo.
		updatedUnfetched = false
	}

	originUnfetched, err := pkg.IsPackageUnfetched(originPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) || !isRootPkg {
			return errors.E(op, types.AsUniquePath(localPath), err)
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
		return errors.E(op, types.AsUniquePath(localPath), fmt.Errorf("no origin available for package"))
	default:
		// Both exists, so just go ahead as normal.
	}

	pkgKf, err := pkg.ReadKptfile(localPath)
	if err != nil {
		return errors.E(op, types.AsUniquePath(localPath), err)
	}
	updater, found := strategies[pkgKf.Upstream.UpdateStrategy]
	if !found {
		return errors.E(op, types.AsUniquePath(localPath),
			fmt.Errorf("unrecognized update strategy %s", u.Strategy))
	}
	pr.Printf("Updating package %q with strategy %q.\n", packageName(localPath), pkgKf.Upstream.UpdateStrategy)
	if err := updater().Update(UpdateOptions{
		RelPackagePath: relPath,
		LocalPath:      localPath,
		UpdatedPath:    updatedPath,
		OriginPath:     originPath,
		IsRoot:         isRootPkg,
	}); err != nil {
		return errors.E(op, types.AsUniquePath(localPath), err)
	}

	return nil
}

func packageName(path types.FileSystemPath) string {
	return filepath.Base(path)
}
