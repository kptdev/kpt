// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdeval

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/GoogleContainerTools/kpt/thirdparty/kyaml/runfn"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/comments"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/order"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetEvalFnRunner returns a EvalFnRunner.
func GetEvalFnRunner(ctx context.Context, parent string) *EvalFnRunner {
	r := &EvalFnRunner{Ctx: ctx}
	r.InitDefaults()

	c := &cobra.Command{
		Use:     "eval [DIR | -] [flags] [--fn-args]",
		Short:   docs.EvalShort,
		Long:    docs.EvalShort + "\n" + docs.EvalLong,
		Example: docs.EvalExamples,
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}
	r.Command = c
	r.Command.Flags().StringVarP(&r.Dest, "output", "o", "",
		fmt.Sprintf("output resources are written to provided location. Allowed values: %s|%s|<OUT_DIR_PATH>", cmdutil.Stdout, cmdutil.Unwrap))
	r.Command.Flags().StringVarP(
		&r.Image, "image", "i", "", "run this image as a function")
	_ = r.Command.RegisterFlagCompletionFunc("image", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.SuggestFunctions(cmd), cobra.ShellCompDirectiveDefault
	})
	r.Command.Flags().StringArrayVarP(
		&r.Keywords, "keywords", "k", nil, "filter functions that match one or more keywords")
	_ = r.Command.RegisterFlagCompletionFunc("keywords", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.SuggestKeywords(cmd), cobra.ShellCompDirectiveDefault
	})
	r.Command.Flags().StringVarP(&r.FnType, "type", "t", "",
		"`mutator` (default) or `validator`. tell the function type for autocompletion and `--save` flag")
	_ = r.Command.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"mutator", "validator"}, cobra.ShellCompDirectiveDefault
	})
	r.Command.Flags().BoolVarP(
		&r.SaveFn, "save", "s", false,
		"save the function and its arguments to Kptfile")
	r.Command.Flags().StringVar(
		&r.Exec, "exec", "", "run an executable as a function")
	r.Command.Flags().StringVar(
		&r.FnConfigPath, "fn-config", "", "path to the function config file")
	r.Command.Flags().BoolVarP(
		&r.IncludeMetaResources, "include-meta-resources", "m", false, "include package meta resources in function input")
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

	r.Command.Flags().Var(&r.RunnerOptions.ImagePullPolicy, "image-pull-policy",
		"pull image before running the container "+r.RunnerOptions.ImagePullPolicy.HelpAllowedValues())
	_ = r.Command.RegisterFlagCompletionFunc("image-pull-policy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return r.RunnerOptions.ImagePullPolicy.AllStrings(), cobra.ShellCompDirectiveDefault
	})

	r.Command.Flags().BoolVar(
		&r.RunnerOptions.AllowWasm, "allow-alpha-wasm", false, "allow alpha wasm functions to be run. If true, you can specify a wasm image with --image flag or a path to a wasm file (must have the .wasm file extension) with --exec flag.")

	// selector flags
	r.Command.Flags().StringVar(
		&r.Selector.APIVersion, "match-api-version", "", "select resources matching the given apiVersion")
	r.Command.Flags().StringVar(
		&r.Selector.Kind, "match-kind", "", "select resources matching the given kind")
	r.Command.Flags().StringVar(
		&r.Selector.Name, "match-name", "", "select resources matching the given name")
	r.Command.Flags().StringVar(
		&r.Selector.Namespace, "match-namespace", "", "select resources matching the given namespace")
	r.Command.Flags().StringArrayVar(
		&r.selectorAnnotations, "match-annotations", []string{}, "select resources matching the given annotations")
	r.Command.Flags().StringArrayVar(
		&r.selectorLabels, "match-labels", []string{}, "select resources matching the given labels")

	// exclusion flags
	r.Command.Flags().StringVar(
		&r.Exclusion.APIVersion, "exclude-api-version", "", "exclude resources matching the given apiVersion")
	r.Command.Flags().StringVar(
		&r.Exclusion.Kind, "exclude-kind", "", "exclude resources matching the given kind")
	r.Command.Flags().StringVar(
		&r.Exclusion.Name, "exclude-name", "", "exclude resources matching the given name")
	r.Command.Flags().StringVar(
		&r.Exclusion.Namespace, "exclude-namespace", "", "exclude resources matching the given namespace")
	r.Command.Flags().StringArrayVar(
		&r.excludeAnnotations, "exclude-annotations", []string{}, "exclude resources matching the given annotations")
	r.Command.Flags().StringArrayVar(
		&r.excludeLabels, "exclude-labels", []string{}, "exclude resources matching the given labels")

	if err := r.Command.Flags().MarkHidden("include-meta-resources"); err != nil {
		panic(err)
	}
	cmdutil.FixDocs("kpt", parent, c)
	return r
}

