// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package runfn

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/printerutil"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
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

	// Function is an function to run against the input.
	Function *runtimeutil.FunctionSpec

	// FnConfig is the configurations passed from command line
	FnConfig *yaml.RNode

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

	RunnerOptions fnruntime.RunnerOptions

	// ExecArgs are the arguments for exec commands
	ExecArgs []string

	// OriginalExec is the original exec commands
	OriginalExec string

	Selector kptfile.Selector

	Exclusion kptfile.Selector
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

func (r RunFns) getNodesAndFilters() (
	*kio.PackageBuffer, []kio.Filter, *kio.LocalPackageReadWriter, error) {
	// Read Resources from Directory or Input
	buff := &kio.PackageBuffer{}
	p := kio.Pipeline{Outputs: []kio.Writer{buff}}
	// save the output dir because we will need it to write back
	// the same one for reading must be used for writing if deleting Resources
	var outputPkg *kio.LocalPackageReadWriter

	if r.Path != "" {
		outputPkg = &kio.LocalPackageReadWriter{
			PackagePath:        string(r.uniquePath),
			MatchFilesGlob:     pkg.MatchAllKRM,
			PreserveSeqIndent:  true,
			PackageFileName:    kptfile.KptFileName,
			IncludeSubpackages: true,
			WrapBareSeqNode:    true,
		}
	}

	if r.Input == nil {
		p.Inputs = []kio.Reader{outputPkg}
	} else {
		p.Inputs = []kio.Reader{&kio.ByteReader{Reader: r.Input, PreserveSeqIndent: true, WrapBareSeqNode: true}}
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
	spec := r.Function
	if spec == nil {
		return nil, nil
	}
	// merge envs from imperative and declarative
	spec.Container.Env = r.mergeContainerEnv(spec.Container.Env)

	c, err := r.functionFilterProvider(*spec, r.FnConfig, user.Current)
	if err != nil {
		return nil, err
	}

	if c == nil {
		return nil, nil
	}
	return []kio.Filter{c}, nil
}

// runFunctions runs the fltrs against the input and writes to either r.Output or output
func (r RunFns) runFunctions(input kio.Reader, output kio.Writer, fltrs []kio.Filter) error {
	// use the previously read Resources as input
	var outputs []kio.Writer
	if r.Output == nil {
		// write back to the package
		outputs = append(outputs, output)
	} else {
		// write to the output instead of the directory if r.Output is specified or
		// the output is nil (reading from Input)
		outputs = append(outputs, kio.ByteWriter{
			Writer:                r.Output,
			KeepReaderAnnotations: true,
			WrappingKind:          kio.ResourceListKind,
			WrappingAPIVersion:    kio.ResourceListAPIVersion,
		})
	}

	inputResources, err := input.Read()
	if err != nil {
		return err
	}

	selectedInput := inputResources

	if !r.Selector.IsEmpty() || !r.Exclusion.IsEmpty() {
		err = fnruntime.SetResourceIds(inputResources)
		if err != nil {
			return err
		}

		// select the resources on which function should be applied
		selectedInput, err = fnruntime.SelectInput(
			inputResources,
			[]kptfile.Selector{r.Selector},
			[]kptfile.Selector{r.Exclusion},
			&fnruntime.SelectionContext{RootPackagePath: r.uniquePath})
		if err != nil {
			return err
		}
	}

	pb := &kio.PackageBuffer{}
	pipeline := kio.Pipeline{
		Inputs:                []kio.Reader{&kio.PackageBuffer{Nodes: selectedInput}},
		Filters:               fltrs,
		Outputs:               []kio.Writer{pb},
		ContinueOnEmptyResult: r.ContinueOnEmptyResult,
	}
	err = pipeline.Execute()
	outputResources := pb.Nodes

	if !r.Selector.IsEmpty() || !r.Exclusion.IsEmpty() {
		outputResources = fnruntime.MergeWithInput(pb.Nodes, selectedInput, inputResources)
		deleteAnnoErr := fnruntime.DeleteResourceIds(outputResources)
		if deleteAnnoErr != nil {
			return deleteAnnoErr
		}
	}

	if err == nil {
		writeErr := outputs[0].Write(outputResources)
		if writeErr != nil {
			return writeErr
		}
	}
	resultsFile, resultErr := fnruntime.SaveResults(filesys.FileSystemOrOnDisk{}, r.ResultsDir, r.fnResults)
	if err != nil {
		// function fails
		if resultErr == nil {
			r.printFnResultsStatus(resultsFile)
		}
		return err
	}
	if resultErr == nil {
		r.printFnResultsStatus(resultsFile)
	}
	return nil
}

func (r RunFns) printFnResultsStatus(resultsFile string) {
	printerutil.PrintFnResultInfo(r.Ctx, resultsFile, true)
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

// init initializes the RunFns with a containerFilterProvider.
func (r *RunFns) init() error {
	// if no path is specified, default reading from stdin and writing to stdout
	if r.Path == "" {
		if r.Output == nil {
			r.Output = printer.FromContextOrDie(r.Ctx).OutStream()
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
	return kptfile.GetValidatedFnConfigFromPath(filesys.FileSystemOrOnDisk{}, "", r.FnConfigPath)
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
	fltr := &runtimeutil.FunctionFilter{
		FunctionConfig: fnConfig,
		DeferFailure:   spec.DeferFailure,
	}
	fnResult := &fnresult.Result{
		// TODO(droot): This is required for making structured results subpackage aware.
		// Enable this once test harness supports filepath based assertions.
		// Pkg: string(r.uniquePath),
	}
	if spec.Container.Image != "" {
		fnResult.Image = spec.Container.Image

		resolvedImage, err := r.RunnerOptions.ResolveToImage(context.TODO(), spec.Container.Image)
		if err != nil {
			return nil, err
		}
		// If AllowWasm is true, we try to use the image field as a wasm image.
		// TODO: we can be smarter here. If the image doesn't support wasm/js platform,
		// it should fallback to run it as container fn.
		if r.RunnerOptions.AllowWasm {
			wFn, err := fnruntime.NewWasmFn(fnruntime.NewOciLoader(filepath.Join(os.TempDir(), "kpt-fn-wasm"), resolvedImage))
			if err != nil {
				return nil, err
			}
			fltr.Run = wFn.Run
		} else {
			// TODO: Add a test for this behavior
			uidgid, err := getUIDGID(r.AsCurrentUser, currentUser)
			if err != nil {
				return nil, err
			}
			c := &fnruntime.ContainerFn{
				Image:           resolvedImage,
				ImagePullPolicy: r.RunnerOptions.ImagePullPolicy,
				UIDGID:          uidgid,
				StorageMounts:   r.StorageMounts,
				Env:             spec.Container.Env,
				FnResult:        fnResult,
				Perm: fnruntime.ContainerFnPermission{
					AllowNetwork: r.Network,
					// mounts are always from CLI flags so we allow
					// them by default for eval
					AllowMount: true,
				},
			}
			fltr.Run = c.Run
		}
	}

	if spec.Exec.Path != "" {
		fnResult.ExecPath = r.OriginalExec

		if r.RunnerOptions.AllowWasm && strings.HasSuffix(spec.Exec.Path, ".wasm") {
			wFn, err := fnruntime.NewWasmFn(&fnruntime.FsLoader{Filename: spec.Exec.Path})
			if err != nil {
				return nil, err
			}
			fltr.Run = wFn.Run
		} else {
			e := &fnruntime.ExecFn{
				Path:     spec.Exec.Path,
				Args:     r.ExecArgs,
				FnResult: fnResult,
			}
			fltr.Run = e.Run
		}
	}

	opts := r.RunnerOptions
	if !r.Selector.IsEmpty() || !r.Exclusion.IsEmpty() {
		opts.DisplayResourceCount = true
	}

	return fnruntime.NewFunctionRunner(r.Ctx, fltr, "", fnResult, r.fnResults, opts)
}
