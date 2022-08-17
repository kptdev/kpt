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

package fnruntime

import (
	"context"
	goerrors "errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/builtins"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/google/shlex"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	FuncGenPkgContext = "builtins/gen-pkg-context"
)

// NewRunner returns a FunctionRunner given a specification of a function
// and it's config.
func NewRunner(
	ctx context.Context,
	fsys filesys.FileSystem,
	f *kptfilev1.Function,
	pkgPath types.UniquePath,
	fnResults *fnresult.ResultList,
	imagePullPolicy ImagePullPolicy,
	setPkgPathAnnotation, displayResourceCount, allowWasm bool,
	runtime fn.FunctionRuntime,
) (*FunctionRunner, error) {
	config, err := newFnConfig(fsys, f, pkgPath)
	if err != nil {
		return nil, err
	}
	if f.Image != "" {
		f.Image = AddDefaultImagePathPrefix(ctx, f.Image)
	}

	fnResult := &fnresult.Result{
		Image:    f.Image,
		ExecPath: f.Exec,
		// TODO(droot): This is required for making structured results subpackage aware.
		// Enable this once test harness supports filepath based assertions.
		// Pkg: string(pkgPath),
	}

	fltr := &runtimeutil.FunctionFilter{
		FunctionConfig: config,
		// by default, the inner most runtimeutil.FunctionFilter scopes resources to the
		// directory specified by the functionConfig, kpt v1+ doesn't scope resources
		// during function execution, so marking the scope to global.
		// See https://github.com/GoogleContainerTools/kpt/issues/3230 for more details.
		GlobalScope: true,
	}

	if runtime != nil {
		if runner, err := runtime.GetRunner(ctx, f); err != nil {
			return nil, fmt.Errorf("function runtime failed to evaluate function %q: %w", f.Image, err)
		} else if runner != nil {
			fltr.Run = runner.Run
		}
	}
	if fltr.Run == nil {
		if f.Image == FuncGenPkgContext {
			pkgCtxGenerator := &builtins.PackageContextGenerator{}
			fltr.Run = pkgCtxGenerator.Run
		} else {
			switch {
			case f.Image != "":
				// If allowWasm is true, we will use wasm runtime for image field.
				if allowWasm {
					wFn, err := NewWasmFn(NewOciLoader(filepath.Join(os.TempDir(), "kpt-fn-wasm"), f.Image))
					if err != nil {
						return nil, err
					}
					fltr.Run = wFn.Run
				} else {
					cfn := &ContainerFn{
						Path:            pkgPath,
						Image:           f.Image,
						ImagePullPolicy: imagePullPolicy,
						Ctx:             ctx,
						FnResult:        fnResult,
					}
					fltr.Run = cfn.Run
				}
			case f.Exec != "":
				// If allowWasm is true, we will use wasm runtime for exec field.
				if allowWasm {
					wFn, err := NewWasmFn(&FsLoader{Filename: f.Exec})
					if err != nil {
						return nil, err
					}
					fltr.Run = wFn.Run
				} else {
					var execArgs []string
					// assuming exec here
					s, err := shlex.Split(f.Exec)
					if err != nil {
						return nil, fmt.Errorf("exec command %q must be valid: %w", f.Exec, err)
					}
					execPath := f.Exec
					if len(s) > 0 {
						execPath = s[0]
					}
					if len(s) > 1 {
						execArgs = s[1:]
					}
					eFn := &ExecFn{
						Path:     execPath,
						Args:     execArgs,
						FnResult: fnResult,
					}
					fltr.Run = eFn.Run
				}
			default:
				return nil, fmt.Errorf("must specify `exec` or `image` to execute a function")
			}
		}
	}
	return NewFunctionRunner(ctx, fltr, pkgPath, fnResult, fnResults, setPkgPathAnnotation, displayResourceCount, allowWasm)
}

// NewFunctionRunner returns a FunctionRunner given a specification of a function
// and it's config.
func NewFunctionRunner(ctx context.Context,
	fltr *runtimeutil.FunctionFilter,
	pkgPath types.UniquePath,
	fnResult *fnresult.Result,
	fnResults *fnresult.ResultList,
	setPkgPathAnnotation bool,
	displayResourceCount bool,
	wasm bool) (*FunctionRunner, error) {
	name := fnResult.Image
	if name == "" {
		name = fnResult.ExecPath
	}
	// by default, the inner most runtimeutil.FunctionFilter scopes resources to the
	// directory specified by the functionConfig, kpt v1+ doesn't scope resources
	// during function execution, so marking the scope to global.
	// See https://github.com/GoogleContainerTools/kpt/issues/3230 for more details.
	fltr.GlobalScope = true
	return &FunctionRunner{
		ctx:                  ctx,
		name:                 name,
		pkgPath:              pkgPath,
		filter:               fltr,
		fnResult:             fnResult,
		fnResults:            fnResults,
		setPkgPathAnnotation: setPkgPathAnnotation,
		displayResourceCount: displayResourceCount,
		wasm:                 wasm,
	}, nil
}

