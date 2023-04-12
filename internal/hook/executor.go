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

package hook

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// ErrAllowedExecNotSpecified indicates user need to authorize to invoke
// exec binaries.
var ErrAllowedExecNotSpecified = fmt.Errorf("must run with `--allow-exec` option to allow running function binaries")

// Executor executes a hook.
type Executor struct {
	PkgPath        string
	ResultsDirPath string
	Output         io.Writer

	RunnerOptions fnruntime.RunnerOptions

	FileSystem filesys.FileSystem

	// function runtime
	Runtime fn.FunctionRuntime

	// fnResults stores function results gathered
	// during pipeline execution.
	fnResults *fnresult.ResultList
}

// Execute executes given hook.
func (e *Executor) Execute(ctx context.Context, hook []kptfilev1.Function) error {
	e.fnResults = fnresult.NewResultList()

	pkgReaderWriter := &kio.LocalPackageReadWriter{
		PackagePath:        e.PkgPath,
		MatchFilesGlob:     pkg.MatchAllKRM,
		PreserveSeqIndent:  true,
		PackageFileName:    kptfilev1.KptFileName,
		IncludeSubpackages: true,
		WrapBareSeqNode:    true,
	}
	mutators, err := e.fnChain(ctx, hook)
	if err != nil {
		return err
	}
	p := kio.Pipeline{
		Inputs:  []kio.Reader{pkgReaderWriter},
		Filters: mutators,
		Outputs: []kio.Writer{pkgReaderWriter},
	}

	return p.Execute()
}

// fnChain returns a slice of function runners given a list of functions defined in pipeline.
func (e *Executor) fnChain(ctx context.Context, fns []kptfilev1.Function) ([]kio.Filter, error) {
	var runners []kio.Filter
	for i := range fns {
		var err error
		var runner kio.Filter
		fn := fns[i]
		if fn.Exec != "" && !e.RunnerOptions.AllowExec {
			return nil, ErrAllowedExecNotSpecified
		}
		opts := e.RunnerOptions
		runner, err = fnruntime.NewRunner(ctx,
			e.FileSystem,
			&fn,
			types.UniquePath(e.PkgPath),
			e.fnResults,
			opts,
			e.Runtime)
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return runners, nil
}
