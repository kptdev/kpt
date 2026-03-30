// Copyright 2022-2026 The kpt Authors
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

// Package kptops contains implementations of kpt operations
package kptops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kptdev/kpt/internal/util/fetch"
	"github.com/kptdev/kpt/internal/util/git"
	"github.com/kptdev/kpt/internal/util/update"
	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/lib/update/updatetypes"
	"github.com/kptdev/kpt/pkg/printer"
	"k8s.io/klog/v2"
)

// Constants for package update operations
const (
	// KptfileName is the name of the kpt configuration file
	KptfileName = "Kptfile"
	// EmptyTempDirPrefix is the prefix for empty temporary directories
	EmptyTempDirPrefix = "kpt-empty-"
	// RootPackagePath represents the root package path
	RootPackagePath = "."
)

// PkgUpdateOpts are options for invoking kpt PkgUpdate.
type PkgUpdateOpts struct {
	// Strategy defines the update strategy to use. Currently unused but reserved for future implementation.
	Strategy string
}

// PkgUpdate updates a package from its upstream source.
// It fetches the latest version of the upstream package and merges changes with the local package.
//
// Parameters:
//   - ctx: Context for cancellation and logging
//   - ref: Git reference to update to (branch, tag, or commit). If empty, uses the current reference.
//   - packageDir: Path to the local package directory
//   - opts: Update options (currently only strategy placeholder)
//
// Returns an error if the update fails.
func PkgUpdate(ctx context.Context, ref string, packageDir string, opts PkgUpdateOpts) error {
	// Validate inputs
	if packageDir == "" {
		return fmt.Errorf("package directory cannot be empty")
	}

	// Initialize printer with proper context
	pr := printer.New(os.Stdout, os.Stderr)
	ctx = printer.WithContext(ctx, pr)

	// Load and validate package configuration
	kf, err := loadAndValidateKptfile(packageDir)
	if err != nil {
		return fmt.Errorf("failed to load package configuration: %w", err)
	}

	// Update reference if provided
	if ref != "" {
		kf.Upstream.Git.Ref = ref
	}

	// Save updated Kptfile
	if err = kptfileutil.WriteFile(packageDir, kf); err != nil {
		return fmt.Errorf("failed to write Kptfile: %w", err)
	}

	// Perform update based on upstream type
	if err := performUpdate(ctx, packageDir, kf); err != nil {
		return fmt.Errorf("failed to perform update: %w", err)
	}

	return nil
}

// loadAndValidateKptfile loads and validates the Kptfile from the package directory.
// It ensures the package has a valid upstream Git reference.
func loadAndValidateKptfile(packageDir string) (*kptfilev1.KptFile, error) {
	if packageDir == "" {
		return nil, fmt.Errorf("package directory cannot be empty")
	}

	fsys := os.DirFS(packageDir)

	f, err := fsys.Open(KptfileName)
	if err != nil {
		return nil, fmt.Errorf("error opening Kptfile: %w", err)
	}
	defer f.Close()

	kf, err := kptfileutil.DecodeKptfile(f)
	if err != nil {
		return nil, fmt.Errorf("error parsing Kptfile: %w", err)
	}

	if kf.Upstream == nil {
		return nil, fmt.Errorf("package must have an upstream reference")
	}

	if kf.Upstream.Git == nil {
		return nil, fmt.Errorf("package upstream must have Git configuration")
	}

	if kf.Upstream.Git.Repo == "" {
		return nil, fmt.Errorf("package upstream Git repository cannot be empty")
	}

	return kf, nil
}

// performUpdate handles the update process based on the upstream type.
// It delegates to the appropriate update implementation based on the upstream type.
func performUpdate(ctx context.Context, packageDir string, kf *kptfilev1.KptFile) error {
	if kf == nil {
		return fmt.Errorf("kptfile cannot be nil")
	}

	switch kf.Upstream.Type {
	case kptfilev1.GitOrigin:
		return updateFromGit(ctx, packageDir, kf)
	case kptfilev1.GenericOrigin:
		return fmt.Errorf("Generic origin updates are not yet implemented")
	default:
		return fmt.Errorf("unsupported upstream type: %s", kf.Upstream.Type)
	}
}

