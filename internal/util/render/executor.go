// Copyright 2022 The kpt Authors
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

package render

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/attribution"
	"github.com/GoogleContainerTools/kpt/internal/util/printerutil"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var errAllowedExecNotSpecified = fmt.Errorf("must run with `--allow-exec` option to allow running function binaries")

// Renderer hydrates a given pkg by running the functions in the input pipeline
type Renderer struct {
	// PkgPath is the absolute path to the root package
	PkgPath string

	// Runtime knows how to pick a function runner for a given function
	Runtime fn.FunctionRuntime

	// ResultsDirPath is absolute path to the directory to write results
	ResultsDirPath string

	// fnResultsList is the list of results from the pipeline execution
	fnResultsList *fnresult.ResultList

	// Output is the writer to which the output resources are written
	Output io.Writer

	// RunnerOptions contains options controlling function execution.
	RunnerOptions fnruntime.RunnerOptions

	// FileSystem is the input filesystem to operate on
	FileSystem filesys.FileSystem
}

// Execute runs a pipeline.
func (e *Renderer) Execute(ctx context.Context) (*fnresult.ResultList, error) {
	const op errors.Op = "fn.render"

	pr := printer.FromContextOrDie(ctx)

	root, err := newPkgNode(e.FileSystem, e.PkgPath, nil)
	if err != nil {
		return nil, errors.E(op, types.UniquePath(e.PkgPath), err)
	}

	// initialize hydration context
	hctx := &hydrationContext{
		root:          root,
		pkgs:          map[types.UniquePath]*pkgNode{},
		fnResults:     fnresult.NewResultList(),
		runnerOptions: e.RunnerOptions,
		fileSystem:    e.FileSystem,
		runtime:       e.Runtime,
	}

	if _, err = hydrate(ctx, root, hctx); err != nil {
		// Note(droot): ignore the error in function result saving
		// to avoid masking the hydration error.
		// don't disable the CLI output in case of error
		_ = e.saveFnResults(ctx, hctx.fnResults)
		return hctx.fnResults, errors.E(op, root.pkg.UniquePath, err)
	}

	// adjust the relative paths of the resources.
	err = adjustRelPath(hctx)
	if err != nil {
		return nil, err
	}

	if err = trackOutputFiles(hctx); err != nil {
		return nil, err
	}

	// add metrics annotation to output resources to track the usage as the resources
	// are rendered by kpt fn group
	at := attribution.Attributor{Resources: hctx.root.resources, CmdGroup: "fn"}
	at.Process()

	if e.Output == nil {
		// the intent of the user is to modify resources in-place
		pkgWriter := &kio.LocalPackageReadWriter{
			PackagePath:        string(root.pkg.UniquePath),
			PreserveSeqIndent:  true,
			PackageFileName:    kptfilev1.KptFileName,
			IncludeSubpackages: true,
			WrapBareSeqNode:    true,
			FileSystem:         filesys.FileSystemOrOnDisk{FileSystem: e.FileSystem},
			MatchFilesGlob:     pkg.MatchAllKRM,
		}
		err = pkgWriter.Write(hctx.root.resources)
		if err != nil {
			return nil, fmt.Errorf("failed to save resources: %w", err)
		}

		if err = pruneResources(e.FileSystem, hctx); err != nil {
			return nil, err
		}
		pr.Printf("Successfully executed %d function(s) in %d package(s).\n", hctx.executedFunctionCnt, len(hctx.pkgs))
	} else {
		// the intent of the user is to write the resources to either stdout|unwrapped|<OUT_DIR>
		// so, write the resources to provided e.Output which will be written to appropriate destination by cobra layer
		writer := &kio.ByteWriter{
			Writer:                e.Output,
			KeepReaderAnnotations: true,
			WrappingAPIVersion:    kio.ResourceListAPIVersion,
			WrappingKind:          kio.ResourceListKind,
		}
		err = writer.Write(hctx.root.resources)
		if err != nil {
			return nil, fmt.Errorf("failed to write resources: %w", err)
		}
	}

	return hctx.fnResults, e.saveFnResults(ctx, hctx.fnResults)
}

func (e *Renderer) saveFnResults(ctx context.Context, fnResults *fnresult.ResultList) error {
	e.fnResultsList = fnResults
	resultsFile, err := fnruntime.SaveResults(e.FileSystem, e.ResultsDirPath, fnResults)
	if err != nil {
		return fmt.Errorf("failed to save function results: %w", err)
	}

	printerutil.PrintFnResultInfo(ctx, resultsFile, false)
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

	// executedFunctionCnt is the counter for functions that have been executed.
	executedFunctionCnt int

	// fnResults stores function results gathered
	// during pipeline execution.
	fnResults *fnresult.ResultList

	runnerOptions fnruntime.RunnerOptions

	fileSystem filesys.FileSystem

	// function runtime
	runtime fn.FunctionRuntime
}

