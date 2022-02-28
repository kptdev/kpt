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
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

var errAllowedExecNotSpecified error = fmt.Errorf("must run with `--allow-exec` option to allow running function binaries")
var ErrHookNotFound error = fmt.Errorf("hook not found")

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
	Runtime fn.FunctionRuntime

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
			return nil, errAllowedExecNotSpecified
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
			true,
			false,
			e.Runtime)
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return runners, nil
}
