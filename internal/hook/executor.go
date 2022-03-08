// Copyright 2022 Google LLC
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
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	fnpkg "github.com/GoogleContainerTools/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// ErrAllowedExecNotSpecified indicates user need to authorize to invoke
// exec binaries.
var ErrAllowedExecNotSpecified error = fmt.Errorf("must run with `--allow-exec` option to allow running function binaries")

// Executor executes a hook.
type Executor struct {
	PkgPath              string
	ResultsDirPath       string
	Output               io.Writer
	ImagePullPolicy      fnruntime.ImagePullPolicy
	ExcludeMetaResources bool
	AllowExec            bool

	FileSystem filesys.FileSystem

	// function runtime
	Runtime fnpkg.FunctionRuntime

	// fnResults stores function results gathered
	// during pipeline execution.
	fnResults *fnresult.ResultList

	// bookkeeping to ensure docker command availability check is done once
	// during rendering
	dockerCheckDone bool
}

// Execute executes given hook.
func (e *Executor) Execute(ctx context.Context, hook []kptfilev1.Function) error {
	e.fnResults = fnresult.NewResultList()

	matchFilesGlob := kio.MatchAll
	if !e.ExcludeMetaResources {
		matchFilesGlob = append(matchFilesGlob, kptfilev1.KptFileName)
	}

	pkgReaderWriter := &kio.LocalPackageReadWriter{
		PackagePath:        e.PkgPath,
		MatchFilesGlob:     matchFilesGlob,
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
		if fn.Exec != "" && !e.AllowExec {
			return nil, ErrAllowedExecNotSpecified
		}
		if fn.Image != "" && !e.dockerCheckDone {
			err := cmdutil.DockerCmdAvailable()
			if err != nil {
				return nil, err
			}
			e.dockerCheckDone = true
		}
		runner, err = fnruntime.NewRunner(ctx,
			e.FileSystem,
			&fn,
			types.UniquePath(e.PkgPath),
			e.fnResults,
			e.ImagePullPolicy,
			false, /* do not display resource */
			e.Runtime)
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return runners, nil
}
