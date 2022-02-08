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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/content/paths"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

// WalkPackage walks the package defined at src and provides a callback for
// every folder and file. Any subpackages and the .git folder are excluded.
func WalkPackage(src paths.FileSystemPath, c func(string, os.FileInfo, error) error) error {
	excludedDirs := make(map[string]bool)
	return src.FileSystem.Walk(src.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return c(path, info, err)
		}
		// don't copy the .git dir
		if path != src.Path {
			rel := strings.TrimPrefix(path, src.Path)
			if copyutil.IsDotGitFolder(rel) {
				return nil
			}
		}

		for dir := range excludedDirs {
			if strings.HasPrefix(path, dir) {
				return nil
			}
		}

		if info.IsDir() && path != src.Path {
			hasKptfile := src.FileSystem.Exists(filepath.Join(path, kptfilev1.KptFileName))
			if hasKptfile {
				excludedDirs[path] = true
				return nil
			}
		}
		return c(path, info, err)
	})
}

// CopyPackage copies the content of a single package from src to dst. If includeSubpackages
// is true, it will copy resources belonging to any subpackages.
func CopyPackage(src, dst paths.FileSystemPath, copyRootKptfile bool, matcher pkg.SubpackageMatcher) error {
	subpackagesToCopy, err := pkg.Subpackages(src.FileSystem, src.Path, matcher, true)
	if err != nil {
		return err
	}

	err = WalkPackage(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// path is an absolute path, rather than a path relative to src.
		// e.g. if src is /path/to/package, then path might be /path/to/package/and/sub/dir
		// we need the path relative to src `and/sub/dir` when we are copying the files to dest.
		copyTo := strings.TrimPrefix(path, src.Path)

		if copyTo == "/"+kptfilev1.KptFileName {
			if dst.FileSystem.Exists(filepath.Join(dst.Path, copyTo)) {
				return nil
			}
		}

		// make directories that don't exist
		if info.IsDir() {
			return dst.FileSystem.MkdirAll(filepath.Join(dst.Path, copyTo))
		}

		if path == filepath.Join(src.Path, kptfilev1.KptFileName) && !copyRootKptfile {
			return nil
		}

		// copy file by reading and writing it
		b, err := src.FileSystem.ReadFile(filepath.Join(src.Path, copyTo))
		if err != nil {
			return err
		}
		err = dst.FileSystem.WriteFile(filepath.Join(dst.Path, copyTo), b)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	for _, subpackage := range subpackagesToCopy {
		subpackageSrc := filepath.Join(src.Path, subpackage)
		// subpackageDest := filepath.Join(dst, strings.TrimPrefix(subpackage, src))
		err = src.FileSystem.Walk(subpackageSrc, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// don't copy the .git dir
			if path != src.Path {
				rel := strings.TrimPrefix(path, subpackageSrc)
				if copyutil.IsDotGitFolder(rel) {
					return nil
				}
			}

			copyTo := strings.TrimPrefix(path, src.Path)
			if info.IsDir() {
				return dst.FileSystem.MkdirAll(filepath.Join(dst.Path, copyTo))
			}

			if copyTo == "/"+kptfilev1.KptFileName {
				if dst.FileSystem.Exists(filepath.Join(dst.Path, copyTo)) {
					return nil
				}
			}

			// copy file by reading and writing it
			b, err := src.FileSystem.ReadFile(filepath.Join(src.Path, copyTo))
			if err != nil {
				return err
			}
			err = dst.FileSystem.WriteFile(filepath.Join(dst.Path, copyTo), b)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func RemovePackageContent(path types.FileSystemPath, removeRootKptfile bool) error {
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
			if p != path.Path {
				dirs = append(dirs, p)
			}
			return nil
		}

		if p == filepath.Join(path.Path, kptfilev1.KptFileName) && !removeRootKptfile {
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
		// First sort based on the number of segments.
		iSegmentCount := len(filepath.SplitList(iPath))
		jSegmentCount := len(filepath.SplitList(jPath))
		if jSegmentCount != iSegmentCount {
			return iSegmentCount < jSegmentCount
		}
		// If two paths are at the same depth, just sort lexicographically.
		return iPath < jPath
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
func FindSubpackagesForPaths(matcher pkg.SubpackageMatcher, recurse bool, pkgPaths ...types.FileSystemPath) ([]string, error) {
	uniquePaths := make(map[string]bool)
	for _, path := range pkgPaths {
		paths, err := pkg.Subpackages(path.FileSystem, path.Path, matcher, recurse)
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

// FormatPackage formats resources and meta-resources in the package and all its subpackages
func FormatPackage(pkgPath string) {
	inout := &kio.LocalPackageReadWriter{
		PackagePath:       pkgPath,
		MatchFilesGlob:    append(kio.DefaultMatch, kptfilev1.KptFileName),
		PreserveSeqIndent: true,
		WrapBareSeqNode:   true,
	}
	f := &filters.FormatFilter{
		UseSchema: true,
	}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{inout},
		Filters: []kio.Filter{f},
		Outputs: []kio.Writer{inout},
	}.Execute()
	if err != nil {
		// do not throw error if formatting fails
		return
	}
	err = RoundTripKptfilesInPkg(pkgPath)
	if err != nil {
		// do not throw error if formatting fails
		return
	}
}

// RoundTripKptfilesInPkg reads and writes all Kptfiles in the package including
// subpackages. This is used to format Kptfiles in the order of go structures
// TODO: phanimarupaka remove this method after addressing https://github.com/GoogleContainerTools/kpt/issues/2052
func RoundTripKptfilesInPkg(pkgPath string) error {
	paths, err := pkg.Subpackages(filesys.FileSystemOrOnDisk{}, pkgPath, pkg.All, true)
	if err != nil {
		return err
	}

	var pkgsPaths []string
	for _, path := range paths {
		// join pkgPath as the paths are relative to pkgPath
		pkgsPaths = append(pkgsPaths, filepath.Join(pkgPath, path))
	}
	// include root package as well
	pkgsPaths = append(pkgsPaths, pkgPath)

	for _, pkgPath := range pkgsPaths {
		kf, err := pkg.ReadKptfile(types.DiskPath(pkgPath))
		if err != nil {
			// do not throw error if formatting fails
			return err
		}
		err = kptfileutil.WriteFile(pkgPath, kf)
		if err != nil {
			// do not throw error if formatting fails
			return err
		}
	}
	return nil
}

// Exists returns true if a file or directory exists on the provided path,
// and false otherwise.
func Exists(path types.FileSystemPath) (bool, error) {
	return path.FileSystem.Exists(path.Path), nil
}
