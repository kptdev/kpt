// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package runfn

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/printerutil"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
)

// RunFns runs the set of configuration functions in a local directory against
// the Resources in that directory
type RunFns struct {
	Ctx context.Context

	StorageMounts []runtimeutil.StorageMount

	// Path is the path to the directory containing functions
	Path string

	// uniquePath is the absolute version of Path
	uniquePath types.UniquePath

	// FnConfigPath specifies a config file which contains the configs used in
	// function input. It can be absolute or relative to kpt working directory.
	// The exact format depends on the OS.
	FnConfigPath string

	// Functions is an explicit list of functions to run against the input.
	Functions []*yaml.RNode

	// Input can be set to read the Resources from Input rather than from a directory
	Input io.Reader

	// Network enables network access for functions that declare it
	Network bool

	// Output can be set to write the result to Output rather than back to the directory
	Output io.Writer

	// ResultsDir is where to write each functions results
	ResultsDir string

	fnResults *fnresult.ResultList

	// functionFilterProvider provides a filter to perform the function.
	// this is a variable so it can be mocked in tests
	functionFilterProvider func(
		filter runtimeutil.FunctionSpec, fnConfig *yaml.RNode, currentUser currentUserFunc) (kio.Filter, error)

	// AsCurrentUser is a boolean to indicate whether docker container should use
	// the uid and gid that run the command
	AsCurrentUser bool

	// Env contains environment variables that will be exported to container
	Env []string

	// ContinueOnEmptyResult configures what happens when the underlying pipeline
	// returns an empty result.
	// If it is false (default), subsequent functions will be skipped and the
	// result will be returned immediately.
	// If it is true, the empty result will be provided as input to the next
	// function in the list.
	ContinueOnEmptyResult bool

	// IncludeMetaResources indicates will kpt add pkg meta resources such as
	// Kptfile to the input resources to the function.
	IncludeMetaResources bool

	ImagePullPolicy fnruntime.ImagePullPolicy
}

// Execute runs the command
func (r RunFns) Execute() error {
	// default the containerFilterProvider if it hasn't been override.  Split out for testing.
	err := (&r).init()
	if err != nil {
		return err
	}
	nodes, fltrs, output, err := r.getNodesAndFilters()
	if err != nil {
		return err
	}
	return r.runFunctions(nodes, output, fltrs)
}

// functionConfigFilterFunc returns a kio.LocalPackageSkipFileFunc filter which will be
// invoked by kio.LocalPackageReader when it reads the package. The filter will return
// true if the file should be skipped during reading. Skipped files will not be included
// in all steps following.
func (r RunFns) functionConfigFilterFunc() (kio.LocalPackageSkipFileFunc, error) {
	if r.IncludeMetaResources {
		return func(relPath string) bool {
			return false
		}, nil
	}

	fnConfigPaths, err := pkg.FunctionConfigFilePaths(r.uniquePath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline config file paths: %w", err)
	}

	return func(relPath string) bool {
		if len(fnConfigPaths) == 0 {
			return false
		}
		// relPath is cleaned so we can directly use it here
		return fnConfigPaths.Has(relPath)
	}, nil
}

func (r RunFns) getNodesAndFilters() (
	*kio.PackageBuffer, []kio.Filter, *kio.LocalPackageReadWriter, error) {
	// Read Resources from Directory or Input
	buff := &kio.PackageBuffer{}
	p := kio.Pipeline{Outputs: []kio.Writer{buff}}
	// save the output dir because we will need it to write back
	// the same one for reading must be used for writing if deleting Resources
	var outputPkg *kio.LocalPackageReadWriter
	matchFilesGlob := kio.MatchAll
	if r.IncludeMetaResources {
		matchFilesGlob = append(matchFilesGlob, v1alpha2.KptFileName)
	}
	if r.Path != "" {
		functionConfigFilter, err := r.functionConfigFilterFunc()
		if err != nil {
			return nil, nil, outputPkg, err
		}
		outputPkg = &kio.LocalPackageReadWriter{
			PackagePath:    string(r.uniquePath),
			MatchFilesGlob: matchFilesGlob,
			FileSkipFunc:   functionConfigFilter,
		}
	}

	if r.Input == nil {
		p.Inputs = []kio.Reader{outputPkg}
	} else {
		p.Inputs = []kio.Reader{&kio.ByteReader{Reader: r.Input}}
	}
	if err := p.Execute(); err != nil {
		return nil, nil, outputPkg, err
	}

	fltrs, err := r.getFilters()
	if err != nil {
		return nil, nil, outputPkg, err
	}
	return buff, fltrs, outputPkg, nil
}

