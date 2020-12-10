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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/merge"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/setters2/settersutil"
)

// ResourceMergeUpdater updates a package by fetching the original and updated source
// packages, and performing a 3-way merge of the Resources.
type ResourceMergeUpdater struct{}

func (u ResourceMergeUpdater) Update(options UpdateOptions) error {
	g := options.KptFile.Upstream.Git
	g.Ref = options.ToRef
	g.Repo = options.ToRepo

	defaultRef, err := gitutil.DefaultRef(g.Repo)
	if err != nil {
		return err
	}

	// get the original repo
	original := &git.RepoSpec{OrgRepo: g.Repo, Path: g.Directory, Ref: g.Commit}
	if err := get.ClonerUsingGitExec(original, defaultRef); err != nil {
		return errors.Errorf("failed to clone git repo: original source: %v", err)
	}
	defer os.RemoveAll(original.AbsPath())

	// get the updated repo
	updated := &git.RepoSpec{OrgRepo: options.ToRepo, Path: g.Directory, Ref: options.ToRef}
	if err := get.ClonerUsingGitExec(updated, defaultRef); err != nil {
		return errors.Errorf("failed to clone git repo: updated source: %v", err)
	}
	defer os.RemoveAll(updated.AbsPath())

	// local package controls the upstream field
	commit, err := u.lookupCommit(updated.AbsPath())
	if err != nil {
		return err
	}

	// Find all subpackages in local, upstream and original. They are sorted
	// in increasing order based on the depth of the subpackage relative to the
	// root package.
	subPkgPaths, err := findAllSubpackages(options.AbsPackagePath, updated.AbsPath(), original.AbsPath())
	if err != nil {
		return err
	}

	// Update each package and subpackage. Parent package is updated before
	// subpackages to make sure auto-setters can work correctly.
	for _, subPkgPath := range subPkgPaths {
		localSubPkgPath := filepath.Join(options.AbsPackagePath, subPkgPath)
		updatedSubPkgPath := filepath.Join(updated.AbsPath(), subPkgPath)
		originalSubPkgPath := filepath.Join(original.AbsPath(), subPkgPath)

		err := u.updatePackage(subPkgPath, localSubPkgPath, updatedSubPkgPath, originalSubPkgPath)
		if err != nil {
			return err
		}
	}

	// Update the kptfile in the local copy to reference the correct
	// upstream after the update.
	options.KptFile.Upstream.Git.Commit = commit
	options.KptFile.Upstream.Git.Ref = options.ToRef
	options.KptFile.Upstream.Git.Repo = options.ToRepo
	err = kptfileutil.WriteFile(options.AbsPackagePath, options.KptFile)
	if err != nil {
		return err
	}

	return nil
}

// findAllSubpackages traverses the packages in local, updated and original
// and finds all subpackages. A subpackage is a subdirectory underneath the
// root that has a Kptfile in it.
// The list is sorted in increasing order based on the depth of the subpackage
// relative to the root package.
func findAllSubpackages(local, updated, original string) ([]string, error) {
	uniquePaths := make(map[string]bool)
	for _, path := range []string{local, updated, original} {
		paths, err := pathutil.DirsWithFile(path, kptfile.KptFileName, true)
		if err != nil {
			return []string{}, err
		}
		for _, p := range paths {
			relPath, err := filepath.Rel(path, p)
			if err != nil {
				return []string{}, err
			}
			uniquePaths[relPath] = true
		}
	}
	var paths []string
	for p := range uniquePaths {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool {
		iPath := paths[i]
		jPath := paths[j]
		if iPath == "." {
			return true
		}
		if jPath == "." {
			return false
		}
		iSegmentCount := len(strings.Split(iPath, "/"))
		jSegmentCount := len(strings.Split(jPath, "/"))
		return iSegmentCount < jSegmentCount
	})
	return paths, nil
}

