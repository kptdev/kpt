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

	"github.com/GoogleContainerTools/kpt/internal/pipeline"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
)

// Absolute unique OS-defined path to the package directory on the filesystem.
type UniquePath string

// Slash-separated path to the package directory on the filesytem relative to current working directory.
// This is not guaranteed to be unique (e.g. in presence of symlinks) and should only
// be used for display purposes and is subject to change.
type DisplayPath string

// Pkg represents a kpt package with a one-to-one mapping to a directory on the local filesystem.
type Pkg struct {
	UniquePath  UniquePath
	DisplayPath DisplayPath

	// A package can contain zero or one Kptfile meta resource.
	// A nil value represents an implicit package.
	kptfile       *kptfile.KptFile
	kptfileLoaded bool

	// A package can contain zero or one Pipeline meta resource.
	pipeline       *pipeline.Pipeline
	pipelineLoaded bool
}

// New returns a pkg given an absolute or relative OS-defined path.
// Use ReadKptfile or ReadPipeline on the return value to read meta resources from filesystem.
func New(path string) (*Pkg, error) {
	p := filepath.Clean(path)

	u, err := filepath.EvalSymlinks(p)
	if err != nil {
		return nil, err
	}

	u, err = filepath.Abs(u)
	if err != nil {
		return nil, err
	}

	var d string
	if filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		d, err = filepath.Rel(wd, p)
		if err != nil {
			return nil, err
		}
	}
	d = filepath.ToSlash(d)
	return &Pkg{UniquePath: UniquePath(u), DisplayPath: DisplayPath(d)}, nil
}

// Kptfile returns the Kptfile meta resource by lazy loading it from the filesytem.
// A nil value represents an implicit package.
func (p *Pkg) Kptfile() *kptfile.KptFile {
	if !p.kptfileLoaded {
		// TODO
		// p.kptfile = ...
		p.kptfileLoaded = true
	}
	return p.kptfile
}

// Pipeline returns the Pipeline meta resource by lazy loading it from the filesystem.
func (p *Pkg) Pipeline() *pipeline.Pipeline {
	if !p.pipelineLoaded {
		// TODO
		// p.pipeline = ...
		p.pipelineLoaded = true
	}
	return p.pipeline
}

// String returns the slash-seperated relative path to the package.
func (p *Pkg) String() string {
	return string(p.DisplayPath)
}
