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

package pipeline

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"k8s.io/klog"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/pathutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Executor hydrates a given pkg.
type Executor struct {
	PkgPath string
}

// Execute runs a pipeline.
func (e *Executor) Execute() error {
	klog.Infof("Running pipeline for pkg %s", e.PkgPath)

	p, err := newPkg(e.PkgPath)
	if err != nil {
		return fmt.Errorf("failed to initialize pkg %w", err)
	}

	// initialize hydration context
	hctx := &hydrationContext{
		root:            p,
		hydrationStates: map[string]hydrationState{},
		pkgs:            map[string]*pkg{},
	}

	resources, err := hydrate(p, hctx)
	if err != nil {
		return fmt.Errorf("failed to hydrate the pkg %s %w", e.PkgPath, err)
	}
	// TODO(droot): support different sink modes
	// for now do in-place
	klog.Infof("hydrated reources: %d", len(resources))

	return nil
}

//
// hydrationContext contains bits to track state of a package hydration.
// This is sort of global state that is available to hydration step at
// each pkg along the hydration walk.
type hydrationContext struct {
	// root points to the root of hydration graph where we bagan our hydration journey
	root *pkg

	// TODO (droot): wire sink-mode here

	// pkgs refers to the discovered pkgs where
	// It's a map where key refers to package's absolute path and value
	// refers to the pkg.
	pkgs map[string]*pkg

	// hydrationStates for each package (dry, hydrating or wet) in the hydration DAG
	hydrationStates map[string]hydrationState
}

//
// pkg represents a kpt package (node) in the hydration DAG.
//
// TODO(droot): kpt will certainly have pkg abstraction, may be embed that here to
// avoid duplication.
type pkg struct {
	// absolute path on the filesystem
	path string

	// pipeline contained in the pkg
	pipeline *Pipeline

	// resources post hydration
	resources []*yaml.RNode
}

// TODO(droot): this will be replaced when pkg abstraction PR is merged
func newPkg(path string) (*pkg, error) {
	pkgPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to find abs path %s %w", path, err)
	}
	p := &pkg{
		path: pkgPath,
	}
	return p, nil
}

func (p *pkg) Path() string {
	return p.path
}

func (p *pkg) Read() error {
	// Read the pipeline for the current package else assume default
	// TODO(droot): integrate with Donny's code once that is merged
	// and read an actual pipeline
	p.pipeline = New()

	return nil
}

// TODO: This will be replaced with the pkg abstraction PR.
func (p *pkg) Pipeline() *Pipeline {
	if p.pipeline == nil {
		// READ the pipeline in the pkg.
		p.pipeline = New()
	}
	return p.pipeline
}

// resolveSources takes a list of sources (./*, ./, ...) and current pkg path
// and returns list of package paths that this pkg depends on.
// This is one of the critical pieces of code
func (p *pkg) resolveSources() ([]string, error) {
	pipeline := p.Pipeline()

	var pkgPaths []string
	for _, s := range pipeline.Sources {
		paths, err := resolveSource(s, p.Path())
		if err != nil {
			return nil, err
		}
		pkgPaths = append(pkgPaths, paths...)
	}
	return pkgPaths, nil
}

func resolveSource(source string, pkgPath string) ([]string, error) {
	switch source {
	case sourceCurrentPkg:
		// include only this pkg sources
		return []string{pkgPath}, nil
	case sourceAllSubPkgs:
		// including current pkg and subpkgs in current directory
		var paths []string
		paths = append(paths, pkgPath)
		files, err := ioutil.ReadDir(pkgPath)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if f.IsDir() {
				// A directory is a package if it has a Kptfile.
				// This may change as the concept of a package is expanded such that
				// every directory is its own kpt package
				absolutePath := path.Join(pkgPath, f.Name())
				subPaths, err := pathutil.DirsWithFile(absolutePath, kptfile.KptFileName, false)
				if err != nil {
					return nil, err
				}
				sort.Strings(subPaths)
				paths = append(paths, subPaths...)
			}
		}
		return paths, nil
	default:
		// s points to a specific sub pkg
		return []string{source}, nil
	}
}

// localResources reads
func (p *pkg) localResources(includeMetadata bool) (resources []*yaml.RNode, err error) {
	// TODO(droot): figure out how to avoid reading sub packages here ?
	pkgReader := &kio.LocalPackageReader{PackagePath: p.Path(), MatchFilesGlob: kio.MatchAll}
	resources, err = pkgReader.Read()
	if err != nil {
		err = fmt.Errorf("failed to read resources for pkg %s %w", p.Path(), err)
		return resources, err
	}
	if !includeMetadata {
		// TODO(droot): this will be the place where we filter kpt resources
		resources = filterMetaData(resources)
	}
	return resources, err
}