func EvalCommand(ctx context.Context, name string) *cobra.Command {
	return GetEvalFnRunner(ctx, name).Command
}

// EvalFnRunner contains the run function
type EvalFnRunner struct {
	Command              *cobra.Command
	Dest                 string
	OutContent           bytes.Buffer
	FromStdin            bool
	Image                string
	SaveFn               bool
	Keywords             []string
	FnType               string
	Exec                 string
	FnConfigPath         string
	ResultsDir           string
	Network              bool
	Mounts               []string
	Env                  []string
	AsCurrentUser        bool
	IncludeMetaResources bool
	Ctx                  context.Context
	Selector             kptfile.Selector
	Exclusion            kptfile.Selector
	dataItems            []string

	RunnerOptions fnruntime.RunnerOptions

	// we will need to parse these values into Selector and Exclusion
	selectorLabels      []string
	selectorAnnotations []string
	excludeLabels       []string
	excludeAnnotations  []string

	runFns runfn.RunFns
}

func (r *EvalFnRunner) InitDefaults() {
	r.RunnerOptions.InitDefaults()
}

func (r *EvalFnRunner) runE(c *cobra.Command, _ []string) error {
	err := runner.HandleError(r.Ctx, r.runFns.Execute())
	if err != nil {
		return err
	}
	if err = cmdutil.WriteFnOutput(r.Dest, r.OutContent.String(), r.FromStdin,
		printer.FromContextOrDie(r.Ctx).OutStream()); err != nil {
		return err
	}
	if r.SaveFn {
		r.SaveFnToKptfile()
	}
	return nil
}

// NewFunction creates a Kptfile.Function object which has the evaluated fn configurations.
// This object can be written to Kptfile `pipeline.mutators`.
func (r *EvalFnRunner) NewFunction() *kptfile.Function {
	newFn := &kptfile.Function{}
	if r.Image != "" {
		newFn.Image = r.Image
	} else {
		newFn.Exec = r.Exec
	}
	if !r.Selector.IsEmpty() {
		newFn.Selectors = []kptfile.Selector{r.Selector}
	}
	if !r.Exclusion.IsEmpty() {
		newFn.Exclusions = []kptfile.Selector{r.Exclusion}
	}
	if r.FnConfigPath != "" {
		fnConfigAbsPath, _, _ := pathutil.ResolveAbsAndRelPaths(r.FnConfigPath)
		pkgAbsPath, _, _ := pathutil.ResolveAbsAndRelPaths(r.runFns.Path)
		newFn.ConfigPath, _ = filepath.Rel(pkgAbsPath, fnConfigAbsPath)
	} else {
		data := map[string]string{}
		for i, s := range r.dataItems {
			kv := strings.SplitN(s, "=", 2)
			if i == 0 && len(kv) == 1 {
				continue
			}
			data[kv[0]] = kv[1]
		}
		if len(data) != 0 {
			newFn.ConfigMap = data
		}
	}
	return newFn
}

// Add the evaluated function to the kptfile.Function list, this Function can either be
// `pipeline.mutators` or `pipeline.validators`
func (r *EvalFnRunner) updateFnList(oldFNs []kptfile.Function) ([]kptfile.Function, string) {
	var newFns []kptfile.Function
	found := false
	newFn := r.NewFunction()
	var message string
	for _, m := range oldFNs {
		switch {
		case m.Image != "" && m.Image == r.Image:
			newFns = append(newFns, *newFn)
			found = true
			message = fmt.Sprintf("Updated %q as %v in the Kptfile.\n", r.Image, r.FnType)
		case m.Exec != "" && m.Exec == r.Exec:
			newFns = append(newFns, *newFn)
			found = true
			message = fmt.Sprintf("Updated %q as %v in the Kptfile.\n", r.Exec, r.FnType)
		default:
			newFns = append(newFns, m)
		}
	}
	if !found {
		newFns = append(newFns, *newFn)
		if newFn.Image != "" {
			message = fmt.Sprintf("Added %q as %v in the Kptfile.\n", r.Image, r.FnType)
		} else if newFn.Exec != "" {
			message = fmt.Sprintf("Added %q as %v in the Kptfile.\n", r.Exec, r.FnType)
		}
	}
	return newFns, message
}

