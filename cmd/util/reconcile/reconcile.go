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

package reconcile

import (
	"io"

	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/yaml"
)

// Cmd reconciles the set of filters expressed as APIs in the package
type Cmd struct {
	// PkgPath is the path to the package to reconcile
	PkgPath string

	ApisPkgs []string

	Output io.Writer

	// filterProvider may be override by tests to mock invoking containers
	filterProvider func(string, *yaml.RNode) kio.Filter
}

// Execute runs the command
func (r Cmd) Execute() error {
	// default the filterProvider if it hasn't been override.  Split out for testing.
	(&r).init()

	// identify local Resources which are reconcilable APIs and should be invoked locally
	buff := &kio.PackageBuffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{kio.LocalPackageReader{PackagePath: r.PkgPath}},
		Filters: []kio.Filter{&filters.IsReconcilerFilter{}},
		Outputs: []kio.Writer{buff},
	}.Execute()
	if err != nil {
		return err
	}

	// read manual apis and write to the buffer
	for i := range r.ApisPkgs {
		err := kio.Pipeline{
			Inputs:  []kio.Reader{kio.LocalPackageReader{PackagePath: r.ApisPkgs[i]}},
			Outputs: []kio.Writer{buff},
		}.Execute()
		if err != nil {
			return err
		}
	}

	// reconcile each local API
	var fltrs []kio.Filter
	for i := range buff.Nodes {
		api := buff.Nodes[i]
		img := filters.GetContainerName(api)
		fltrs = append(fltrs, r.filterProvider(img, api))
	}

	pkgIO := &kio.LocalPackageReadWriter{PackagePath: r.PkgPath}
	inputs := []kio.Reader{pkgIO}
	var outputs []kio.Writer
	if r.Output == nil {
		// write back to the package
		outputs = append(outputs, pkgIO)
	} else {
		// write to the output instead of the package
		outputs = append(outputs, kio.ByteWriter{Writer: r.Output})
	}
	return kio.Pipeline{Inputs: inputs, Filters: fltrs, Outputs: outputs}.Execute()
}

// init initializes the Cmd with a filterProvider
func (r *Cmd) init() {
	if r.filterProvider == nil {
		r.filterProvider = func(image string, api *yaml.RNode) kio.Filter {
			return &filters.ContainerFilter{Image: image, Config: api}
		}
	}
}