// updatePackage updates the package in the location specified by localPath
// using the provided paths to the updated version of the package and the
// original version of the package.
func (u ResourceMergeUpdater) updatePackage(subPkgPath, localPath, updatedPath, originalPath string) error {
	localExists, err := exists(localPath)
	if err != nil {
		return err
	}

	updatedExists, err := exists(updatedPath)
	if err != nil {
		return err
	}

	originalExists, err := exists(originalPath)
	if err != nil {
		return err
	}

	switch {
	// Check if subpackage has been added both in upstream and in local
	case !originalExists && localExists && updatedExists:
		return fmt.Errorf("subpackage %q added in both upstream and local", subPkgPath)
	// Package added in upstream
	case !originalExists && !localExists && updatedExists:
		if err := u.copyPackage(updatedPath, localPath); err != nil {
			return err
		}
	// Package added locally
	case !originalExists && localExists && !updatedExists:
		break // No action needed.
	// Package deleted from both upstream and local
	case originalExists && !localExists && !updatedExists:
		break // No action needed.
	// Package deleted from local
	case originalExists && !localExists && updatedExists:
		break // In this case we assume the user knows what they are doing, so
		// we don't re-add the updated package from upstream.
	// Package deleted from upstream
	case originalExists && localExists && !updatedExists:
		// Check the diff. If there are local changes, we keep the subpackage.
		// TODO(mortent): How should we handle (auto)setters and formatting here?
		// It doesn't seem like there is an obvious way to tell auto-setter
		// changes from manual user changes.
		diff, err := copyutil.Diff(originalPath, localPath)
		if err != nil {
			return err
		}
		if diff.Len() == 0 {
			if err := os.RemoveAll(localPath); err != nil {
				return err
			}
		}
	default:
		if err := u.mergePackage(localPath, updatedPath, originalPath); err != nil {
			return err
		}
	}
	return nil
}

// mergePackage merge a package. It does a 3-way merge by using the provided
// paths to the local, updated and original versions of the package.
func (u ResourceMergeUpdater) mergePackage(localPath, updatedPath, originalPath string) error {
	kf, err := u.updatedKptfile(localPath, updatedPath, originalPath)
	if err != nil {
		return err
	}

	if err := kptfileutil.WriteFile(localPath, kf); err != nil {
		return err
	}

	err = settersutil.SetAllSetterDefinitions(
		false,
		filepath.Join(localPath, kptfile.KptFileName),
		originalPath,
		updatedPath,
		localPath,
	)
	if err != nil {
		return err
	}

	// merge the Resources: original + updated + dest => dest
	err = merge.Merge3{
		OriginalPath: originalPath,
		UpdatedPath:  updatedPath,
		DestPath:     localPath,
		// TODO: Write a test to ensure this is set
		MergeOnPath:        true,
		IncludeSubPackages: false,
	}.Merge()
	if err != nil {
		return err
	}

	return ReplaceNonKRMFiles(updatedPath, originalPath, localPath)
}

// copyPackage copies the content of a single package from src to dst. It
// will not copy resources belonging to any subpackages.
func (u ResourceMergeUpdater) copyPackage(src, dst string) error {
	return walkPackage(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// path is an absolute path, rather than a path relative to src.
		// e.g. if src is /path/to/package, then path might be /path/to/package/and/sub/dir
		// we need the path relative to src `and/sub/dir` when we are copying the files to dest.
		copyTo := strings.TrimPrefix(path, src)

		// make directories that don't exist
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(dst, copyTo), info.Mode())
		}

		// copy file by reading and writing it
		b, err := ioutil.ReadFile(filepath.Join(src, copyTo))
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dst, copyTo), b, info.Mode())
		if err != nil {
			return err
		}

		return nil
	})
}

// walkPackage walks the package defined at src and provides a callback for
// every folder and file. Any subpackages and the .git folder are excluded.
func walkPackage(src string, c func(string, os.FileInfo, error) error) error {
	excludedDirs := make(map[string]bool)
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return c(path, info, err)
		}
		// don't copy the .git dir
		if path != src {
			rel := strings.TrimPrefix(path, src)
			if copyutil.IsDotGitFolder(rel) {
				return nil
			}
		}

		for dir := range excludedDirs {
			if strings.HasPrefix(path, dir) {
				return nil
			}
		}

		if info.IsDir() {
			_, err := os.Stat(filepath.Join(path, kptfile.KptFileName))
			if err != nil && !os.IsNotExist(err) {
				return c(path, info, err)
			}
			if err == nil && path != src {
				excludedDirs[path] = true
				return nil
			}
		}
		return c(path, info, err)
	})
}

// exists returns true if a file or directory exists on the provided path,
// and false otherwise.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return !os.IsNotExist(err), nil
}

// lookupCommit looks up the sha of the current commit on the repo at the
// provided path.
func (u ResourceMergeUpdater) lookupCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = repoPath
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	commit := strings.TrimSpace(string(b))
	return commit, nil
}