// SaveFnToKptfile adds the evaluated function and its arguments to Kptfile `pipeline.mutators` or `pipeline.validators` .
func (r *EvalFnRunner) SaveFnToKptfile() {
	pr := printer.FromContextOrDie(r.Ctx)
	kf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, r.runFns.Path)
	if err != nil {
		pr.Printf("function not added: Kptfile not exists\n")
		return
	}

	if kf.Pipeline == nil {
		kf.Pipeline = &kptfile.Pipeline{}
	}
	var usrMsg string
	switch r.FnType {
	case "mutator":
		kf.Pipeline.Mutators, usrMsg = r.updateFnList(kf.Pipeline.Mutators)
	case "validator":
		kf.Pipeline.Validators, usrMsg = r.updateFnList(kf.Pipeline.Validators)
	}

	mutatedKfAsYNode, err := r.preserveCommentsAndFieldOrder(kf)
	if err != nil {
		pr.Printf("function is not added to Kptfile: %v\n", err)
	}

	// When saving function to Kptfile, the functionConfig should be the relative path
	// to the kpt package, not the relative path to the current working dir.
	// error handling are ignored since they have been validated in preRunE.
	if err := kptfileutil.WriteFile(r.runFns.Path, mutatedKfAsYNode); err != nil {
		pr.Printf("function is not added to Kptfile: %v\n", err)
		return
	}
	pr.Printf(usrMsg)
}

// preserveCommentsAndFieldOrder syncs the mutated Kptfile with the original to preserve
// comments and field order, and returns the result as a yaml Node
func (r *EvalFnRunner) preserveCommentsAndFieldOrder(kf *kptfile.KptFile) (*yaml.Node, error) {
	kfAsRNode, err := yaml.ReadFile(filepath.Join(r.runFns.Path, kptfile.KptFileName))
	if err != nil {
		return nil, fmt.Errorf("could not read Kptfile: %v", err)
	}
	mutatedKfAsBytes, err := yaml.Marshal(kf)
	if err != nil {
		return nil, fmt.Errorf("could not Marshal Kptfile into bytes: %v", err)
	}
	mutatedKfAsRNode, err := yaml.Parse(string(mutatedKfAsBytes))
	if err != nil {
		return nil, fmt.Errorf("could not parse Kptfile: %v", err)
	}
	// preserve comments and sync field order
	if err := comments.CopyComments(kfAsRNode, mutatedKfAsRNode); err != nil {
		return nil, fmt.Errorf("could not preserve Kptfile comments: %v", err)
	}
	if err := order.SyncOrder(kfAsRNode, mutatedKfAsRNode); err != nil {
		return nil, fmt.Errorf("could not preserve Kptfile field order %v", err)
	}
	return mutatedKfAsRNode.YNode(), nil
}

// getCLIFunctionConfig parses the commandline flags and arguments into explicit
// function config
func (r *EvalFnRunner) getCLIFunctionConfig(ctx context.Context, dataItems []string) (*yaml.RNode, error) {
	if r.Image == "" && r.Exec == "" {
		return nil, nil
	}

	// TODO: This probably doesn't belong here, but moving it changes the test output
	if r.Image != "" {
		img, err := r.RunnerOptions.ResolveToImage(ctx, r.Image)
		if err != nil {
			return nil, err
		}
		r.Image = img
	}

	var err error

	// create the function config
	rc, err := yaml.Parse(`
metadata:
  name: function-input
data: {}
`)
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
			// When we are using a ConfigMap as the functionConfig, we should create
			// the node with type string instead of creating a scalar node. Because
			// a scalar node might be parsed as int, float or bool later.
			err := dataField.PipeE(yaml.SetField(kv[0], yaml.NewStringRNode(kv[1])))
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
	return rc, nil
}

