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

package pipeline

import (
	"fmt"
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Executor hydrates a given pkg.
type Executor struct {
	PkgPath string
}

// Execute runs a pipeline.
func (e *Executor) Execute() error {
	root, err := newPkgNode(e.PkgPath, nil)
	if err != nil {
		return err
	}

	// initialize hydration context
	hctx := &hydrationContext{
		root: root,
		pkgs: map[pkg.UniquePath]*pkgNode{},
	}

	resources, err := hydrate(root, hctx)
	if err != nil {
		return fmt.Errorf("failed to run pipeline in package %s %w", root.pkg, err)
	}

	pkgWriter := &kio.LocalPackageReadWriter{PackagePath: string(root.pkg.UniquePath)}
	err = pkgWriter.Write(resources)
	if err != nil {
		return fmt.Errorf("failed to save resources: %w", err)
	}
	return nil
}

//
// hydrationContext contains bits to track state of a package hydration.
// This is sort of global state that is available to hydration step at
// each pkg along the hydration walk.
type hydrationContext struct {
	// root points to the root of hydration graph where we bagan our hydration journey
	root *pkgNode

	// pkgs refers to the packages undergoing hydration. pkgs are key'd by their
	// unique paths.
	pkgs map[pkg.UniquePath]*pkgNode
}

//
// pkgNode represents a package being hydrated. Think of it as a node in the hydration DAG.
//
type pkgNode struct {
	pkg *pkg.Pkg

	// state indicates if the pkg is being hydrated or done.
	state hydrationState

	// KRM resources that we have gathered post hydration for this package.
	// These inludes resources at this pkg as well all it's children.
	resources []*yaml.RNode
}

// newPkgNode returns a pkgNode instance given a path or pkg.
func newPkgNode(path string, p *pkg.Pkg) (pn *pkgNode, err error) {
	if path == "" && p == nil {
		return pn, fmt.Errorf("missing package path %s or package", path)
	}
	if path != "" {
		p, err = pkg.New(path)
		if err != nil {
			return pn, fmt.Errorf("failed to read package %w", err)
		}
	}
	// Note: Ensuring the presence of Kptfile can probably be moved
	// to the lower level pkg abstraction, but not sure if that
	// is desired in all the cases. So revisit this.
	if _, err = p.Kptfile(); err != nil {
		return pn, fmt.Errorf("failed to read kptfile for package %s %w", p, err)
	}
	pn = &pkgNode{
		pkg:   p,
		state: Dry, // package starts in dry state
	}
	return pn, nil
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

// hydrate hydrates given pkg and returns wet resources.
func hydrate(pn *pkgNode, hctx *hydrationContext) (output []*yaml.RNode, err error) {
	curr, found := hctx.pkgs[pn.pkg.UniquePath]
	if found {
		switch curr.state {
		case Hydrating:
			// we detected a cycle
			err = fmt.Errorf("found cycle in dependencies for package %s", curr.pkg)
			return output, err
		case Wet:
			output = curr.resources
			return output, err
		default:
			return output, fmt.Errorf("package %s detected in invalid state", curr.pkg)
		}
	}
	// add it to the discovered package list
	hctx.pkgs[pn.pkg.UniquePath] = pn
	curr = pn
	// mark the pkg in hydrating
	curr.state = Hydrating

	var input []*yaml.RNode

	// determine sub packages to be hydrated
	subpkgs, err := curr.pkg.DirectSubpackages()
	if err != nil {
		return output, err
	}
	// hydrate recursively and gather hydated transitive resources.
	for _, subpkg := range subpkgs {
		var transitiveResources []*yaml.RNode
		var subPkgNode *pkgNode

		if subPkgNode, err = newPkgNode("", subpkg); err != nil {
			return output, err
		}

		transitiveResources, err = hydrate(subPkgNode, hctx)
		if err != nil {
			err = fmt.Errorf("failed to run pipeline on subpackage %s %w", subpkg, err)
			return output, err
		}

		input = append(input, transitiveResources...)
	}

	// gather resources present at the current package
	currPkgResources, err := curr.pkg.LocalResources(false)
	if err != nil {
		return output, err
	}
	// include current package's resources in the input resource list
	input = append(input, currPkgResources...)

	output, err = curr.runPipeline(input)
	if err != nil {
		return output, err
	}

	// Resources are read from local filesystem or generated at a package level, so the
	// path annotation in each resource points to path relative to that package.
	// But the resources are written to the file system at the root package level, so
	// the path annotation in each resources needs to be adjusted to be relative to the rootPkg.
	relPath, err := curr.pkg.RelativePathTo(hctx.root.pkg)
	if err != nil {
		return nil, err
	}
	output, err = adjustRelPath(output, relPath)
	if err != nil {
		return nil, fmt.Errorf("adjust relative path: %w", err)
	}

	// pkg is hydrated, mark the pkg as wet and update the resources
	curr.state = Wet
	curr.resources = output

	return output, err
}

// runPipeline runs the pipeline defined at current pkgNode on given input resources.
func (pn *pkgNode) runPipeline(input []*yaml.RNode) ([]*yaml.RNode, error) {
	if len(input) == 0 {
		return nil, nil
	}

	pl, err := pn.pkg.Pipeline()
	if err != nil {
		return nil, fmt.Errorf("pipeline read: %s %w", pn.pkg, err)
	}

	if pl.IsEmpty() {
		return input, nil
	}

	fnChain, err := fnChain(pl, pn.pkg.UniquePath)
	if err != nil {
		return nil, fmt.Errorf("function filters: %w", err)
	}

	output := &kio.PackageBuffer{}
	// create a kio pipeline from kyaml library to execute the function chains
	kioPipeline := kio.Pipeline{
		Inputs: []kio.Reader{
			&kio.PackageBuffer{Nodes: input},
		},
		Filters: fnChain,
		Outputs: []kio.Writer{output},
	}
	err = kioPipeline.Execute()
	if err != nil {
		return nil, fmt.Errorf("pipeline run: %v %w", pn.pkg, err)
	}
	return output.Nodes, nil
}

// path (location) of a KRM resources is tracked in a special key in
// metadata.annotation field. adjustRelPath updates that path annotation by prepending
// the given relPath to the current path annotation if it doesn't exist already.
// During hydration, paths are adjusted with the relative path to the root
// package before returning the resources to the parent in hydrate call.
func adjustRelPath(resources []*yaml.RNode, relPath string) ([]*yaml.RNode, error) {
	if relPath == "" {
		return resources, nil
	}
	for _, r := range resources {
		meta, err := r.GetMeta()
		if err != nil {
			return resources, err
		}
		if !strings.HasPrefix(meta.Annotations[kioutil.PathAnnotation], relPath) {
			newPath := path.Join(relPath, meta.Annotations[kioutil.PathAnnotation])
			err = r.PipeE(yaml.SetAnnotation(kioutil.PathAnnotation, newPath))
			if err != nil {
				return resources, err
			}
		}
	}
	return resources, nil
}

// fnChain returns a slice of function runners from the
// functions and configs defined in pipeline.
func fnChain(pl *kptfilev1alpha2.Pipeline, pkgPath pkg.UniquePath) ([]kio.Filter, error) {
	fns := []kptfilev1alpha2.Function{}
	fns = append(fns, pl.Mutators...)
	// TODO: Validators cannot modify resources.
	fns = append(fns, pl.Validators...)
	var runners []kio.Filter
	for i := range fns {
		fn := fns[i]
		r, err := newFnRunner(&fn, pkgPath)
		if err != nil {
			return nil, err
		}
		runners = append(runners, r)
	}
	return runners, nil
}
