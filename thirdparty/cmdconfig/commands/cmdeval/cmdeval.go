// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdeval

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/GoogleContainerTools/kpt/thirdparty/kyaml/runfn"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetEvalFnRunner returns a EvalFnRunner.
func GetEvalFnRunner(ctx context.Context, name string) *EvalFnRunner {
	r := &EvalFnRunner{Ctx: ctx}
	c := &cobra.Command{
		Use:     "eval [DIR | -]",
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}

	r.Command = c
	r.Command.Flags().BoolVar(
		&r.DryRun, "dry-run", false, "print results to stdout")
	r.Command.Flags().StringVar(
		&r.Image, "image", "",
		"run this image as a function")
	r.Command.Flags().StringVar(
		&r.ExecPath, "exec-path", "", "run an executable as a function. (Alpha)")
	r.Command.Flags().StringVar(
		&r.FnConfigPath, "fn-config", "", "path to the function config file")
	r.Command.Flags().BoolVar(
		&r.IncludeMetaResources, "include-meta-resources", false, "include package meta resources in function input")
	r.Command.Flags().StringVar(
		&r.ResultsDir, "results-dir", "", "write function results to this dir")
	r.Command.Flags().BoolVar(
		&r.Network, "network", false, "enable network access for functions that declare it")
	r.Command.Flags().StringArrayVar(
		&r.Mounts, "mount", []string{},
		"a list of storage options read from the filesystem")
	r.Command.Flags().StringArrayVarP(
		&r.Env, "env", "e", []string{},
		"a list of environment variables to be used by functions")
	r.Command.Flags().BoolVar(
		&r.AsCurrentUser, "as-current-user", false, "use the uid and gid that kpt is running with to run the function in the container")
	return r
}

func EvalCommand(ctx context.Context, name string) *cobra.Command {
	return GetEvalFnRunner(ctx, name).Command
}

// EvalFnRunner contains the run function
type EvalFnRunner struct {
	Command              *cobra.Command
	DryRun               bool
	Image                string
	ExecPath             string
	FnConfigPath         string
	RunFns               runfn.RunFns
	ResultsDir           string
	Network              bool
	Mounts               []string
	Env                  []string
	AsCurrentUser        bool
	IncludeMetaResources bool
	Ctx                  context.Context
}

func (r *EvalFnRunner) runE(c *cobra.Command, _ []string) error {
	return runner.HandleError(c, r.RunFns.Execute())
}

// getContainerFunctions parses the commandline flags and arguments into explicit
// Functions to run.
// TODO: refactor this function to avoid using annotations in function config.
func (r *EvalFnRunner) getContainerFunctions(dataItems []string) (
	[]*yaml.RNode, error) {

	if r.Image == "" && r.ExecPath == "" {
		return nil, nil
	}

	var fn *yaml.RNode
	var err error

	if r.Image != "" {
		// create the function spec to set as an annotation
		fn, err = yaml.Parse(`container: {}`)
		if err != nil {
			return nil, err
		}
		// TODO: add support network, volumes, etc based on flag values
		err = fn.PipeE(
			yaml.Lookup("container"),
			yaml.SetField("image", yaml.NewScalarRNode(r.Image)))
		if err != nil {
			return nil, err
		}
		if r.Network {
			err = fn.PipeE(
				yaml.Lookup("container"),
				yaml.SetField("network", yaml.NewScalarRNode("true")))
			if err != nil {
				return nil, err
			}
		}
	} else if r.ExecPath != "" {
		// check the flags that doesn't make sense with exec function
		// --mount, --as-current-user, --network and --env are
		// only used with container functions
		if r.AsCurrentUser || r.Network ||
			len(r.Mounts) != 0 || len(r.Env) != 0 {
			return nil, fmt.Errorf("--mount, --as-current-user, --network and --env can only be used with container functions")
		}
		// create the function spec to set as an annotation
		fn, err = yaml.Parse(`exec: {}`)
		if err != nil {
			return nil, err
		}

		err = fn.PipeE(
			yaml.Lookup("exec"),
			yaml.SetField("path", yaml.NewScalarRNode(r.ExecPath)))
		if err != nil {
			return nil, err
		}
	}

	// create the function config
	rc, err := yaml.Parse(`
metadata:
  name: function-input
data: {}
`)
	if err != nil {
		return nil, err
	}

	// set the function annotation on the function config so it
	// is parsed by RunFns
	value, err := fn.String()
	if err != nil {
		return nil, err
	}
	err = rc.PipeE(
		yaml.LookupCreate(yaml.MappingNode, "metadata", "annotations"),
		yaml.SetField(runtimeutil.FunctionAnnotationKey, yaml.NewScalarRNode(value)))
	if err != nil {
		return nil, err
	}

	// default the function config kind to ConfigMap, this may be overridden
	var kind = "ConfigMap"
	var version = "v1"

	// populate the function config with data.  this is a convention for functions
	// to be more commandline friendly
	if len(dataItems) > 0 {
		dataField, err := rc.Pipe(yaml.Lookup("data"))
		if err != nil {
			return nil, err
		}
		for i, s := range dataItems {
			kv := strings.SplitN(s, "=", 2)
			if i == 0 && len(kv) == 1 {
				// first argument may be the kind
				kind = s
				continue
			}
			if len(kv) != 2 {
				return nil, fmt.Errorf("args must have keys and values separated by =")
			}
			err := dataField.PipeE(yaml.SetField(kv[0], yaml.NewScalarRNode(kv[1])))
			if err != nil {
				return nil, err
			}
		}
	}
	err = rc.PipeE(yaml.SetField("kind", yaml.NewScalarRNode(kind)))
	if err != nil {
		return nil, err
	}
	err = rc.PipeE(yaml.SetField("apiVersion", yaml.NewScalarRNode(version)))
	if err != nil {
		return nil, err
	}
	return []*yaml.RNode{rc}, nil
}