// hydrationState represent hydration state of a pkg.
type hydrationState int

// constants for all the hydration states
const (
	Dry hydrationState = iota
	Hydrating
	Wet
)

func (s hydrationState) String() string {
	return []string{"Dry", "Hydrating", "Wet"}[s]
}

//
// hydrate hydrates a pkg and returns wet resources.
func hydrate(p *pkg, hctx *hydrationContext) (resources []*yaml.RNode, err error) {
	state, found := hctx.hydrationStates[p.Path()]
	if found {
		switch state {
		case Hydrating:
			// we have a cycle
			// TODO(droot): improve the error to contain useful information
			err = fmt.Errorf("found cycle in dependencies for pkg %s", p.Path())
		case Wet:
			// already hydrated, we sort of assume pkgs in hctx will be populated but we
			// can be defensive here
			resources = hctx.pkgs[p.Path()].resources
		default:
			err = fmt.Errorf("invalid pkg state %v detected", state)
		}
		return resources, err
	}
	// mark the pkg in hydrating
	hctx.hydrationStates[p.Path()] = Hydrating
	// add to the graph
	hctx.pkgs[p.Path()] = p

	// read the pkg, the pipeline, dependencies etc.
	err = p.Read()
	if err != nil {
		err = fmt.Errorf("failed to read the pkg %w", err)
		return resources, err
	}

	// input resources for current pkg's hydration
	var input []*yaml.RNode
	// here we explore the p.sources to determine sub packages etc.
	sources, err := p.resolveSources()
	if err != nil {
		return resources, err
	}
	for _, s := range sources {
		// TODO(droot): sync with pipeline's defaults once Donny's PR is merged
		if s == p.Path() {
			var currPkgResources []*yaml.RNode
			// current pkg will be hydrated as we drop of this loop
			currPkgResources, err = p.localResources(false)
			if err != nil {
				return resources, err
			}
			input = append(input, currPkgResources...)
		} else {
			// TODO(droot): recursive hydration is broken for now, needs more work
			var transitiveResources []*yaml.RNode
			var depPkg *pkg

			if depPkg, err = newPkg(s); err != nil {
				return resources, err
			}

			transitiveResources, err = hydrate(depPkg, hctx)
			if err != nil {
				err = fmt.Errorf("failed to hydrated sub pkg %s %w", s, err)
				return resources, err
			}
			input = append(input, transitiveResources...)
		}
	}

	output := &kio.PackageBuffer{}
	// TODO(droot): parameterize the sink-mode (hctx.sinkMode) to determine whether to
	// write it in-place or not. Currently in-place.
	pkgWriter := &kio.LocalPackageWriter{PackagePath: p.Path()}
	// create a kio pipeline from kyaml library to execute the function chains
	kioPipeline := kio.Pipeline{
		Inputs: []kio.Reader{
			&kio.PackageBuffer{Nodes: input},
		},
		Filters: fnFilters(p.Pipeline()),         // we will gather filters from the pipeline
		Outputs: []kio.Writer{pkgWriter, output}, // here may be we don't want to write to the fs yet
	}
	err = kioPipeline.Execute()
	if err != nil {
		err = fmt.Errorf("failed to run pipeline for pkg %s %w", p.path, err)
		return
	}

	// pkg is hydrated
	resources = output.Nodes
	// mark the pkg as wet and update the resources
	hctx.hydrationStates[p.Path()] = Wet
	p.resources = resources
	return resources, err
}

// filterMetaData filters kpt metadata files such as pipeline,
// Kptfile, permissions etc.
func filterMetaData(resources []*yaml.RNode) []*yaml.RNode {
	var filtered []*yaml.RNode
	for _, r := range resources {
		meta, _ := r.GetMeta()
		if !strings.Contains(meta.APIVersion, "kpt.dev") {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// fnFilters returns chain of functions that are applicable
// to a given pipeline.
func fnFilters(_ *Pipeline) []kio.Filter {
	fn := &annotator{
		key:   "builtin/setter-1",
		value: "test",
	}

	// TODO(droot): Implement this with the logic to create
	// function chain from a pipeline
	// hardcoding built-in set-annotation function for testing
	return []kio.Filter{
		&fnRunner{
			fn: fn,
		},
	}
}