// FunctionRunner wraps FunctionFilter and implements kio.Filter interface.
type FunctionRunner struct {
	ctx              context.Context
	name             string
	pkgPath          types.UniquePath
	disableCLIOutput bool
	filter           *runtimeutil.FunctionFilter
	fnResult         *fnresult.Result
	fnResults        *fnresult.ResultList
	// when set to true, function runner will set the package path annotation
	// on resources that do not have it set. The resources generated by
	// functions do not have this annotation set.
	setPkgPathAnnotation bool
	displayResourceCount bool
	wasm                 bool
}

func (fr *FunctionRunner) Filter(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	pr := printer.FromContextOrDie(fr.ctx)
	if !fr.disableCLIOutput {
		if fr.wasm {
			pr.Printf("[RUNNING] WASM %q", fr.name)
		} else {
			pr.Printf("[RUNNING] %q", fr.name)
		}
		if fr.displayResourceCount {
			pr.Printf(" on %d resource(s)", len(input))
		}
		pr.Printf("\n")
	}
	t0 := time.Now()
	output, err = fr.do(input)
	if err != nil {
		printOpt := printer.NewOpt()
		pr.OptPrintf(printOpt, "[FAIL] %q in %v\n", fr.name, time.Since(t0).Truncate(time.Millisecond*100))
		printFnResult(fr.ctx, fr.fnResult, printOpt)
		var fnErr *ExecError
		if goerrors.As(err, &fnErr) {
			printFnExecErr(fr.ctx, fnErr)
			return nil, errors.ErrAlreadyHandled
		}
		return nil, err
	}
	if !fr.disableCLIOutput {
		pr.Printf("[PASS] %q in %v\n", fr.name, time.Since(t0).Truncate(time.Millisecond*100))
		printFnResult(fr.ctx, fr.fnResult, printer.NewOpt())
		printFnStderr(fr.ctx, fr.fnResult.Stderr)
	}
	return output, err
}

// SetFnConfig updates the functionConfig for the FunctionRunner instance.
func (fr *FunctionRunner) SetFnConfig(conf *yaml.RNode) {
	fr.filter.FunctionConfig = conf
}

// do executes the kpt function and returns the modified resources.
// fnResult is updated with the function results returned by the kpt function.
func (fr *FunctionRunner) do(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	if krmErr := kptfilev1.AreKRM(input); krmErr != nil {
		return output, fmt.Errorf("input resource list must contain only KRM resources: %s", krmErr.Error())
	}

	fnResult := fr.fnResult
	output, err = fr.filter.Filter(input)

	if fr.setPkgPathAnnotation {
		if pkgPathErr := setPkgPathAnnotationIfNotExist(output, fr.pkgPath); pkgPathErr != nil {
			return output, pkgPathErr
		}
	}
	if pathErr := enforcePathInvariants(output); pathErr != nil {
		return output, pathErr
	}
	if krmErr := kptfilev1.AreKRM(output); krmErr != nil {
		return output, fmt.Errorf("output resource list must contain only KRM resources: %s", krmErr.Error())
	}

	// parse the results irrespective of the success/failure of fn exec
	resultErr := parseStructuredResult(fr.filter.Results, fnResult)
	if resultErr != nil {
		// Not sure if it's a good idea. This may mask the original
		// function exec error. Revisit this if this turns out to be true.
		return output, resultErr
	}
	if err != nil {
		var execErr *ExecError
		if goerrors.As(err, &execErr) {
			fnResult.ExitCode = execErr.ExitCode
			fnResult.Stderr = execErr.Stderr
			fr.fnResults.ExitCode = 1
		}
		// accumulate the results
		fr.fnResults.Items = append(fr.fnResults.Items, *fnResult)
		return output, err
	}
	fnResult.ExitCode = 0
	fr.fnResults.Items = append(fr.fnResults.Items, *fnResult)
	return output, nil
}