// updateFromGit performs update from a Git repository.
// It fetches both the upstream and origin repositories, then merges the changes.
func updateFromGit(ctx context.Context, packageDir string, kf *kptfilev1.KptFile) error {
	if kf.Upstream == nil || kf.Upstream.Git == nil {
		return fmt.Errorf("package must have a Git upstream reference")
	}

	// Fetch updated upstream
	updatedRepoSpec, updatedDir, err := fetchUpstreamGit(ctx, kf.Upstream.Git)
	if err != nil {
		return fmt.Errorf("failed to fetch upstream: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(updatedDir); cleanupErr != nil {
			klog.Warningf("Failed to cleanup updated directory %s: %v", updatedDir, cleanupErr)
		}
	}()

	// Fetch origin if available
	originDir, err := fetchOriginGit(ctx, kf.UpstreamLock)
	if err != nil {
		return fmt.Errorf("failed to fetch origin: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(originDir); cleanupErr != nil {
			klog.Warningf("Failed to cleanup origin directory %s: %v", originDir, cleanupErr)
		}
	}()

	// Perform the actual update
	if err := updatePackageResources(ctx, packageDir, updatedDir, originDir); err != nil {
		return fmt.Errorf("failed to update package resources: %w", err)
	}

	// Update the upstream lock
	if err := kptfileutil.UpdateUpstreamLockFromGit(packageDir, &updatedRepoSpec); err != nil {
		return fmt.Errorf("failed to update upstream lock: %w", err)
	}

	return nil
}

// fetchUpstreamGit fetches the upstream Git repository.
// It clones the repository and returns the repository specification and local path.
func fetchUpstreamGit(ctx context.Context, upstream *kptfilev1.Git) (git.RepoSpec, string, error) {
	if upstream == nil {
		return git.RepoSpec{}, "", fmt.Errorf("upstream Git configuration cannot be nil")
	}

	if upstream.Repo == "" {
		return git.RepoSpec{}, "", fmt.Errorf("upstream repository cannot be empty")
	}

	upstreamSpec := &git.RepoSpec{
		OrgRepo: upstream.Repo,
		Path:    upstream.Directory,
		Ref:     upstream.Ref,
	}

	klog.Infof("Fetching upstream from %s@%s", upstreamSpec.OrgRepo, upstreamSpec.Ref)

	updated := *upstreamSpec
	if err := fetch.NewCloner(&updated).ClonerUsingGitExec(ctx); err != nil {
		return git.RepoSpec{}, "", fmt.Errorf("failed to fetch upstream: %w", err)
	}

	return updated, updated.AbsPath(), nil
}

// fetchOriginGit fetches the origin Git repository if available.
// If no upstream lock exists, it creates an empty temporary directory.
func fetchOriginGit(ctx context.Context, upstreamLock *kptfilev1.Locator) (string, error) {
	if upstreamLock == nil || upstreamLock.Git == nil {
		// Create empty directory for origin when no lock exists
		dir, err := os.MkdirTemp("", EmptyTempDirPrefix)
		if err != nil {
			return "", fmt.Errorf("failed to create temporary directory: %w", err)
		}
		klog.Infof("No upstream lock found, using empty origin directory: %s", dir)
		return dir, nil
	}

	if upstreamLock.Git.Repo == "" {
		return "", fmt.Errorf("upstream lock repository cannot be empty")
	}

	originSpec := &git.RepoSpec{
		OrgRepo: upstreamLock.Git.Repo,
		Path:    upstreamLock.Git.Directory,
		Ref:     upstreamLock.Git.Commit,
	}

	klog.Infof("Fetching origin from %s@%s", originSpec.OrgRepo, originSpec.Ref)

	if err := fetch.NewCloner(originSpec).ClonerUsingGitExec(ctx); err != nil {
		return "", fmt.Errorf("failed to fetch origin: %w", err)
	}

	return originSpec.AbsPath(), nil
}

// updatePackageResources updates the package resources using the merge updater.
// It performs the actual three-way merge between local, updated, and origin resources.
func updatePackageResources(ctx context.Context, packageDir, updatedDir, originDir string) error {
	if packageDir == "" || updatedDir == "" || originDir == "" {
		return fmt.Errorf("package directory paths cannot be empty")
	}

	updateOptions := updatetypes.Options{
		RelPackagePath: RootPackagePath,
		LocalPath:      filepath.Join(packageDir, RootPackagePath),
		UpdatedPath:    filepath.Join(updatedDir, RootPackagePath),
		OriginPath:     filepath.Join(originDir, RootPackagePath),
		IsRoot:         true,
	}

	updater := update.ResourceMergeUpdater{}
	if err := updater.Update(updateOptions); err != nil {
		return fmt.Errorf("failed to update package resources: %w", err)
	}

	return nil
}
