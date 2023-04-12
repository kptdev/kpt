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

package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	pkgdiff "github.com/GoogleContainerTools/kpt/internal/util/diff"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

// Updater updates a package to a new upstream version.
//
// If the package at pkgPath differs from the upstream ref it was fetch from, then Update will
// fail without making any changes.
type FastForwardUpdater struct{}

var kptfileSet = func() sets.String {
	s := sets.String{}
	s.Insert(kptfilev1.KptFileName)
	return s
}()

// We should try to pull the common code up into the Update command.
func (u FastForwardUpdater) Update(options Options) error {
	const op errors.Op = "update.Update"
	// Verify that there are no local changes that would prevent us from
	// using the FastForward strategy.
	if err := u.checkForLocalChanges(options.LocalPath, options.OriginPath); err != nil {
		return errors.E(op, types.UniquePath(options.LocalPath), err)
	}

	if err := (&ReplaceUpdater{}).Update(options); err != nil {
		return errors.E(op, types.UniquePath(options.LocalPath), err)
	}
	return nil
}

func (u FastForwardUpdater) checkForLocalChanges(localPath, originalPath string) error {
	const op errors.Op = "update.checkForLocalChanges"
	found, err := pkgutil.Exists(originalPath)
	if err != nil {
		return errors.E(op, types.UniquePath(localPath), err)
	}
	if !found {
		return nil
	}

	subPkgPaths, err := pkgutil.FindSubpackagesForPaths(pkg.Local, true, localPath, originalPath)
	if err != nil {
		return errors.E(op, types.UniquePath(localPath), err)
	}
	aggDiff := sets.String{}
	for _, subPkgPath := range append([]string{"."}, subPkgPaths...) {
		localSubPkgPath := filepath.Join(localPath, subPkgPath)
		originalSubPkgPath := filepath.Join(originalPath, subPkgPath)

		localExists, err := pkgutil.Exists(localSubPkgPath)
		if err != nil {
			return errors.E(op, types.UniquePath(localSubPkgPath), err)
		}
		originalExists, err := pkgutil.Exists(originalSubPkgPath)
		if err != nil {
			return errors.E(op, types.UniquePath(localSubPkgPath), err)
		}
		if !originalExists || !localExists {
			aggDiff.Insert("%s (Package)", subPkgPath)
			continue
		}
		d, err := pkgdiff.PkgDiff(localSubPkgPath, originalSubPkgPath)
		if err != nil {
			return errors.E(op, types.UniquePath(localSubPkgPath), err)
		}
		// If the original package didn't have a Kptfile, one was created
		// in local, but we don't consider that a change unless the user
		// has made additional changes.
		if d.Has(kptfilev1.KptFileName) && subPkgPath == "." {
			hasDiff, err := hasKfDiff(localSubPkgPath, originalSubPkgPath)
			if err != nil {
				return errors.E(op, types.UniquePath(localSubPkgPath), err)
			}
			if !hasDiff {
				d = d.Difference(kptfileSet)
			}
		}

		aggDiff.Insert(d.List()...)
	}
	if aggDiff.Len() > 0 {
		return errors.E(op, types.UniquePath(localPath), fmt.Sprintf(
			"local package files have been modified: %v.\n  use a different update --strategy.",
			aggDiff.List()))
	}
	return nil
}

func hasKfDiff(localPath, orgPath string) (bool, error) {
	const op errors.Op = "update.hasKfDiff"
	localKf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, localPath)
	if err != nil {
		return false, errors.E(op, types.UniquePath(localPath), err)
	}
	localKf.Upstream = nil
	localKf.UpstreamLock = nil

	_, err = os.Stat(filepath.Join(orgPath, kptfilev1.KptFileName))
	if err != nil {
		if os.IsNotExist(err) {
			// We know that there aren't any Kptfile in the original
			// package, so we ignore the diff if the local Kptfile
			// is just the minimal Kptfile generated automatically.
			isDefault, err := isDefaultKptfile(localKf, filepath.Base(localPath))
			if err != nil {
				return false, errors.E(op, types.UniquePath(localPath), err)
			}
			return !isDefault, nil
		}
		return false, errors.E(op, types.UniquePath(localPath), err)
	}
	orgKf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, orgPath)
	if err != nil {
		return false, errors.E(op, types.UniquePath(localPath), err)
	}

	orgKf.Name = localKf.Name
	equal, err := kptfileutil.Equal(localKf, orgKf)
	if err != nil {
		return false, errors.E(op, types.UniquePath(localPath), err)
	}

	return !equal, nil
}

func isDefaultKptfile(localKf *kptfilev1.KptFile, name string) (bool, error) {
	defaultKf := kptfileutil.DefaultKptfile(name)
	return kptfileutil.Equal(localKf, defaultKf)
}