// pkgNode represents a package being hydrated. Think of it as a node in the hydration DAG.
type pkgNode struct {
	pkg *pkg.Pkg

	// state indicates if the pkg is being hydrated or done.
	state hydrationState

	// KRM resources that we have gathered post hydration for this package.
	// These inludes resources at this pkg as well all it's children.
	resources []*yaml.RNode
}

// newPkgNode returns a pkgNode instance given a path or pkg.
func newPkgNode(fsys filesys.FileSystem, path string, p *pkg.Pkg) (pn *pkgNode, err error) {
	const op errors.Op = "pkg.read"

	if path == "" && p == nil {
		return pn, fmt.Errorf("missing package path %s or package", path)
	}
	if path != "" {
		p, err = pkg.New(fsys, path)
		if err != nil {
			return pn, errors.E(op, path, err)
		}
	}
	// Note: Ensuring the presence of Kptfile can probably be moved
	// to the lower level pkg abstraction, but not sure if that
	// is desired in all the cases. So revisit this.
	kf, err := p.Kptfile()
	if err != nil {
		return pn, errors.E(op, p.UniquePath, err)
	}

	if err := kf.Validate(fsys, p.UniquePath); err != nil {
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
func hydrate(ctx context.Context, pn *pkgNode, hctx *hydrationContext) (output []*yaml.RNode, err error) {
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

	var input []*yaml.RNode

	// determine sub packages to be hydrated
	subpkgs, err := curr.pkg.DirectSubpackages()
	if err != nil {
		return output, errors.E(op, curr.pkg.UniquePath, err)
	}
	// hydrate recursively and gather hydated transitive resources.
	for _, subpkg := range subpkgs {
		var transitiveResources []*yaml.RNode
		var subPkgNode *pkgNode

		if subPkgNode, err = newPkgNode(hctx.fileSystem, "", subpkg); err != nil {
			return output, errors.E(op, subpkg.UniquePath, err)
		}

		transitiveResources, err = hydrate(ctx, subPkgNode, hctx)
		if err != nil {
			return output, errors.E(op, subpkg.UniquePath, err)
		}

		input = append(input, transitiveResources...)
	}

	// gather resources present at the current package
	currPkgResources, err := curr.pkg.LocalResources()
	if err != nil {
		return output, errors.E(op, curr.pkg.UniquePath, err)
	}

	err = trackInputFiles(hctx, relPath, currPkgResources)
	if err != nil {
		return nil, err
	}

	// include current package's resources in the input resource list
	input = append(input, currPkgResources...)

	output, err = curr.runPipeline(ctx, hctx, input)
	if err != nil {
		return output, errors.E(op, curr.pkg.UniquePath, err)
	}

	// pkg is hydrated, mark the pkg as wet and update the resources
	curr.state = Wet
	curr.resources = output

	return output, err
}

// runPipeline runs the pipeline defined at current pkgNode on given input resources.
func (pn *pkgNode) runPipeline(ctx context.Context, hctx *hydrationContext, input []*yaml.RNode) ([]*yaml.RNode, error) {
	const op errors.Op = "pipeline.run"
	pr := printer.FromContextOrDie(ctx)
	// TODO: the DisplayPath is a relative file path. It cannot represent the
	// package structure. We should have function to get the relative package
	// path here.
	pr.OptPrintf(printer.NewOpt().PkgDisplay(pn.pkg.DisplayPath), "\n")

	pl, err := pn.pkg.Pipeline()
	if err != nil {
		return nil, err
	}

	if pl.IsEmpty() {
		if err := kptfilev1.AreKRM(input); err != nil {
			return nil, fmt.Errorf("input resource list must contain only KRM resources: %s", err.Error())
		}
		return input, nil
	}

	// perform runtime validation for pipeline
	if err := pn.pkg.ValidatePipeline(); err != nil {
		return nil, err
	}

	mutatedResources, err := pn.runMutators(ctx, hctx, input)
	if err != nil {
		return nil, errors.E(op, pn.pkg.UniquePath, err)
	}

	if err = pn.runValidators(ctx, hctx, mutatedResources); err != nil {
		return nil, errors.E(op, pn.pkg.UniquePath, err)
	}
	// print a new line after a pipeline running
	pr.Printf("\n")
	return mutatedResources, nil
}

// runMutators runs a set of mutators functions on given input resources.
func (pn *pkgNode) runMutators(ctx context.Context, hctx *hydrationContext, input []*yaml.RNode) ([]*yaml.RNode, error) {
	pl, err := pn.pkg.Pipeline()
	if err != nil {
		return nil, err
	}

	if len(pl.Mutators) == 0 {
		return input, nil
	}

	mutators, err := fnChain(ctx, hctx, pn.pkg.UniquePath, pl.Mutators)
	if err != nil {
		return nil, err
	}

	for i, mutator := range mutators {
		if pl.Mutators[i].ConfigPath != "" {
			// kpt v1.0.0-beta15+ onwards, functionConfigs are included in the
			// function inputs during `render` and as a result, they can be
			// mutated during the `render`.
			// So functionConfigs needs be updated in the FunctionRunner instance
			// before every run.
			for _, r := range input {
				pkgPath, err := pkg.GetPkgPathAnnotation(r)
				if err != nil {
					return nil, err
				}
				currPath, _, err := kioutil.GetFileAnnotations(r)
				if err != nil {
					return nil, err
				}
				if pkgPath == pn.pkg.UniquePath.String() && // resource belong to current package
					currPath == pl.Mutators[i].ConfigPath { // configPath matches
					mutator.SetFnConfig(r)
					continue
				}
			}
		}

		selectors := pl.Mutators[i].Selectors
		exclusions := pl.Mutators[i].Exclusions

		if len(selectors) > 0 || len(exclusions) > 0 {
			// set kpt-resource-id annotation on each resource before mutation
			err = fnruntime.SetResourceIds(input)
			if err != nil {
				return nil, err
			}
		}
		// select the resources on which function should be applied
		selectedInput, err := fnruntime.SelectInput(input, selectors, exclusions, &fnruntime.SelectionContext{RootPackagePath: hctx.root.pkg.UniquePath})
		if err != nil {
			return nil, err
		}
		output := &kio.PackageBuffer{}
		// create a kio pipeline from kyaml library to execute the function chains
		mutation := kio.Pipeline{
			Inputs: []kio.Reader{
				&kio.PackageBuffer{Nodes: selectedInput},
			},
			Filters: []kio.Filter{mutator},
			Outputs: []kio.Writer{output},
		}
		err = mutation.Execute()
		if err != nil {
			return nil, err
		}
		hctx.executedFunctionCnt++

		if len(selectors) > 0 || len(exclusions) > 0 {
			// merge the output resources with input resources
			input = fnruntime.MergeWithInput(output.Nodes, selectedInput, input)
			// delete the kpt-resource-id annotation on each resource
			err = fnruntime.DeleteResourceIds(input)
			if err != nil {
				return nil, err
			}
		} else {
			input = output.Nodes
		}
	}
	return input, nil
}

// runValidators runs a set of validator functions on input resources.
// We bail out on first validation failure today, but the logic can be
// improved to report multiple failures. Reporting multiple failures
// will require changes to the way we print errors
func (pn *pkgNode) runValidators(ctx context.Context, hctx *hydrationContext, input []*yaml.RNode) error {
	pl, err := pn.pkg.Pipeline()
	if err != nil {
		return err
	}

	if len(pl.Validators) == 0 {
		return nil
	}

	for i := range pl.Validators {
		function := pl.Validators[i]
		// validators are run on a copy of mutated resources to ensure
		// resources are not mutated.
		selectedResources, err := fnruntime.SelectInput(input, function.Selectors, function.Exclusions, &fnruntime.SelectionContext{RootPackagePath: hctx.root.pkg.UniquePath})
		if err != nil {
			return err
		}
		var validator kio.Filter
		displayResourceCount := false
		if len(function.Selectors) > 0 || len(function.Exclusions) > 0 {
			displayResourceCount = true
		}
		if function.Exec != "" && !hctx.runnerOptions.AllowExec {
			return errAllowedExecNotSpecified
		}
		opts := hctx.runnerOptions
		opts.SetPkgPathAnnotation = true
		opts.DisplayResourceCount = displayResourceCount
		validator, err = fnruntime.NewRunner(ctx, hctx.fileSystem, &function, pn.pkg.UniquePath, hctx.fnResults, opts, hctx.runtime)
		if err != nil {
			return err
		}
		if _, err = validator.Filter(cloneResources(selectedResources)); err != nil {
			return err
		}
		hctx.executedFunctionCnt++
	}
	return nil
}

func cloneResources(input []*yaml.RNode) (output []*yaml.RNode) {
	for _, resource := range input {
		output = append(output, resource.Copy())
	}
	return
}

// path (location) of a KRM resources is tracked in a special key in
// metadata.annotation field that is used to write the resources to the filesystem.
// When resources are read from local filesystem or generated at a package level, the
// path annotation in a resource points to path relative to that package. But the resources
// are written to the file system at the root package level, so
// the path annotation in each resources needs to be adjusted to be relative to the rootPkg.
// adjustRelPath updates the path annotation by prepending the path of the package
// relative to the root package.
func adjustRelPath(hctx *hydrationContext) error {
	resources := hctx.root.resources
	for _, r := range resources {
		pkgPath, err := pkg.GetPkgPathAnnotation(r)
		if err != nil {
			return err
		}
		// Note: kioutil.GetFileAnnotation returns OS specific
		// paths today, https://github.com/kubernetes-sigs/kustomize/issues/3749
		currPath, _, err := kioutil.GetFileAnnotations(r)
		if err != nil {
			return err
		}
		newPath, err := pathRelToRoot(string(hctx.root.pkg.UniquePath), pkgPath, currPath)
		if err != nil {
			return err
		}
		// in kyaml v0.12.0, we are supporting both the new path annotation key
		// internal.config.kubernetes.io/path, as well as the legacy one config.kubernetes.io/path
		if err = r.PipeE(yaml.SetAnnotation(kioutil.PathAnnotation, newPath)); err != nil {
			return err
		}
		if err = r.PipeE(yaml.SetAnnotation(kioutil.LegacyPathAnnotation, newPath)); err != nil { // nolint:staticcheck
			return err
		}
		if err = pkg.RemovePkgPathAnnotation(r); err != nil {
			return err
		}
	}
	return nil
}

// pathRelToRoot computes resource's path relative to root package given:
// rootPkgPath: absolute path to the root package
// subpkgPath: absolute path to subpackage
// resourcePath: resource's path relative to the subpackage
// All the inputs paths are assumed to be OS specific.
func pathRelToRoot(rootPkgPath, subPkgPath, resourcePath string) (relativePath string, err error) {
	if !filepath.IsAbs(rootPkgPath) {
		return "", fmt.Errorf("root package path %q must be absolute", rootPkgPath)
	}

	if !filepath.IsAbs(subPkgPath) {
		return "", fmt.Errorf("subpackage path %q must be absolute", subPkgPath)
	}

	if subPkgPath == "" {
		// empty subpackage path means resource belongs to the root package
		return resourcePath, nil
	}

	// subpackage's path relative to the root package
	subPkgRelPath, err := filepath.Rel(rootPkgPath, subPkgPath)
	if err != nil {
		return "", fmt.Errorf("subpackage %q must be relative to %q: %w",
			rootPkgPath, subPkgPath, err)
	}
	// Note: Rel("/tmp", "/a") = "../", which isn't valid for our use-case.
	dotdot := ".." + string(os.PathSeparator)
	if strings.HasPrefix(subPkgRelPath, dotdot) || subPkgRelPath == ".." {
		return "", fmt.Errorf("subpackage %q is not a descendant of %q", subPkgPath, rootPkgPath)
	}
	relativePath = filepath.Join(subPkgRelPath, filepath.Clean(resourcePath))
	return relativePath, nil
}

// fnChain returns a slice of function runners given a list of functions defined in pipeline.
func fnChain(ctx context.Context, hctx *hydrationContext, pkgPath types.UniquePath, fns []kptfilev1.Function) ([]*fnruntime.FunctionRunner, error) {
	var runners []*fnruntime.FunctionRunner
	for i := range fns {
		var err error
		var runner *fnruntime.FunctionRunner
		function := fns[i]
		displayResourceCount := false
		if len(function.Selectors) > 0 || len(function.Exclusions) > 0 {
			displayResourceCount = true
		}
		if function.Exec != "" && !hctx.runnerOptions.AllowExec {
			return nil, errAllowedExecNotSpecified
		}
		opts := hctx.runnerOptions
		opts.SetPkgPathAnnotation = true
		opts.DisplayResourceCount = displayResourceCount
		runner, err = fnruntime.NewRunner(ctx, hctx.fileSystem, &function, pkgPath, hctx.fnResults, opts, hctx.runtime)
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return runners, nil
}

// trackInputFiles records file paths of input resources in the hydration context.
func trackInputFiles(hctx *hydrationContext, relPath string, input []*yaml.RNode) error {
	if hctx.inputFiles == nil {
		hctx.inputFiles = sets.String{}
	}
	for _, r := range input {
		path, _, err := kioutil.GetFileAnnotations(r)
		if err != nil {
			return fmt.Errorf("path annotation missing: %w", err)
		}
		path = filepath.Join(relPath, filepath.Clean(path))
		hctx.inputFiles.Insert(path)
	}
	return nil
}

// trackOutputFiles records the file paths of output resources in the hydration
// context. It should be invoked post hydration.
func trackOutputFiles(hctx *hydrationContext) error {
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

// pruneResources compares the input and output of the hydration and prunes
// resources that are no longer present in the output of the hydration.
func pruneResources(fsys filesys.FileSystem, hctx *hydrationContext) error {
	filesToBeDeleted := hctx.inputFiles.Difference(hctx.outputFiles)
	for f := range filesToBeDeleted {
		if err := fsys.RemoveAll(filepath.Join(string(hctx.root.pkg.UniquePath), f)); err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}
	return nil
}
