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

// AllMatcher is a FileMatcher implementation that matches all files and
// directories.
var AllMatcher = FileMatcherFunc(func(string, os.FileInfo) bool {
	return true
})

// YamlMatcher is a FileMatcher implementation that matches all files
// with .yaml or .yml extensions and all directories.
var YamlMatcher = FileMatcherFunc(func(_ string, info os.FileInfo) bool {
	if info.IsDir() {
		// We want to match all directories, so just return true here.
		return true
	}

	ext := filepath.Ext(info.Name())
	if ext == ".yaml" || ext == ".yml" {
		return true
	}
	return false
})

// KptfileYamlMatcher is a FileMatcher implementation that matches the
// Kptfile, files with either .yaml or .yml extensions, and all directories.
var KptfileYamlMatcher = FileMatcherFunc(func(s string, info os.FileInfo) bool {
	if info.IsDir() {
		// We want to match all directories, so just return true here.
		return true
	}

	if info.Name() == kptfilev1alpha2.KptFileName {
		return true
	}

	ext := filepath.Ext(info.Name())
	return ext == ".yaml" || ext == ".yml"
})

// WalkFunc defines the type for the callback function used by the
// Walker.Walk function.
type WalkFunc func(string, os.FileInfo, error) error

// FileMatcher defines the interface used by the Walker to determine which
// files and directories that should be visited by the walker.
type FileMatcher interface {
	Match(string, os.FileInfo) bool
}

// FileMatcherFunc provides a function implementation of the FileMatcher
// interface.
type FileMatcherFunc func(string, os.FileInfo) bool

func (f FileMatcherFunc) Match(path string, info os.FileInfo) bool {
	return f(path, info)
}

// Walker provides functionality for walking the content of a
// kpt package. It allows for customization of the files and directories
// that will be provided to the caller through the callback function.
type Walker struct {
	// FileMatcher is used to determine which files or directories that should
	// trigger a callback for the walker. By default all files and directories
	// will be provided.
	FileMatcher FileMatcher

	// IgnoreKptfileIgnorePatterns defines whether the walker should consider
	// the Ignore patterns in the Kptfile.
	IgnoreKptfileIgnorePatterns bool
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

	// Read the ignore patterns from the Kptfile (if found) and combine
	// them into a single string. This is the format required by the
	// gitignore library.
	var ignorePatterns string
	if err == nil && len(kf.Ignore) > 0 {
		ignorePatterns = strings.Join(kf.Ignore, "\n")
	}

	pkgPath := string(pkg.UniquePath)
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
		if p.FileMatcher != nil && !p.FileMatcher.Match(path, info) {
			return nil
		}

		// If IgnoreKptfileIgnorePatterns if false and the file/directory is covered by
		// any of the ignore patterns, we don't invoke the callback.
		if !p.IgnoreKptfileIgnorePatterns && ignoreMatcher.Match(path, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return cb(path, info, nil)
	})
}
