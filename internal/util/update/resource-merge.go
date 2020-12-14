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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
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

	kf, err := u.updatedKptfile(updated.AbsPath(), original.AbsPath(), options)
	if err != nil {
		return err
	}

	if err := kptfileutil.WriteFile(options.PackagePath, kf); err != nil {
		return err
	}

	err = settersutil.SetAllSetterDefinitions(
		filepath.Join(options.PackagePath, kptfile.KptFileName),
		original.AbsPath(),
		updated.AbsPath(),
		options.PackagePath,
	)
	if err != nil {
		return err
	}

	if err := MergeSubPackages(options.PackagePath, updated.AbsPath(), original.AbsPath()); err != nil {
		return err
	}

	// merge the Resources: original + updated + dest => dest
	err = filters.Merge3{
		OriginalPath: original.AbsPath(),
		UpdatedPath:  updated.AbsPath(),
		DestPath:     options.PackagePath,
		// TODO: Write a test to ensure this is set
		MergeOnPath: true,
	}.Merge()
	if err != nil {
		return err
	}

	return ReplaceNonKRMFiles(updated.AbsPath(), original.AbsPath(), options.PackagePath)
}

// updatedKptfile returns a Kptfile to replace the existing local Kptfile as part of the update
func (u ResourceMergeUpdater) updatedKptfile(updatedPath, originalPath string, options UpdateOptions) (
	kptfile.KptFile, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = updatedPath
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return kptfile.KptFile{}, err
	}
	commit := strings.TrimSpace(string(b))
	updatedKf, err := kptfileutil.ReadFile(updatedPath)
	if err != nil {
		updatedKf, err = kptfileutil.ReadFile(options.PackagePath)
		if err != nil {
			return kptfile.KptFile{}, err
		}
	}

	originalKf, err := kptfileutil.ReadFile(originalPath)
	if err != nil {
		originalKf, err = kptfileutil.ReadFile(options.PackagePath)
		if err != nil {
			return kptfile.KptFile{}, err
		}
	}

	// local package controls the upstream field
	updatedKf.Upstream = options.KptFile.Upstream
	updatedKf.Upstream.Git.Commit = commit
	updatedKf.Upstream.Git.Ref = options.ToRef
	updatedKf.Upstream.Git.Repo = options.ToRepo

	// keep the local OpenAPI values
	err = updatedKf.MergeOpenAPI(options.KptFile, originalKf)
	return updatedKf, err
}

// MergeSubPackages merges the Kptfiles in the nested subdirectories of the
// root package and also sets the setter definitions in updated to match with
// locally set values so the the resources are correctly identified and merged
func MergeSubPackages(localRoot, updatedRoot, originalRoot string) error {
	localPkgPaths, err := pathutil.DirsWithFile(localRoot, kptfile.KptFileName, true)
	if err != nil {
		return err
	}
	for _, localPkgPath := range localPkgPaths {
		// skip the top level file as it should be merged differently
		if filepath.Clean(localPkgPath) == filepath.Clean(localRoot) {
			continue
		}

		cleanLocalPkgPath := filepath.Clean(localPkgPath)
		relativePkgPath, err := filepath.Rel(localRoot, cleanLocalPkgPath)
		if err != nil {
			return err
		}

		localKf, err := kptfileutil.ReadFile(localPkgPath)
		if err != nil {
			return err
		}

		var updatedKf kptfile.KptFile
		updatedPkgPath := filepath.Join(updatedRoot, relativePkgPath)
		if !fileExists(filepath.Join(updatedPkgPath, kptfile.KptFileName)) {
			// if there is no Kptfile in upstream then use the local Kptfile
			// to retain it
			updatedKf = localKf
		} else {
			updatedKf, err = kptfileutil.ReadFile(updatedPkgPath)
			if err != nil {
				return err
			}
		}

		var originalKf kptfile.KptFile
		originalPkgPath := filepath.Join(originalRoot, relativePkgPath)
		if !fileExists(filepath.Join(originalPkgPath, kptfile.KptFileName)) {
			// if there is no Kptfile at origin then use the local Kptfile
			// to retain it
			originalKf = localKf
		} else {
			originalKf, err = kptfileutil.ReadFile(originalPkgPath)
			if err != nil {
				return err
			}
		}

		err = updatedKf.MergeOpenAPI(localKf, originalKf)
		if err != nil {
			return err
		}

		if err := kptfileutil.WriteFile(localPkgPath, updatedKf); err != nil {
			return err
		}

		// make sure that the updated and original packages are set with the new values
		// to setter parameters in local so that the resources are identified and merged
		// correctly in subpackages
		dirsForSettersUpdate := []string{localPkgPath}
		if fileExists(filepath.Join(updatedPkgPath, kptfile.KptFileName)) {
			dirsForSettersUpdate = append(dirsForSettersUpdate, updatedPkgPath)
		}
		if fileExists(filepath.Join(originalPkgPath, kptfile.KptFileName)) {
			dirsForSettersUpdate = append(dirsForSettersUpdate, originalPkgPath)
		}

		err = settersutil.SetAllSetterDefinitions(
			filepath.Join(localPkgPath, kptfile.KptFileName),
			dirsForSettersUpdate...,
		)
		if err != nil {
			return err
		}
	}
	return nil
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
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if err != nil {
				return errors.Wrap(err)
			}
			path = strings.TrimPrefix(path, root)
			if len(path) > 0 && !strings.Contains(path, ".git") {
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

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