func toStorageMounts(mounts []string) []runtimeutil.StorageMount {
	var sms []runtimeutil.StorageMount
	for _, mount := range mounts {
		sms = append(sms, runtimeutil.StringToStorageMount(mount))
	}
	return sms
}

func checkFnConfigPathExistence(path string) error {
	// check does fn config file exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("missing function config file: %s", path)
	}
	return nil
}

func (r *EvalFnRunner) preRunE(c *cobra.Command, args []string) error {
	if r.Image == "" && r.ExecPath == "" {
		return errors.Errorf("must specify --image or --exec-path")
	}
	var dataItems []string
	if c.ArgsLenAtDash() >= 0 {
		dataItems = append(dataItems, args[c.ArgsLenAtDash():]...)
		args = args[:c.ArgsLenAtDash()]
	}
	if len(args) == 0 {
		// default to current working directory
		args = append(args, ".")
	}
	if len(args) > 1 {
		return errors.Errorf("0 or 1 arguments supported, function arguments go after '--'")
	}
	if len(dataItems) > 0 && r.FnConfigPath != "" {
		return fmt.Errorf("function arguments can only be specified without function config file")
	}

	fns, err := r.getContainerFunctions(dataItems)
	if err != nil {
		return err
	}

	// set the output to stdout if in dry-run mode or no arguments are specified
	var output io.Writer
	var input io.Reader
	if args[0] == "-" {
		output = c.OutOrStdout()
		input = c.InOrStdin()
		// clear args as it indicates stdin and not path
		args = []string{}
	} else if r.DryRun {
		output = c.OutOrStdout()
	}

	// set the path if specified as an argument
	var path string
	if len(args) == 1 {
		// argument is the directory
		path = args[0]
	}

	// parse mounts to set storageMounts
	storageMounts := toStorageMounts(r.Mounts)

	if r.FnConfigPath != "" {
		err = checkFnConfigPathExistence(r.FnConfigPath)
		if err != nil {
			return err
		}
	}

	r.RunFns = runfn.RunFns{
		Ctx:                  r.Ctx,
		Functions:            fns,
		Output:               output,
		Input:                input,
		Path:                 path,
		Network:              r.Network,
		StorageMounts:        storageMounts,
		ResultsDir:           r.ResultsDir,
		Env:                  r.Env,
		AsCurrentUser:        r.AsCurrentUser,
		FnConfigPath:         r.FnConfigPath,
		IncludeMetaResources: r.IncludeMetaResources,
		// fn eval should remove all files when all resources
		// are deleted.
		ContinueOnEmptyResult: true,
	}

	// don't consider args for the function
	return nil
}
