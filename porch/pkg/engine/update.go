// Copyright 2022 The kpt Authors
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

package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
)

// packageUpdater knows how to update a local package given original and upstream package resources.
type packageUpdater interface {
	Update(ctx context.Context, localResources, originalResources, upstreamResources repository.PackageResources) (updatedResources repository.PackageResources, err error)
}

// defaultPackageUpdater implements packageUpdater interface.
type defaultPackageUpdater struct{}

func (m *defaultPackageUpdater) Update(
	ctx context.Context,
	localResources,
	originalResources,
	upstreamResources repository.PackageResources) (updatedResources repository.PackageResources, err error) {

	localDir, err := os.MkdirTemp("", "kpt-pkg-update-*")
	if err != nil {
		return repository.PackageResources{}, err
	}
	defer os.RemoveAll(localDir)

	originalDir, err := os.MkdirTemp("", "kpt-pkg-update-*")
	if err != nil {
		return repository.PackageResources{}, err
	}
	defer os.RemoveAll(originalDir)

	upstreamDir, err := os.MkdirTemp("", "kpt-pkg-update-*")
	if err != nil {
		return repository.PackageResources{}, err
	}
	defer os.RemoveAll(upstreamDir)

	if err := writeResourcesToDirectory(localDir, localResources); err != nil {
		return repository.PackageResources{}, err
	}

	if err := writeResourcesToDirectory(originalDir, originalResources); err != nil {
		return repository.PackageResources{}, err
	}

	if err := writeResourcesToDirectory(upstreamDir, upstreamResources); err != nil {
		return repository.PackageResources{}, err
	}

	if err := m.do(ctx, localDir, originalDir, upstreamDir); err != nil {
		return repository.PackageResources{}, err
	}

	return loadResourcesFromDirectory(localDir)
}

// PkgUpdate is a wrapper around `kpt pkg update`, running it against the package in packageDir
func (m *defaultPackageUpdater) do(ctx context.Context, localPkgDir, originalPkgDir, upstreamPkgDir string) error {
	// TODO: Printer should be a logr
	pr := printer.New(os.Stdout, os.Stderr)
	ctx = printer.WithContext(ctx, pr)

	{
		relPath := "."
		localPath := filepath.Join(localPkgDir, relPath)
		updatedPath := filepath.Join(upstreamPkgDir, relPath)
		originPath := filepath.Join(originalPkgDir, relPath)
		isRoot := true

		updateOptions := update.Options{
			RelPackagePath: relPath,
			LocalPath:      localPath,
			UpdatedPath:    updatedPath,
			OriginPath:     originPath,
			IsRoot:         isRoot,
		}
		updater := update.ResourceMergeUpdater{}
		if err := updater.Update(updateOptions); err != nil {
			return err
		}
	}
	return nil
}
