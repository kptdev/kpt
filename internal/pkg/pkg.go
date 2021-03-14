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

// Package pkg defines the concept of a kpt package.
package pkg

import (
	"os"
	"path/filepath"

	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
)

// Absolute unique OS-defined path to the package directory on the filesystem.
type UniquePath string

// Slash-separated path to the package directory on the filesytem relative to current working directory.
// This is not guaranteed to be unique (e.g. in presence of symlinks) and should only
// be used for display purposes and is subject to change.
type DisplayPath string

const CurDir = "."
const ParentDir = ".."

// Pkg represents a kpt package with a one-to-one mapping to a directory on the local filesystem.
type Pkg struct {
	UniquePath  UniquePath
	DisplayPath DisplayPath

	// A package can contain zero or one Kptfile meta resource.
	// A nil value represents an implicit package.
	kptfile       *kptfilev1alpha2.KptFile
	kptfileLoaded bool

	// A package can contain zero or one Pipeline meta resource.
	pipeline       *kptfilev1alpha2.Pipeline
	pipelineLoaded bool
}

// New returns a pkg given an absolute or relative OS-defined path.
// Use ReadKptfile or ReadPipeline on the return value to read meta resources from filesystem.
func New(path string) (*Pkg, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var relPath string
	var absPath string
	if filepath.IsAbs(path) {
		// If the provided path is absolute, we find the relative path by
		// comparing it to the current working directory.
		relPath, err = filepath.Rel(cwd, path)
		if err != nil {
			return nil, err
		}
		absPath = filepath.Clean(path)
	} else {
		// If the provided path is relative, we find the absolute path by
		// combining the current working directory with the relative path.
		relPath = filepath.Clean(path)
		absPath = filepath.Join(cwd, path)
	}
	return &Pkg{UniquePath: UniquePath(absPath), DisplayPath: DisplayPath(relPath)}, nil
}

// Kptfile returns the Kptfile meta resource by lazy loading it from the filesytem.
// A nil value represents an implicit package.
func (p *Pkg) Kptfile() *kptfilev1alpha2.KptFile {
	if !p.kptfileLoaded {
		// TODO
		// p.kptfile = ...
		p.kptfileLoaded = true
	}
	return p.kptfile
}

// Pipeline returns the Pipeline meta resource by lazy loading it from the filesystem.
func (p *Pkg) Pipeline() *kptfilev1alpha2.Pipeline {
	if !p.pipelineLoaded {
		// TODO
		// p.pipeline = ...
		p.pipelineLoaded = true
	}
	return p.pipeline
}