// updatedKptfile returns a Kptfile to replace the existing local Kptfile as part of the update
func (u ResourceMergeUpdater) updatedKptfile(localPath, updatedPath, originalPath string) (
	kptfile.KptFile, error) {

	updatedKf, err := kptfileutil.ReadFile(updatedPath)
	if err != nil {
		updatedKf, err = kptfileutil.ReadFile(localPath)
		if err != nil {
			return kptfile.KptFile{}, err
		}
	}

	originalKf, err := kptfileutil.ReadFile(originalPath)
	if err != nil {
		originalKf, err = kptfileutil.ReadFile(localPath)
		if err != nil {
			return kptfile.KptFile{}, err
		}
	}

	localKf, err := kptfileutil.ReadFile(localPath)
	if err != nil {
		return kptfile.KptFile{}, err
	}

	// keep the local OpenAPI values
	err = updatedKf.MergeOpenAPI(localKf, originalKf)
	localKf.OpenAPI = updatedKf.OpenAPI
	localKf.Upstream = updatedKf.Upstream
	return localKf, err
}

// replaceNonKRMFiles replaces the non KRM files in localDir with the corresponding files in updatedDir,
// it also deletes non KRM files and sub dirs which are present in localDir and not in updatedDir
func ReplaceNonKRMFiles(updatedDir, originalDir, localDir string) error {
	updatedSubDirs, updatedFiles, err := getSubDirsAndNonKrmFiles(updatedDir)
	if err != nil {
		return err
	}

	originalSubDirs, originalFiles, err := getSubDirsAndNonKrmFiles(originalDir)
	if err != nil {
		return err
	}

	localSubDirs, localFiles, err := getSubDirsAndNonKrmFiles(localDir)
	if err != nil {
		return err
	}

	// identify all non KRM files modified locally, to leave them untouched
	locallyModifiedFiles := sets.String{}
	for _, file := range localFiles.List() {
		if !originalFiles.Has(file) {
			// new local file has been added
			locallyModifiedFiles.Insert(file)
			continue
		}
		same, err := compareFiles(filepath.Join(originalDir, file), filepath.Join(localDir, file))
		if err != nil {
			return err
		}
		if !same {
			// local file has been modified
			locallyModifiedFiles.Insert(file)
			continue
		}

		// remove the file from local if it is not modified and is deleted from updated upstream
		if !updatedFiles.Has(file) {
			if err = os.Remove(filepath.Join(localDir, file)); err != nil {
				return err
			}
		}
	}

	// make sure local has all sub-dirs present in updated
	for _, dir := range updatedSubDirs.List() {
		if err = os.MkdirAll(filepath.Join(localDir, dir), 0700); err != nil {
			return err
		}
	}

	// replace all non KRM files in local with the ones in updated
	for _, file := range updatedFiles.List() {
		if locallyModifiedFiles.Has(file) {
			// skip syncing locally modified files
			continue
		}
		err = copyutil.SyncFile(filepath.Join(updatedDir, file), filepath.Join(localDir, file))
		if err != nil {
			return err
		}
	}

	// delete all the empty dirs in local which are not in updated
	for _, dir := range localSubDirs.List() {
		if !updatedSubDirs.Has(dir) && originalSubDirs.Has(dir) {
			// removes only empty dirs
			os.Remove(filepath.Join(localDir, dir))
		}
	}

	return nil
}

// getSubDirsAndNonKrmFiles returns the list of all non git sub dirs and, non git+non KRM files
// in the root directory
func getSubDirsAndNonKrmFiles(root string) (sets.String, sets.String, error) {
	files := sets.String{}
	dirs := sets.String{}
	err := walkPackage(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err)
		}

		if info.IsDir() {
			path = strings.TrimPrefix(path, root)
			if len(path) > 0 {
				dirs.Insert(path)
			}
			return nil
		}
		isKrm, err := isKrmFile(path)
		if err != nil {
			return err
		}
		if !isKrm {
			path = strings.TrimPrefix(path, root)
			if len(path) > 0 && !strings.Contains(path, ".git") {
				files.Insert(path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return dirs, files, nil
}

var krmFilesGlob = append([]string{kptfile.KptFileName}, kio.DefaultMatch...)

// isKrmFile checks if the file pointed to by the path is a yaml file (including
// the Kptfile).
func isKrmFile(path string) (bool, error) {
	for _, g := range krmFilesGlob {
		if match, err := filepath.Match(g, filepath.Base(path)); err != nil {
			return false, errors.Wrap(err)
		} else if match {
			return true, nil
		}
	}
	return false, nil
}

// compareFiles returns true if src file content is equal to dst file content
func compareFiles(src, dst string) (bool, error) {
	b1, err := ioutil.ReadFile(src)
	if err != nil {
		return false, err
	}
	b2, err := ioutil.ReadFile(dst)
	if err != nil {
		return false, err
	}
	if bytes.Equal(b1, b2) {
		return true, nil
	}
	return false, nil
}
