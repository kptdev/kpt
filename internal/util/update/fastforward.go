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
	"os"
	"path/filepath"

	pkgdiff "github.com/GoogleContainerTools/kpt/internal/util/diff"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
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

// We should try to pull the common code up into the Update command.
func (u FastForwardUpdater) Update(options UpdateOptions) error {
	localPath := filepath.Join(options.LocalPath, options.RelPackagePath)
	originalPath := filepath.Join(options.OriginPath, options.RelPackagePath)

	// Verify that there are no local changes that would prevent us from
	// using the FastForward strategy.
	if err := u.checkForLocalChanges(localPath, originalPath); err != nil {
		return err
	}

	return (&ReplaceUpdater{}).Update(options)
}

func (u FastForwardUpdater) checkForLocalChanges(localPath, originalPath string) error {
	found, err := pkgutil.Exists(originalPath)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	subPkgPaths, err := pkgutil.FindLocalRecursiveSubpackagesForPaths(localPath, originalPath)
	if err != nil {
		return err
	}
	aggDiff := sets.String{}
	for _, subPkgPath := range subPkgPaths {
		localSubPkgPath := filepath.Join(localPath, subPkgPath)
		originalSubPkgPath := filepath.Join(originalPath, subPkgPath)

		localExists, err := pkgutil.Exists(localSubPkgPath)
		if err != nil {
			return err
		}
		originalExists, err := pkgutil.Exists(originalSubPkgPath)
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
	localKf.Upstream = nil
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
