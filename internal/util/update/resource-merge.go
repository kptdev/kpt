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

	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
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

	// get the original repo
	original := &git.RepoSpec{OrgRepo: g.Repo, Path: g.Directory, Ref: g.Commit}
	if err := get.ClonerUsingGitExec(original); err != nil {
		return errors.Errorf("failed to clone git repo: original source: %v", err)
	}
	defer os.RemoveAll(original.AbsPath())

	// get the updated repo
	updated := &git.RepoSpec{OrgRepo: options.ToRepo, Path: g.Directory, Ref: options.ToRef}
	if err := get.ClonerUsingGitExec(updated); err != nil {
		return errors.Errorf("failed to clone git repo: updated source: %v", err)
	}
	defer os.RemoveAll(updated.AbsPath())

	err := settersutil.SetAllSetterDefinitions(
		filepath.Join(options.PackagePath, "Kptfile"),
		original.AbsPath(),
		updated.AbsPath(),
	)
	if err != nil {
		return err
	}

	// get the Kptfile to write after the merge
	kf, err := u.updatedKptfile(updated.AbsPath(), options)
	if err != nil {
		return err
	}

	// merge the Resources: original + updated + dest => dest
	err = filters.Merge3{
		OriginalPath: original.AbsPath(),
		UpdatedPath:  updated.AbsPath(),
		DestPath:     options.PackagePath,
	}.Merge()
	if err != nil {
		return err
	}

	if err := kptfileutil.WriteFile(options.PackagePath, kf); err != nil {
		return err
	}

	return ReplaceNonKRMFiles(updated.AbsPath(), original.AbsPath(), options.PackagePath)
}

// updatedKptfile returns a Kptfile to replace the existing local Kptfile as part of the update
func (u ResourceMergeUpdater) updatedKptfile(updatedPath string, options UpdateOptions) (
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
	kf, err := kptfileutil.ReadFile(updatedPath)
	if err != nil {
		kf, err = kptfileutil.ReadFile(options.PackagePath)
		if err != nil {
			return kptfile.KptFile{}, err
		}
	}

	// local package controls the upstream field
	kf.Upstream = options.KptFile.Upstream
	kf.Upstream.Git.Commit = commit
	kf.Upstream.Git.Ref = options.ToRef
	kf.Upstream.Git.Repo = options.ToRepo

	// keep the local OpenAPI values
	err = kf.MergeOpenAPI(options.KptFile)
	return kf, err
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
	r := kio.LocalPackageReader{}
	r.MatchFilesGlob = kio.DefaultMatch
	r.MatchFilesGlob = append(r.MatchFilesGlob, "Kptfile")
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
		match, err := r.ShouldSkipFile(info)
		if err != nil {
			return err
		}
		if !match {
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
