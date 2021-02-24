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

package update

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	pkgdiff "github.com/GoogleContainerTools/kpt/internal/util/diff"
	"github.com/GoogleContainerTools/kpt/internal/util/fetch"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

// Updater updates a package to a new upstream version.
//
// If the package at pkgPath differs from the upstream ref it was fetch from, then Update will
// fail without making any changes.
type FastForwardUpdater struct{}

var kptfileSet = func() sets.String {
	s := sets.String{}
	s.Insert(kptfilev1alpha2.KptFileName)
	return s
}()

// TODO(mortent): There is duplicate code between the different update strategies.
// We should try to pull the common code up into the Update command.
func (u FastForwardUpdater) Update(options UpdateOptions) error {
	g := options.KptFile.UpstreamLock.GitLock
	g.Ref = options.ToRef
	g.Repo = options.ToRepo

	// get the original repo
	original := &git.RepoSpec{OrgRepo: g.Repo, Path: g.Directory, Ref: g.Commit}
	if err := fetch.ClonerUsingGitExec(original); err != nil {
		return errors.Errorf("failed to clone git repo: original source: %v", err)
	}
	defer os.RemoveAll(original.AbsPath())

	// get the updated repo
	updated := &git.RepoSpec{OrgRepo: options.ToRepo, Path: g.Directory, Ref: options.ToRef}
	if err := fetch.ClonerUsingGitExec(updated); err != nil {
		return errors.Errorf("failed to clone git repo: updated source: %v", err)
	}
	defer os.RemoveAll(updated.AbsPath())

	// Verify that there are no local changes that would prevent us from
	// using the FastForward strategy.
	if err := u.checkForLocalChanges(options.AbsPackagePath, original.AbsPath()); err != nil {
		return err
	}

	// Look up all subpackages across the local package and the updated (from upstream)
	// package.
	subPkgPaths, err := findAllSubpackages(options.AbsPackagePath, updated.AbsPath())
	if err != nil {
		return err
	}

	// Update each package individually, starting with the root package.
	for _, subPkgPath := range subPkgPaths {
		localSubPkgPath := filepath.Join(options.AbsPackagePath, subPkgPath)
		updatedSubPkgPath := filepath.Join(updated.AbsPath(), subPkgPath)

		// Walk the package (while ignoring subpackages) and delete all files.
		// We capture the paths to any subdirectories in the package so we
		// can handle those later. We can't do it while walking the package
		// since we don't want to end up deleting directories that might
		// contain a nested subpackage.
		var dirs []string
		if err := pkgutil.WalkPackage(localSubPkgPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}

			if info.IsDir() {
				if path != localSubPkgPath {
					dirs = append(dirs, path)
				}
				return nil
			}
			return os.Remove(path)
		}); err != nil {
			return err
		}

		// Delete any of the directories in the package that are
		// empty. We start with the most deeply nested directories
		// so we can just check every directory for files/directories.
		sort.Slice(dirs, subPkgFirstSorter(dirs))
		for _, p := range dirs {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			// List up to one file or folder in the directory.
			_, err = f.Readdirnames(1)
			if err != nil && err != io.EOF {
				return err
			}
			// If the returned error is EOF, it means the folder
			// was empty and we can remove it.
			if err == io.EOF {
				err = os.RemoveAll(p)
				if err != nil {
					return err
				}
			}
		}

		// If the package doesn't exist in updated, we make sure it is
		// deleted from the local package. If it exists in updated, we copy
		// the content of the package into local.
		_, err = os.Stat(updatedSubPkgPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if os.IsNotExist(err) {
			if err = os.RemoveAll(localSubPkgPath); err != nil {
				return err
			}
		} else {
			if err = pkgutil.CopyPackage(updatedSubPkgPath, localSubPkgPath); err != nil {
				return err
			}
		}
	}

	return fetch.UpsertKptfile(options.AbsPackagePath, filepath.Base(options.AbsPackagePath), updated)
}

func (u FastForwardUpdater) checkForLocalChanges(localPath, originalPath string) error {
	subPkgPaths, err := findAllSubpackages(localPath, originalPath)
	if err != nil {
		return err
	}
	aggDiff := sets.String{}
	for _, subPkgPath := range subPkgPaths {
		localSubPkgPath := filepath.Join(localPath, subPkgPath)
		originalSubPkgPath := filepath.Join(originalPath, subPkgPath)

		localExists, err := exists(localSubPkgPath)
		if err != nil {
			return err
		}
		originalExists, err := exists(originalSubPkgPath)
		if err != nil {
			return err
		}
		if !originalExists || !localExists {
			aggDiff.Insert("%s (Package)", subPkgPath)
			continue
		}
		d, err := pkgdiff.PkgDiff(localSubPkgPath, originalSubPkgPath)
		if err != nil {
			return err
		}
		// If the original package didn't have a Kptfile, one was created
		// in local, but we don't consider that a change unless the user
		// has made additional changes.
		if d.Has(kptfilev1alpha2.KptFileName) && subPkgPath == "." {
			hasDiff, err := hasKfDiff(localSubPkgPath, originalSubPkgPath)
			if err != nil {
				return err
			}
			if !hasDiff {
				d = d.Difference(kptfileSet)
			}
		}

		aggDiff.Insert(d.List()...)
	}
	if aggDiff.Len() > 0 {
		return DiffError(fmt.Sprintf(
			"local package files have been modified: %v.\n  use a different update --strategy.",
			aggDiff.List()))
	}
	return nil
}

func hasKfDiff(localPath, orgPath string) (bool, error) {
	localKf, err := kptfileutil.ReadFile(localPath)
	if err != nil {
		return false, err
	}
	localKf.UpstreamLock = nil

	_, err = os.Stat(filepath.Join(orgPath, kptfilev1alpha2.KptFileName))
	if err != nil {
		if os.IsNotExist(err) {
			// We know that there aren't any Kptfile in the original
			// package, so we ignore the diff if the local Kptfile
			// is just the minimal Kptfile generated automatically.
			isDefault, err := isDefaultKptfile(localKf, filepath.Base(localPath))
			if err != nil {
				return false, err
			}
			return !isDefault, nil
		}
		return false, err
	}
	orgKf, err := kptfileutil.ReadFile(orgPath)
	if err != nil {
		return false, err
	}

	orgKf.Name = localKf.Name
	equal, err := kptfileutil.Equal(localKf, orgKf)
	if err != nil {
		return false, err
	}

	return !equal, nil
}

func isDefaultKptfile(localKf kptfilev1alpha2.KptFile, name string) (bool, error) {
	defaultKf := kptfileutil.DefaultKptfile(name)
	return kptfileutil.Equal(localKf, defaultKf)
}

// DiffError is returned if the local package and upstream package contents do not match.
type DiffError string

func (d DiffError) Error() string {
	return string(d)
}
