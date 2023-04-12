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
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
)

// Updater updates a package to a new upstream version.
//
// If the package at pkgPath differs from the upstream ref it was fetch from, then Update will
// delete the local package.  This will wipe all local changes.
type ReplaceUpdater struct{}

func (u ReplaceUpdater) Update(options Options) error {
	const op errors.Op = "update.Update"
	paths, err := pkgutil.FindSubpackagesForPaths(pkg.Local, true, options.LocalPath, options.UpdatedPath)
	if err != nil {
		return errors.E(op, types.UniquePath(options.LocalPath), err)
	}

	for _, p := range append([]string{"."}, paths...) {
		isRootPkg := false
		if p == "." && options.IsRoot {
			isRootPkg = true
		}
		localSubPkgPath := filepath.Join(options.LocalPath, p)
		updatedSubPkgPath := filepath.Join(options.UpdatedPath, p)
		err = pkgutil.RemovePackageContent(localSubPkgPath, !isRootPkg)
		if err != nil {
			return errors.E(op, types.UniquePath(localSubPkgPath), err)
		}

		// If the package doesn't exist in updated, we make sure it is
		// deleted from the local package. If it exists in updated, we copy
		// the content of the package into local.
		_, err = os.Stat(updatedSubPkgPath)
		if err != nil && !os.IsNotExist(err) {
			return errors.E(op, types.UniquePath(localSubPkgPath), err)
		}
		if os.IsNotExist(err) {
			if err = os.RemoveAll(localSubPkgPath); err != nil {
				return errors.E(op, types.UniquePath(localSubPkgPath), err)
			}
		} else {
			if err = pkgutil.CopyPackage(updatedSubPkgPath, localSubPkgPath, !isRootPkg, pkg.None); err != nil {
				return errors.E(op, types.UniquePath(localSubPkgPath), err)
			}
		}
	}
	return nil
}
