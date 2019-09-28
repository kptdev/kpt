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
	"regexp"

	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/yaml"
)

// Cmd reconciles the set of filters expressed as APIs in the package
type Cmd struct {
	// PkgPath is the path to the package to reconcile
	PkgPath string

	// filterProvider may be override by tests to mock invoking containers
	filterProvider func(string, *yaml.RNode) kio.Filter
}

// match specifies the set of apiVersions to recognize as being container images
var match = regexp.MustCompile(`(docker\.io|.*\.?gcr\.io)/.*(:.*)?`)

// matchReconcilableAPIs filters Resources to only include Resources for APIs that
// may be locally reconciled.
var matchReconcilableAPIs = kio.FilterFunc(func(inputs []*yaml.RNode) ([]*yaml.RNode, error) {
	var out []*yaml.RNode
	for i := range inputs {
		if getContainerName(inputs[i]) != "" {
			out = append(out, inputs[i])
		}
	}
	return out, nil
})

// Execute runs the command
func (r Cmd) Execute() error {
	// default the filterProvider if it hasn't been override.  Split out for testing.
	(&r).init()

	// identify local Resources which are reconcilable APIs and should be invoked locally
	buff := &kio.PackageBuffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{kio.LocalPackageReader{PackagePath: r.PkgPath}},
		Filters: []kio.Filter{matchReconcilableAPIs},
		Outputs: []kio.Writer{buff},
	}.Execute()
	if err != nil {
		return err
	}

	// reconcile each local API
	var filters []kio.Filter
	for i := range buff.Nodes {
		api := buff.Nodes[i]
		img := getContainerName(api)
		filters = append(filters, r.filterProvider(img, api))
	}
	pkgIO := kio.LocalPackageReadWriter{PackagePath: r.PkgPath}
	return kio.Pipeline{
		Inputs:  []kio.Reader{pkgIO},
		Filters: filters,
		Outputs: []kio.Writer{pkgIO},
	}.Execute()
}

// init initializes the Cmd with a filterProvider
func (r *Cmd) init() {
	if r.filterProvider == nil {
		r.filterProvider = func(image string, api *yaml.RNode) kio.Filter {
			return &filters.ContainerFilter{Image: image, Config: api}
		}
	}
}

// getContainerName returns the container image for an API if one exists
func getContainerName(n *yaml.RNode) string {
	meta, _ := n.GetMeta()
	container := meta.Annotations["kpt.dev/container"]
	if container != "" {
		return container
	}

	if match.MatchString(meta.ApiVersion) {
		return meta.ApiVersion
	}

	return ""
}