func (r *EvalFnRunner) getFunctionSpec() (*runtimeutil.FunctionSpec, []string, error) {
	fn := &runtimeutil.FunctionSpec{}
	var execArgs []string
	if r.Image != "" {
		if err := kptfile.ValidateFunctionImageURL(r.Image); err != nil {
			return nil, nil, err
		}
		fn.Container.Image = r.Image
	} else if r.Exec != "" {
		// check the flags that doesn't make sense with exec function
		// --mount, --as-current-user, --network and --env are
		// only used with container functions
		if r.AsCurrentUser || r.Network ||
			len(r.Mounts) != 0 || len(r.Env) != 0 {
			return nil, nil, fmt.Errorf("--mount, --as-current-user, --network and --env can only be used with container functions")
		}
		s, err := shlex.Split(r.Exec)
		if err != nil {
			return nil, nil, fmt.Errorf("exec command %q must be valid: %w", r.Exec, err)
		}
		if len(s) > 0 {
			fn.Exec.Path = s[0]
			execArgs = s[1:]
		}
	}
	return fn, execArgs, nil
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

func (r *EvalFnRunner) validateOptionalFlags() error {
	// Let users know that --include-meta-resources is no longer necessary
	// since meta resources are included by default.
	if r.IncludeMetaResources {
		return fmt.Errorf("--include-meta-resources is no longer necessary because meta resources are now included by default")
	}
	// SaveFn stores function to Kptfile. If not enabled, only make in-place changes.
	if r.SaveFn {
		if r.FnType == "" {
			return fmt.Errorf("--type must be specified if saving functions to Kptfile (--save=true)")
		}
		if r.FnType != "mutator" && r.FnType != "validator" {
			return fmt.Errorf("--type must be either `mutator` or `validator`")
		}
	}
	// ResultsDir stores the hydrated output in a structured format to result dir. If not specified, only make
	// in-place changes.
	if r.ResultsDir != "" {
		err := os.MkdirAll(r.ResultsDir, 0755)
		if err != nil {
			return fmt.Errorf("cannot read or create results dir %q: %w", r.ResultsDir, err)
		}
	}

	return nil
}

func (r *EvalFnRunner) preRunE(c *cobra.Command, args []string) error {
	// separate the optional flag validation to fix linter issue: cyclomatic complexity
	if err := r.validateOptionalFlags(); err != nil {
		return err
	}
	if r.Dest != "" && r.Dest != cmdutil.Stdout && r.Dest != cmdutil.Unwrap {
		if err := cmdutil.CheckDirectoryNotPresent(r.Dest); err != nil {
			return err
		}
	}
	if r.Image == "" && r.Exec == "" {
		return errors.Errorf("must specify --image or --exec")
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
	fnConfig, err := r.getCLIFunctionConfig(c.Context(), dataItems)
	if err != nil {
		return err
	}
	r.dataItems = dataItems
	fnSpec, execArgs, err := r.getFunctionSpec()
	if err != nil {
		return err
	}

	// set the output to stdout if in dry-run mode or no arguments are specified
	var output io.Writer
	var input io.Reader
	r.OutContent = bytes.Buffer{}
	if args[0] == "-" {
		output = &r.OutContent
		input = c.InOrStdin()
		r.FromStdin = true

		// clear args as it indicates stdin and not path
		args = []string{}
	} else if r.Dest != "" {
		output = &r.OutContent
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

	if path != "" {
		path, err = argutil.ResolveSymlink(r.Ctx, path)
		if err != nil {
			return err
		}
	}
	if r.SaveFn && r.FnConfigPath != "" {
		fnConfigAbsPath, _, _ := pathutil.ResolveAbsAndRelPaths(r.FnConfigPath)
		pkgAbsPath, _, _ := pathutil.ResolveAbsAndRelPaths(path)
		if !strings.HasPrefix(fnConfigAbsPath, pkgAbsPath) {
			return fmt.Errorf("--fn-config must be under %v if saving functions to Kptfile (--save=true)",
				pkgAbsPath)
		}
	}
	r.parseSelectors()
	r.runFns = runfn.RunFns{
		Ctx:           r.Ctx,
		Function:      fnSpec,
		ExecArgs:      execArgs,
		OriginalExec:  r.Exec,
		Output:        output,
		Input:         input,
		Path:          path,
		Network:       r.Network,
		StorageMounts: storageMounts,
		ResultsDir:    r.ResultsDir,
		Env:           r.Env,
		AsCurrentUser: r.AsCurrentUser,
		FnConfig:      fnConfig,
		FnConfigPath:  r.FnConfigPath,
		// fn eval should remove all files when all resources
		// are deleted.
		ContinueOnEmptyResult: true,
		Selector:              r.Selector,
		Exclusion:             r.Exclusion,
		RunnerOptions:         r.RunnerOptions,
	}

	return nil
}

// parses annotation and label based selectors and exclusion from the command line input
func (r *EvalFnRunner) parseSelectors() {
	r.Selector.Annotations = parseSelectorMap(r.selectorAnnotations)
	r.Selector.Labels = parseSelectorMap(r.selectorLabels)
	r.Exclusion.Annotations = parseSelectorMap(r.excludeAnnotations)
	r.Exclusion.Labels = parseSelectorMap(r.excludeLabels)
}

func parseSelectorMap(selectors []string) map[string]string {
	if len(selectors) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, s := range selectors {
		parts := strings.Split(s, "=")
		key, value := parts[0], parts[1]
		result[key] = value
	}
	return result
}
