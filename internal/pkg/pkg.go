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

// pkg represents the static metadata for a kpt package.
type pkg struct {
	// Absolute unique OS-defined path to the package directory.
	UniquePath string

	// Relative slash-separated path to the package directory.
	// This is not guaranteed to be unique (e.g. in presence of symlinks) and should only
	// be used for display purposes.
	DisplayPath string

	// A package can contain zero or one Kptfile meta resource.
	// A nil value represents an implicit package.
	Kptfile *kptfile.KptFile

	// A package can contain zero or one Pipeline meta resource.
	Pipeline *pipeline.Pipeline
}

// New returns a pkg given an absolute or relative OS-defined path.
// Use ReadKptfile or ReadPipeline on the return value to read meta resources from filesystem.
func New(path string) (pkg, error) {
	p := filepath.Clean(path)

	u, err := filepath.EvalSymlinks(p)
	if err != nil {
		return pkg{}, err
	}

	u, err = filepath.Abs(u)
	if err != nil {
		return pkg{}, err
	}

	var d string
	if filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return pkg{}, err
		}
		d, err = filepath.Rel(wd, p)
		if err != nil {
			return pkg{}, err
		}
	}
	d = filepath.ToSlash(d)
	return pkg{UniquePath: u, DisplayPath: d}, nil
}

// ReadKptfile reads the Kptfile meta resource from filesystem.
func (p *pkg) ReadKptfile() *pkg {
	// TODO
	// p.Kptfile = ...
	return p
}

// ReadPipeline reads the Pipeline meta resource from filesystem.
func (p *pkg) ReadPipeline() *pkg {
	// TODO
	// p.Pipeline = ...
	return p
}

// String returns the slash-seperated relative path to the package.
func (p *pkg) String() string {
	return p.DisplayPath
}
