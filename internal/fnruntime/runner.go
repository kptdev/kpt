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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1alpha2"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// NewContainerRunner returns a kio.Filter given a specification of a container function
// and it's config.
func NewContainerRunner(ctx context.Context, f *kptfilev1alpha2.Function, pkgPath types.UniquePath, fnResults *fnresult.ResultList) (kio.Filter, error) {
	config, err := newFnConfig(f, pkgPath)
	if err != nil {
		return nil, err
	}
	cfn := &ContainerFn{
		Path:  pkgPath,
		Image: f.Image,
		Ctx:   ctx,
	}
	fltr := &runtimeutil.FunctionFilter{
		Run:            cfn.Run,
		FunctionConfig: config,
	}
	return NewFunctionRunner(ctx, fltr, f.Image, false, fnResults)
}

// NewFunctionRunner returns a kio.Filter given a specification of a function
// and it's config.
func NewFunctionRunner(ctx context.Context, fltr *runtimeutil.FunctionFilter, name string, disableOutput bool, fnResults *fnresult.ResultList) (kio.Filter, error) {
	return &FunctionRunner{
		ctx:           ctx,
		name:          name,
		filter:        fltr,
		disableOutput: disableOutput,
		fnResults:     fnResults,
	}, nil
}

// FunctionRunner wraps FunctionFilter and implements kio.Filter interface.
type FunctionRunner struct {
	ctx           context.Context
	name          string
	disableOutput bool
	filter        *runtimeutil.FunctionFilter
	fnResults     *fnresult.ResultList
}

func (fr *FunctionRunner) Filter(input []*yaml.RNode) (output []*yaml.RNode, err error) {

	if fr.disableOutput {
		output, err = fr.do(input)
	} else {
		pr := printer.FromContextOrDie(fr.ctx)
		printOpt := printer.NewOpt()
		pr.OptPrintf(printOpt, "[RUNNING] %q\n", fr.name)
		output, err = fr.do(input)
		if err != nil {
			pr.OptPrintf(printOpt, "[FAIL] %q\n", fr.name)
			var fnErr *errors.FnExecError
			if goerrors.As(err, &fnErr) {
				pr.OptPrintf(printOpt.Stderr(), "%s\n", fnErr.String())
				return nil, errors.ErrAlreadyHandled
			}
			return nil, err
		}
		// capture the result from running the function
		pr.OptPrintf(printOpt, "[PASS] %q\n", fr.name)
	}
	return output, err
}

/*
func printResults(ctx context.Context, fnResult *fnresult.Result) {
	pr := printer.FromContextOrDie(ctx)
	printOpt := printer.NewOpt()
	for _, item := range fnResult.Results {
		pr.OptPrintf(printOpt, "\t[%s] %s  : \n", strings.ToUpper(string(item.Severity)), item.Message)
	}
} */

func (fr *FunctionRunner) do(input []*yaml.RNode) (output []*yaml.RNode, err error) {
	fnResult := &fnresult.Result{Image: fr.name}

	output, err = fr.filter.Filter(input)

	// parse the results irrespective of the success/failure of fn exec
	resultErr := parseStructuredResult(fr.filter.Results, fnResult)
	if resultErr != nil {
		// Not sure if it's a good idea. This may mask the original
		// function exec error. Revisit this if this turns out to be true.
		return output, resultErr
	}
	if err != nil {
		var fnErr *errors.FnExecError
		if goerrors.As(err, &fnErr) {
			fnResult.ExitCode = fnErr.ExitCode
			fnResult.Stderr = fnErr.Stderr
			// fnErr.FnResult = fnResult
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

func parseStructuredResult(yml *yaml.RNode, fnResult *fnresult.Result) error {
	if yml.IsNilOrEmpty() {
		return nil
	}
	// TODO(droot): checking for legacy structured result format
	// can be made more robust. Want to revisit if structure change
	// is offering any benefit ?
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
	return nil
}

func newFnConfig(f *kptfilev1alpha2.Function, pkgPath types.UniquePath) (*yaml.RNode, error) {
	const op errors.Op = "fn.readConfig"
	var fn errors.Fn = errors.Fn(f.Image)

	var node *yaml.RNode
	switch {
	case f.ConfigPath != "":
		path := filepath.Join(string(pkgPath), f.ConfigPath)
		file, err := os.Open(path)
		if err != nil {
			return nil, errors.E(op, fn,
				fmt.Errorf("missing function config %q", f.ConfigPath))
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		node, err = yaml.Parse(string(b))
		if err != nil {
			return nil, errors.E(op, fn, fmt.Errorf("invalid function config %q %w", f.ConfigPath, err))
		}
		// directly use the config from file
		return node, nil
	case !kptfilev1alpha2.IsNodeZero(&f.Config):
		// directly use the inline config
		return yaml.NewRNode(&f.Config), nil
	case len(f.ConfigMap) != 0:
		node = yaml.NewMapRNode(&f.ConfigMap)
		if node == nil {
			return nil, nil
		}
		// create a ConfigMap only for configMap config
		configNode := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {}
`)
		err := configNode.PipeE(yaml.SetField("data", node))
		if err != nil {
			return nil, errors.E(op, fn, err)
		}
		return configNode, nil
	}
	// no need to return ConfigMap if no config given
	return nil, nil
}