func (r RunFns) getFilters() ([]kio.Filter, error) {
	fns := r.Functions
	var fltrs []kio.Filter
	for i := range fns {
		api := fns[i]
		spec := runtimeutil.GetFunctionSpec(api)
		if spec == nil {
			// resource doesn't have function spec
			continue
		}
		if spec.Container.Network && !r.Network {
			// TODO(eddiezane): Provide error info about which function needs the network
			return fltrs, errors.Errorf("network required but not enabled with --network")
		}
		// merge envs from imperative and declarative
		spec.Container.Env = r.mergeContainerEnv(spec.Container.Env)

		c, err := r.functionFilterProvider(*spec, api, user.Current)
		if err != nil {
			return nil, err
		}

		if c == nil {
			continue
		}
		fltrs = append(fltrs, c)
	}
	return fltrs, nil
}

// runFunctions runs the fltrs against the input and writes to either r.Output or output
func (r RunFns) runFunctions(
	input kio.Reader, output kio.Writer, fltrs []kio.Filter) error {
	// use the previously read Resources as input
	var outputs []kio.Writer
	if r.Output == nil {
		// write back to the package
		outputs = append(outputs, output)
	} else {
		// write to the output instead of the directory if r.Output is specified or
		// the output is nil (reading from Input)
		outputs = append(outputs, kio.ByteWriter{Writer: r.Output})
	}

	// add format filter at the end to consistently format output resources
	fmtfltr := filters.FormatFilter{UseSchema: true}
	fltrs = append(fltrs, fmtfltr)

	var err error
	pipeline := kio.Pipeline{
		Inputs:                []kio.Reader{input},
		Filters:               fltrs,
		Outputs:               outputs,
		ContinueOnEmptyResult: r.ContinueOnEmptyResult,
	}
	err = pipeline.Execute()
	resultsFile, resultErr := fnruntime.SaveResults(r.ResultsDir, r.fnResults)
	if err != nil {
		// function fails
		if resultErr == nil {
			r.printFnResultsStatus(resultsFile, true)
		}
		return err
	}
	if resultErr == nil {
		r.printFnResultsStatus(resultsFile, false)
	}
	return nil
}

func (r RunFns) printFnResultsStatus(resultsFile string, toStdErr bool) {
	if r.isOutputDisabled() {
		return
	}
	printerutil.PrintFnResultInfo(r.Ctx, resultsFile, true, toStdErr)
}

// mergeContainerEnv will merge the envs specified by command line (imperative) and config
// file (declarative). If they have same key, the imperative value will be respected.
func (r RunFns) mergeContainerEnv(envs []string) []string {
	imperative := fnruntime.NewContainerEnvFromStringSlice(r.Env)
	declarative := fnruntime.NewContainerEnvFromStringSlice(envs)
	for key, value := range imperative.EnvVars {
		declarative.AddKeyValue(key, value)
	}

	for _, key := range imperative.VarsToExport {
		declarative.AddKey(key)
	}

	return declarative.Raw()
}

// sortFns sorts functions so that functions with the longest paths come first
func sortFns(buff *kio.PackageBuffer) error {
	var outerErr error
	// sort the nodes so that we traverse them depth first
	// functions deeper in the file system tree should be run first
	sort.Slice(buff.Nodes, func(i, j int) bool {
		mi, _ := buff.Nodes[i].GetMeta()
		pi := filepath.ToSlash(mi.Annotations[kioutil.PathAnnotation])

		mj, _ := buff.Nodes[j].GetMeta()
		pj := filepath.ToSlash(mj.Annotations[kioutil.PathAnnotation])

		// If the path is the same, we decide the ordering based on the
		// index annotation.
		if pi == pj {
			iIndex, err := strconv.Atoi(mi.Annotations[kioutil.IndexAnnotation])
			if err != nil {
				outerErr = err
				return false
			}
			jIndex, err := strconv.Atoi(mj.Annotations[kioutil.IndexAnnotation])
			if err != nil {
				outerErr = err
				return false
			}
			return iIndex < jIndex
		}

		if filepath.Base(path.Dir(pi)) == "functions" {
			// don't count the functions dir, the functions are scoped 1 level above
			pi = filepath.Dir(path.Dir(pi))
		} else {
			pi = filepath.Dir(pi)
		}

		if filepath.Base(path.Dir(pj)) == "functions" {
			// don't count the functions dir, the functions are scoped 1 level above
			pj = filepath.Dir(path.Dir(pj))
		} else {
			pj = filepath.Dir(pj)
		}

		// i is "less" than j (comes earlier) if its depth is greater -- e.g. run
		// i before j if it is deeper in the directory structure
		li := len(strings.Split(pi, "/"))
		if pi == "." {
			// local dir should have 0 path elements instead of 1
			li = 0
		}
		lj := len(strings.Split(pj, "/"))
		if pj == "." {
			// local dir should have 0 path elements instead of 1
			lj = 0
		}
		if li != lj {
			// use greater-than because we want to sort with the longest
			// paths FIRST rather than last
			return li > lj
		}

		// sort by path names if depths are equal
		return pi < pj
	})
	return outerErr
}

