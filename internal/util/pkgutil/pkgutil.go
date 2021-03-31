// Copyright 2020 Google LLC
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

package pkgutil

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

// WalkPackage walks the package defined at src and provides a callback for
// every folder and file. Any subpackages and the .git folder are excluded.
func WalkPackage(src string, c func(string, os.FileInfo, error) error) error {
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
			_, err := os.Stat(filepath.Join(path, kptfilev1alpha2.KptFileName))
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

// CopyPackage copies the content of a single package from src to dst. It
// will not copy resources belonging to any subpackages.
func CopyPackage(src, dst string, copyRootKptfile bool) error {
	return WalkPackage(src, func(path string, info os.FileInfo, err error) error {
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

		if path == filepath.Join(src, kptfilev1alpha2.KptFileName) && !copyRootKptfile {
			return nil
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

func CopyPackageWithSubpackages(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// don't copy the .git dir
		if path != src {
			rel := strings.TrimPrefix(path, src)
			if copyutil.IsDotGitFolder(rel) {
				return nil
			}
		}

		copyTo := strings.TrimPrefix(path, src)
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(dst, copyTo), info.Mode())
		}

		if copyTo == "/Kptfile" {
			_, err := os.Stat(filepath.Join(dst, copyTo))
			if err == nil {
				return nil
			}
			if !os.IsNotExist(err) {
				return err
			}
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

func RemovePackageContent(path string, removeRootKptfile bool) error {
	// Walk the package (while ignoring subpackages) and delete all files.
	// We capture the paths to any subdirectories in the package so we
	// can handle those later. We can't do it while walking the package
	// since we don't want to end up deleting directories that might
	// contain a nested subpackage.
	var dirs []string
	if err := WalkPackage(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if info.IsDir() {
			if p != path {
				dirs = append(dirs, p)
			}
			return nil
		}

		if p == filepath.Join(path, kptfilev1alpha2.KptFileName) && !removeRootKptfile {
			return nil
		}

		return os.Remove(p)
	}); err != nil {
		return err
	}

	// Delete any of the directories in the package that are
	// empty. We start with the most deeply nested directories
	// so we can just check every directory for files/directories.
	sort.Slice(dirs, SubPkgFirstSorter(dirs))
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
	return nil
}

// RootPkgFirstSorter returns a "less" function that can be used with the
// sort.Slice function to correctly sort package paths so parent packages
// are always before subpackages.
func RootPkgFirstSorter(paths []string) func(i, j int) bool {
	return func(i, j int) bool {
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
	}
}

// SubPkgFirstSorter returns a "less" function that can be used with the
// sort.Slice function to correctly sort package paths so subpackages are
// always before parent packages.
func SubPkgFirstSorter(paths []string) func(i, j int) bool {
	sorter := RootPkgFirstSorter(paths)
	return func(i, j int) bool {
		return !sorter(i, j)
	}
}

// FindSubpackagesForPaths traverses the provided package paths
// and finds all subpackages using the provided pkgLocatorFunc
func FindSubpackagesForPaths(matcher pkg.SubpackageMatcher, recurse bool, pkgPaths ...string) ([]string, error) {
	uniquePaths := make(map[string]bool)
	for _, path := range pkgPaths {
		paths, err := pkg.Subpackages(path, matcher, recurse)
		if err != nil {
			return []string{}, err
		}
		for _, p := range paths {
			uniquePaths[p] = true
		}
	}
	paths := []string{}
	for p := range uniquePaths {
		paths = append(paths, p)
	}
	sort.Slice(paths, RootPkgFirstSorter(paths))
	return paths, nil
}

// Exists returns true if a file or directory exists on the provided path,
// and false otherwise.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return !os.IsNotExist(err), nil
}