func setPkgPathAnnotationIfNotExist(resources []*yaml.RNode, pkgPath types.UniquePath) error {
	for _, r := range resources {
		currPkgPath, err := pkg.GetPkgPathAnnotation(r)
		if err != nil {
			return err
		}
		if currPkgPath == "" {
			if err = pkg.SetPkgPathAnnotation(r, pkgPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseStructuredResult(yml *yaml.RNode, fnResult *fnresult.Result) error {
	if yml.IsNilOrEmpty() {
		return nil
	}
	// Note: TS SDK and Go SDK implements two different formats for the
	// result. Go SDK wraps result items while TS SDK doesn't. So examine
	// if items are wrapped or not to support both the formats for now.
	// Refer to https://github.com/GoogleContainerTools/kpt/pull/1923#discussion_r628604165
	// for some more details.
	if yml.YNode().Kind == yaml.MappingNode {
		// check if legacy structured result wraps ResultItems
		itemsNode, err := yml.Pipe(yaml.Lookup("items"))
		if err != nil {
			return err
		}
		if !itemsNode.IsNilOrEmpty() {
			// if legacy structured result, uplift the items
			yml = itemsNode
		}
	}
	err := yaml.Unmarshal([]byte(yml.MustString()), &fnResult.Results)
	if err != nil {
		return err
	}

	return migrateLegacyResult(yml, fnResult)
}

// migrateLegacyResult populates name and namespace in fnResult.Result if a
// function (e.g. using kyaml Go SDKs) gives results in a schema
// that puts a resourceRef's name and namespace under a metadata field
// TODO: fix upstream (https://github.com/GoogleContainerTools/kpt/issues/2091)
func migrateLegacyResult(yml *yaml.RNode, fnResult *fnresult.Result) error {
	items, err := yml.Elements()
	if err != nil {
		return err
	}

	for i := range items {
		if err = populateResourceRef(items[i], fnResult.Results[i]); err != nil {
			return err
		}
		if err = populateProposedValue(items[i], fnResult.Results[i]); err != nil {
			return err
		}
	}

	return nil
}

func populateProposedValue(item *yaml.RNode, resultItem *framework.Result) error {
	sv, err := item.Pipe(yaml.Lookup("field", "suggestedValue"))
	if err != nil {
		return err
	}
	if sv == nil {
		return nil
	}
	if resultItem.Field == nil {
		resultItem.Field = &framework.Field{}
	}
	resultItem.Field.ProposedValue = sv
	return nil
}

func populateResourceRef(item *yaml.RNode, resultItem *framework.Result) error {
	r, err := item.Pipe(yaml.Lookup("resourceRef", "metadata"))
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}
	nameNode, err := r.Pipe(yaml.Lookup("name"))
	if err != nil {
		return err
	}
	namespaceNode, err := r.Pipe(yaml.Lookup("namespace"))
	if err != nil {
		return err
	}
	if nameNode != nil {
		resultItem.ResourceRef.Name = strings.TrimSpace(nameNode.MustString())
	}
	if namespaceNode != nil {
		namespace := strings.TrimSpace(namespaceNode.MustString())
		if namespace != "" && namespace != "''" {
			resultItem.ResourceRef.Namespace = strings.TrimSpace(namespace)
		}
	}
	return nil
}

// printFnResult prints given function result in a user friendly
// format on kpt CLI.
func printFnResult(ctx context.Context, fnResult *fnresult.Result, opt *printer.Options) {
	pr := printer.FromContextOrDie(ctx)
	if len(fnResult.Results) > 0 {
		// function returned structured results
		var lines []string
		for _, item := range fnResult.Results {
			lines = append(lines, item.String())
		}
		ri := &multiLineFormatter{
			Title:          "Results",
			Lines:          lines,
			TruncateOutput: printer.TruncateOutput,
		}
		pr.OptPrintf(opt, "%s", ri.String())
	}
}

// printFnExecErr prints given ExecError in a user friendly format
// on kpt CLI.
func printFnExecErr(ctx context.Context, fnErr *ExecError) {
	pr := printer.FromContextOrDie(ctx)
	printFnStderr(ctx, fnErr.Stderr)
	pr.Printf("  Exit code: %d\n\n", fnErr.ExitCode)
}

// printFnStderr prints given stdErr in a user friendly format on kpt CLI.
func printFnStderr(ctx context.Context, stdErr string) {
	pr := printer.FromContextOrDie(ctx)
	if len(stdErr) > 0 {
		errLines := &multiLineFormatter{
			Title:          "Stderr",
			Lines:          strings.Split(stdErr, "\n"),
			UseQuote:       true,
			TruncateOutput: printer.TruncateOutput,
		}
		pr.Printf("%s", errLines.String())
	}
}

// path (location) of a KRM resources is tracked in a special key in
// metadata.annotation field. enforcePathInvariants throws an error if there is a path
// to a file outside the package, or if the same index/path is on multiple resources
func enforcePathInvariants(nodes []*yaml.RNode) error {
	// map has structure pkgPath-->path -> index -> bool
	// to keep track of paths and indexes found
	pkgPaths := make(map[string]map[string]map[string]bool)
	for _, node := range nodes {
		pkgPath, err := pkg.GetPkgPathAnnotation(node)
		if err != nil {
			return err
		}
		if pkgPaths[pkgPath] == nil {
			pkgPaths[pkgPath] = make(map[string]map[string]bool)
		}
		currPath, index, err := kioutil.GetFileAnnotations(node)
		if err != nil {
			return err
		}
		fp := path.Clean(currPath)
		if strings.HasPrefix(fp, "../") {
			return fmt.Errorf("function must not modify resources outside of package: resource has path %s", currPath)
		}
		if pkgPaths[pkgPath][fp] == nil {
			pkgPaths[pkgPath][fp] = make(map[string]bool)
		}
		if _, ok := pkgPaths[pkgPath][fp][index]; ok {
			return fmt.Errorf("resource at path %q and index %q already exists", fp, index)
		}
		pkgPaths[pkgPath][fp][index] = true
	}
	return nil
}

// multiLineFormatter knows how to format multiple lines in pretty format
// that can be displayed to an end user.
type multiLineFormatter struct {
	// Title under which lines need to be printed
	Title string

	// Lines to be printed on the CLI.
	Lines []string

	// TruncateOuput determines if output needs to be truncated or not.
	TruncateOutput bool

	// MaxLines to be printed if truncation is enabled.
	MaxLines int

	// UseQuote determines if line needs to be quoted or not
	UseQuote bool
}

// String returns multiline string.
func (ri *multiLineFormatter) String() string {
	if ri.MaxLines == 0 {
		ri.MaxLines = FnExecErrorTruncateLines
	}
	strInterpolator := "%s"
	if ri.UseQuote {
		strInterpolator = "%q"
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s:\n", ri.Title))
	lineIndent := strings.Repeat(" ", FnExecErrorIndentation+2)
	if !ri.TruncateOutput {
		// stderr string should have indentations
		for _, s := range ri.Lines {
			// suppress newlines to avoid poor formatting
			s = strings.ReplaceAll(s, "\n", " ")
			b.WriteString(fmt.Sprintf(lineIndent+strInterpolator+"\n", s))
		}
		return b.String()
	}
	printedLines := 0
	for i, s := range ri.Lines {
		if i >= ri.MaxLines {
			break
		}
		// suppress newlines to avoid poor formatting
		s = strings.ReplaceAll(s, "\n", " ")
		b.WriteString(fmt.Sprintf(lineIndent+strInterpolator+"\n", s))
		printedLines++
	}
	truncatedLines := len(ri.Lines) - printedLines
	if truncatedLines > 0 {
		b.WriteString(fmt.Sprintf(lineIndent+"...(%d line(s) truncated, use '--truncate-output=false' to disable)\n", truncatedLines))
	}
	return b.String()
}

func newFnConfig(fsys filesys.FileSystem, f *kptfilev1.Function, pkgPath types.UniquePath) (*yaml.RNode, error) {
	const op errors.Op = "fn.readConfig"
	fn := errors.Fn(f.Image)

	var node *yaml.RNode
	switch {
	case f.ConfigPath != "":
		path := filepath.Join(string(pkgPath), f.ConfigPath)
		file, err := fsys.Open(path)
		if err != nil {
			return nil, errors.E(op, fn,
				fmt.Errorf("missing function config %q", f.ConfigPath))
		}
		defer file.Close()
		b, err := io.ReadAll(file)
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		node, err = yaml.Parse(string(b))
		if err != nil {
			return nil, errors.E(op, fn, fmt.Errorf("invalid function config %q %w", f.ConfigPath, err))
		}
		// directly use the config from file
		return node, nil
	case len(f.ConfigMap) != 0:
		configNode, err := NewConfigMap(f.ConfigMap)
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		return configNode, nil
	}
	// no need to return ConfigMap if no config given
	return nil, nil
}
