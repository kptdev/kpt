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

package cmdrender

import (
	"context"
	"fmt"
	"github.com/GoogleContainerTools/kpt/internal/util/openapi"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/sets"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// Executor hydrates a given pkg.
type Executor struct {
	PkgPath string
	Printer printer.Printer
}

// Execute runs a pipeline.
func (e *Executor) Execute(ctx context.Context) error {
	const op errors.Op = "fn.render"

	pr := printer.FromContextOrDie(ctx)

	root, err := newPkgNode(e.PkgPath, nil)
	if err != nil {
		return errors.E(op, types.UniquePath(e.PkgPath), err)
	}

	// initialize hydration context
	hctx := &hydrationContext{
		root: root,
		pkgs: map[types.UniquePath]*pkgNode{},
	}

	resources, err := hydrate(ctx, root, hctx)
	if err != nil {
		return errors.E(op, root.pkg.UniquePath, err)
	}

	if err = trackOutputFiles(hctx); err != nil {
		return err
	}

	pkgWriter := &kio.LocalPackageReadWriter{PackagePath: string(root.pkg.UniquePath)}
	err = pkgWriter.Write(resources)
	if err != nil {
		return fmt.Errorf("failed to save resources: %w", err)
	}

	if err = pruneResources(hctx); err != nil {
		return err
	}

	pr.PkgPrintf(root.pkg.UniquePath, "rendered successfully\n")
	return nil
}

// hydrationContext contains bits to track state of a package hydration.
// This is sort of global state that is available to hydration step at
// each pkg along the hydration walk.
type hydrationContext struct {
	// root points to the root pkg of hydration graph
	root *pkgNode

	// pkgs refers to the packages undergoing hydration. pkgs are key'd by their
	// unique paths.
	pkgs map[types.UniquePath]*pkgNode

	// inputFiles is a set of filepaths containing input resources to the
	// functions across all the packages during hydration.
	// The file paths are relative to the root package.
	inputFiles sets.String

	// outputFiles is a set of filepaths containing output resources. This
	// will be compared with the inputFiles to identify files be pruned.
	outputFiles sets.String
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
	resources []*kyaml.RNode
}

// newPkgNode returns a pkgNode instance given a path or pkg.
func newPkgNode(path string, p *pkg.Pkg) (pn *pkgNode, err error) {
	const op errors.Op = "pkg.read"

	if path == "" && p == nil {
		return pn, fmt.Errorf("missing package path %s or package", path)
	}
	if path != "" {
		p, err = pkg.New(path)
		if err != nil {
			return pn, errors.E(op, p.UniquePath, err)
		}
	}
	// Note: Ensuring the presence of Kptfile can probably be moved
	// to the lower level pkg abstraction, but not sure if that
	// is desired in all the cases. So revisit this.
	kf, err := p.Kptfile()
	if err != nil {
		return pn, errors.E(op, p.UniquePath, err)
	}

	if err := kf.Validate(); err != nil {
		return pn, errors.E(op, p.UniquePath, err)
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
func hydrate(ctx context.Context, pn *pkgNode, hctx *hydrationContext) (output []*kyaml.RNode, err error) {
	const op errors.Op = "pkg.render"

	curr, found := hctx.pkgs[pn.pkg.UniquePath]
	if found {
		switch curr.state {
		case Hydrating:
			// we detected a cycle
			err = fmt.Errorf("cycle detected in pkg dependencies")
			return output, errors.E(op, curr.pkg.UniquePath, err)
		case Wet:
			output = curr.resources
			return output, nil
		default:
			return output, errors.E(op, curr.pkg.UniquePath,
				fmt.Errorf("package found in invalid state %v", curr.state))
		}
	}
	// add it to the discovered package list
	hctx.pkgs[pn.pkg.UniquePath] = pn
	curr = pn
	// mark the pkg in hydrating
	curr.state = Hydrating

	relPath, err := curr.pkg.RelativePathTo(hctx.root.pkg)
	if err != nil {
		return nil, errors.E(op, curr.pkg.UniquePath, err)
	}

	var input []*kyaml.RNode

	// determine sub packages to be hydrated
	subpkgs, err := curr.pkg.DirectSubpackages()
	if err != nil {
		return output, errors.E(op, curr.pkg.UniquePath, err)
	}
	// hydrate recursively and gather hydated transitive resources.
	for _, subpkg := range subpkgs {
		var transitiveResources []*kyaml.RNode
		var subPkgNode *pkgNode

		if subPkgNode, err = newPkgNode("", subpkg); err != nil {
			return output, errors.E(op, subpkg.UniquePath, err)
		}

		transitiveResources, err = hydrate(ctx, subPkgNode, hctx)
		if err != nil {
			return output, errors.E(op, subpkg.UniquePath, err)
		}

		input = append(input, transitiveResources...)
	}

	// gather resources present at the current package
	currPkgResources, err := curr.pkg.LocalResources(false)
	if err != nil {
		return output, errors.E(op, curr.pkg.UniquePath, err)
	}

	// ensure input resource's paths are relative to root pkg.
	currPkgResources, err = adjustRelPath(currPkgResources, relPath)
	if err != nil {
		return nil, fmt.Errorf("adjust relative path: %w", err)
	}

	err = trackInputFiles(hctx, currPkgResources)
	if err != nil {
		return nil, err
	}

	// include current package's resources in the input resource list
	input = append(input, currPkgResources...)

	output, err = curr.runPipeline(ctx, input)
	if err != nil {
		return output, errors.E(op, curr.pkg.UniquePath, err)
	}

	// ensure generated resource's file path are relative to root pkg.
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
func (pn *pkgNode) runPipeline(ctx context.Context, input []*kyaml.RNode) ([]*kyaml.RNode, error) {
	const op errors.Op = "pipeline.run"
	if len(input) == 0 {
		return nil, nil
	}

	pl, err := pn.pkg.Pipeline()
	if err != nil {
		return nil, err
	}

	if pl.IsEmpty() {
		return input, nil
	}

	fnChain, err := fnChain(ctx, pl, pn.pkg.UniquePath)
	if err != nil {
		return nil, errors.E(op, pn.pkg.UniquePath, err)
	}

	schemaKRM, err := openapi.GetSchemaKRM()
	if err != nil {
		return nil, err
	}
	input = append(input, schemaKRM)

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
		return nil, errors.E(op, pn.pkg.UniquePath, err)
	}
	return openapi.RemoveSchemaKRM(output.Nodes)
}

// path (location) of a KRM resources is tracked in a special key in
// metadata.annotation field. adjustRelPath updates that path annotation by prepending
// the given relPath to the current path annotation if it doesn't exist already.
// Resources are read from local filesystem or generated at a package level, so the
// path annotation in each resource points to path relative to that package.
// But the resources are written to the file system at the root package level, so
// the path annotation in each resources needs to be adjusted to be relative to the rootPkg.
func adjustRelPath(resources []*kyaml.RNode, relPath string) ([]*kyaml.RNode, error) {
	if relPath == "" {
		return resources, nil
	}
	for _, r := range resources {
		currPath, _, err := kioutil.GetFileAnnotations(r)
		if err != nil {
			return resources, err
		}
		// if currPath is relative to root pkg i.e. already has relPath, skip it
		if !strings.HasPrefix(currPath, relPath+"/") {
			newPath := path.Join(relPath, currPath)
			err = r.PipeE(kyaml.SetAnnotation(kioutil.PathAnnotation, newPath))
			if err != nil {
				return resources, err
			}
		}
	}
	return resources, nil
}

// fnChain returns a slice of function runners from the
// functions and configs defined in pipeline.
func fnChain(ctx context.Context, pl *kptfilev1alpha2.Pipeline, pkgPath types.UniquePath) ([]kio.Filter, error) {
	fns := []kptfilev1alpha2.Function{}
	fns = append(fns, pl.Mutators...)
	// TODO: Validators cannot modify resources.
	fns = append(fns, pl.Validators...)
	var runners []kio.Filter
	for i := range fns {
		fn := fns[i]
		r, err := newFnRunner(ctx, &fn, pkgPath)
		if err != nil {
			return nil, err
		}
		runners = append(runners, r)
	}
	return runners, nil
}

// trackInputFiles records file paths of input resources in the hydration context.
func trackInputFiles(hctx *hydrationContext, input []*kyaml.RNode) error {
	if err := detectPathConflicts(input); err != nil {
		return err
	}
	if hctx.inputFiles == nil {
		hctx.inputFiles = sets.String{}
	}
	for _, r := range input {
		path, _, err := kioutil.GetFileAnnotations(r)
		if err != nil {
			return fmt.Errorf("path annotation missing: %w", err)
		}
		hctx.inputFiles.Insert(path)
	}
	return nil
}

// trackOutputfiles records the file paths of output resources in the hydration
// context. It should be invoked post hydration.
func trackOutputFiles(hctx *hydrationContext) error {
	if err := detectPathConflicts(hctx.root.resources); err != nil {
		return err
	}
	outputSet := sets.String{}

	for _, r := range hctx.root.resources {
		path, _, err := kioutil.GetFileAnnotations(r)
		if err != nil {
			return fmt.Errorf("path annotation missing: %w", err)
		}
		outputSet.Insert(path)
	}
	hctx.outputFiles = outputSet
	return nil
}

// detectPathConflicts returns an error if the same index/path is on multiple resources
func detectPathConflicts(nodes []*kyaml.RNode) error {
	// map has structure path -> index -> bool
	// to keep track of paths and indexes found
	pathIndexes := make(map[string]map[string]bool)
	for _, node := range nodes {
		fp, index, err := kioutil.GetFileAnnotations(node)
		if err != nil {
			return err
		}
		fp = path.Clean(fp)
		if pathIndexes[fp] == nil {
			pathIndexes[fp] = make(map[string]bool)
		}
		if _, ok := pathIndexes[fp][index]; ok {
			return fmt.Errorf("resource at path %q and index %q already exists", fp, index)
		}
		pathIndexes[fp][index] = true
	}
	return nil
}

// pruneResources compares the input and output of the hydration and prunes
// resources that are no longer present in the output of the hydration.
func pruneResources(hctx *hydrationContext) error {
	filesToBeDeleted := hctx.inputFiles.Difference(hctx.outputFiles)
	for f := range filesToBeDeleted {
		if err := os.Remove(filepath.Join(string(hctx.root.pkg.UniquePath), f)); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}
	return nil
}
