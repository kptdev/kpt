// Copyright 2021 Google LLC
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

package pkg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/monochromegane/go-gitignore"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

// AllFileMatchFunc is a matcher func that matches all files.
var AllFileMatchFunc FileMatchFunc = func(string, os.FileInfo) bool {
	return true
}

// YamlFileMatchFunc is a matcher func that matches all files with .yaml or
// .yml extensions.
var YamlFileMatchFunc FileMatchFunc = func(_ string, info os.FileInfo) bool {
	if info.IsDir() {
		return true
	}

	ext := filepath.Ext(info.Name())
	if ext == ".yaml" || ext == ".yml" {
		return true
	}
	return false
}

// KptfileYamlFileMatchFunc is a matcher func that matches the Kptfile and files
// with either .yaml or .yml extensions.
var KptfileYamlFileMatchFunc FileMatchFunc = func(s string, info os.FileInfo) bool {
	if info.IsDir() {
		return true
	}

	if info.Name() == kptfilev1alpha2.KptFileName {
		return true
	}

	ext := filepath.Ext(info.Name())
	if ext == ".yaml" || ext == ".yml" {
		return true
	}
	return false
}

// WalkFunc defines the type for the callback function used by the
// Walker.Walk function.
type WalkFunc func(string, os.FileInfo, error) error

// FileMatchFunc is the type of a function that can be provided to the
// Walker to filter the files that will be provided to the caller.
type FileMatchFunc func(string, os.FileInfo) bool

// Walker provides functionality for walking the content of a
// kpt package. It allows for customization of the files and directories
// that will be provided to the caller through the callback function.
type Walker struct {
	// FileMatchFunc defines the function that will be used by the walker to
	// filter which files should be provided to the callback. By default, all
	// files will be provided.
	FileMatchFunc FileMatchFunc

	// HonorKptfileIgnore defines whether the walker should honor the ignore
	// list from the package Kptfile.
	HonorKptfileIgnore bool
}

// Walk traverses the kpt package provided and invokes the provided
// callback function for every file and directory in the package that
// matches the criteria defined in the Walker.
func (p *Walker) Walk(pkg *Pkg, cb WalkFunc) error {
	kf, err := pkg.Kptfile()

	// Any other errors than os.ErrNotExist means we should error out.
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// If the Kptfile could be read and it has at least one ignore pattern,
	// turn it into a string.
	var ignorePatterns string
	if err == nil && len(kf.Ignore) > 0 {
		ignorePatterns = strings.Join(kf.Ignore, "\n")
	}

	pkgPath := pkg.UniquePath.String()
	ignoreMatcher := gitignore.NewGitIgnoreFromReader(pkgPath, bytes.NewBufferString(ignorePatterns))

	return filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return cb(path, info, err)
		}

		// We ignore any .git folders.
		rel := strings.TrimPrefix(path, pkgPath)
		if copyutil.IsDotGitFolder(rel) {
			return nil
		}

		// If there is a directory and it contains a Kptfile, we skip it since
		// we don't want to traverse into subpackages. We make sure we don't
		// consider the root package here.
		if path != pkgPath && info.IsDir() {
			_, err = os.Stat(filepath.Join(path, kptfilev1alpha2.KptFileName))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err == nil {
				return filepath.SkipDir
			}
		}

		// If the file doesn't meet the matcher criteria, we can just return.
		if !p.FileMatchFunc(path, info) {
			return nil
		}

		// If HonorKptfileIgnore is set and the file/directory is covered by
		// any of the ignore patterns, we don't invoke the callback.
		if p.HonorKptfileIgnore && ignoreMatcher.Match(path, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return cb(path, info, nil)
	})
}
