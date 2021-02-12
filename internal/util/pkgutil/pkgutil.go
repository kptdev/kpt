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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

type RemoteSubPkgInfo struct {
	PackagePath          string
	DeclaringKptfilePath string
}

// FindLocalSubpackages returns a slice with the paths to all local subpackages
// under the provided path. Any remote subpackages are excluded.
func FindLocalSubpackages(path string) ([]string, error) {
	var localSubPkgs []string
	var remoteSubPkgs []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, dir := range remoteSubPkgs {
			if strings.HasPrefix(p, dir) {
				return nil
			}
		}

		if filepath.Base(p) != kptfilev1alpha2.KptFileName {
			return nil
		}

		pkgPath := filepath.Dir(p)
		localSubPkgs = append(localSubPkgs, pkgPath)

		kf, err := kptfileutil.ReadFile(pkgPath)
		if err != nil {
			return err
		}
		for _, sp := range kf.Subpackages {
			spPath := filepath.Join(pkgPath, sp.LocalDir)
			remoteSubPkgs = append(remoteSubPkgs, spPath)
		}
		return nil
	})
	return localSubPkgs, err
}

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
func CopyPackage(src, dst string) error {
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

// CheckForParentPackage checks the parent folder of the provided path and
// looks for a Kptfile. It will return the path to the closest Kptfile if
// one is found.
func CheckForParentPackage(src string) (string, bool, error) {
	path := src
	for {
		previous := path
		path = filepath.Dir(path)
		if path == previous {
			return "", false, nil
		}
		found, err := kptfileutil.HasKptfile(path)
		if err != nil {
			return "", false, err
		}
		if found {
			return path, true, nil
		}
	}
}

var emptyFunc = func() error { return nil }

type kptFunc = func(string, kptfilev1alpha2.KptFile) (kptfilev1alpha2.KptFile, error)

// UpdateParentKptfile provides the basics for making changes to the Kptfile
// of a parent package. It takes a path to the current pakcage and a mutator
// function that is allows to make changes to the Kptfile of the parent package
// if one is found. This function returns a function that can be used to revert
// the changes to the Kptfile if needed.
func UpdateParentKptfile(path string, f kptFunc) (func() error, error) {
	if !filepath.IsAbs(path) {
		return emptyFunc, fmt.Errorf("path must be absolute")
	}

	parentPath, found, err := CheckForParentPackage(path)
	if err != nil {
		return emptyFunc, err
	}

	if found {
		kf, err := kptfileutil.ReadFile(parentPath)
		if err != nil {
			return emptyFunc, err
		}

		// Read the file again so we have a copy that we can use
		// to restore the original content if fetching the package fails.
		orgKf, err := kptfileutil.ReadFile(parentPath)
		if err != nil {
			return emptyFunc, err
		}

		newKf, err := f(parentPath, kf)
		if err != nil {
			return emptyFunc, err
		}

		err = kptfileutil.WriteFile(parentPath, newKf)
		if err != nil {
			return emptyFunc, err
		}
		return func() error {
			return kptfileutil.WriteFile(parentPath, orgKf)
		}, nil
	}
	return emptyFunc, nil
}