// init initializes the RunFns with a containerFilterProvider.
func (r *RunFns) init() error {
	// if no path is specified, default reading from stdin and writing to stdout
	if r.Path == "" {
		if r.Output == nil {
			r.Output = os.Stdout
		}
		if r.Input == nil {
			r.Input = os.Stdin
		}
	} else {
		// make the path absolute so it works on mac
		var err error
		absPath, err := filepath.Abs(r.Path)
		if err != nil {
			return errors.Wrap(err)
		}
		r.uniquePath = types.UniquePath(absPath)
	}

	r.fnResults = fnresult.NewResultList()

	// functionFilterProvider set the filter provider
	if r.functionFilterProvider == nil {
		r.functionFilterProvider = r.defaultFnFilterProvider
	}

	// fn config path should be absolute
	if r.FnConfigPath != "" && !filepath.IsAbs(r.FnConfigPath) {
		// if the FnConfigPath is relative, we should use the
		// current directory to construct full path.
		path, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		r.FnConfigPath = filepath.Join(path, r.FnConfigPath)
	}
	return nil
}

type currentUserFunc func() (*user.User, error)

// getUIDGID will return "nobody" if asCurrentUser is false. Otherwise
// return "uid:gid" according to the return from currentUser function.
func getUIDGID(asCurrentUser bool, currentUser currentUserFunc) (string, error) {
	if !asCurrentUser {
		return "nobody", nil
	}

	u, err := currentUser()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", u.Uid, u.Gid), nil
}

// getFunctionConfig returns yaml representation of functionConfig that can
// be provided to a function as input.
func (r *RunFns) getFunctionConfig() (*yaml.RNode, error) {
	return v1alpha2.GetValidatedFnConfigFromPath("", r.FnConfigPath)
}

// defaultFnFilterProvider provides function filters
func (r *RunFns) defaultFnFilterProvider(spec runtimeutil.FunctionSpec, fnConfig *yaml.RNode, currentUser currentUserFunc) (kio.Filter, error) {
	if spec.Container.Image == "" && spec.Exec.Path == "" {
		return nil, fmt.Errorf("either image name or executable path need to be provided")
	}

	var err error
	if r.FnConfigPath != "" {
		fnConfig, err = r.getFunctionConfig()
		if err != nil {
			return nil, err
		}
	}
	var fltr *runtimeutil.FunctionFilter
	fnResult := &fnresult.Result{
		// TODO(droot): This is required for making structured results subpackage aware.
		// Enable this once test harness supports filepath based assertions.
		// Pkg: string(r.uniquePath),
	}
	if spec.Container.Image != "" {
		// TODO: Add a test for this behavior
		uidgid, err := getUIDGID(r.AsCurrentUser, currentUser)
		if err != nil {
			return nil, err
		}
		c := &fnruntime.ContainerFn{
			Path:            r.uniquePath,
			Image:           spec.Container.Image,
			ImagePullPolicy: r.ImagePullPolicy,
			UIDGID:          uidgid,
			StorageMounts:   r.StorageMounts,
			Env:             spec.Container.Env,
			Perm: fnruntime.ContainerFnPermission{
				AllowNetwork: spec.Container.Network,
				// mounts are always from CLI flags so we allow
				// them by default for eval
				AllowMount: true,
			},
		}
		fltr = &runtimeutil.FunctionFilter{
			Run:            c.Run,
			FunctionConfig: fnConfig,
			DeferFailure:   spec.DeferFailure,
		}
		fnResult.Image = spec.Container.Image
	}

	if spec.Exec.Path != "" {
		e := &fnruntime.ExecFn{
			Path: spec.Exec.Path,
		}
		fltr = &runtimeutil.FunctionFilter{
			Run:            e.Run,
			FunctionConfig: fnConfig,
			DeferFailure:   spec.DeferFailure,
		}
		fnResult.ExecPath = spec.Exec.Path
	}
	return fnruntime.NewFunctionRunner(r.Ctx, fltr, r.isOutputDisabled(), fnResult, r.fnResults)
}

func (r RunFns) isOutputDisabled() bool {
	// if output is not nil we will write the resources to stdout
	return r.Output != nil
}
