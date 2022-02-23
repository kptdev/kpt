package cmdhook

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

var errAllowedExecNotSpecified error = fmt.Errorf("must run with `--allow-exec` option to allow running function binaries")
var ErrHookNotFound error = fmt.Errorf("hook not found")

// Executor executes a hook.
type Executor struct {
	Hook                 string
	PkgPath              string
	ResultsDirPath       string
	Output               io.Writer
	ImagePullPolicy      fnruntime.ImagePullPolicy
	ExcludeMetaResources bool
	AllowExec            bool
}

// Execute runs a pipeline.
func (e *Executor) Execute(ctx context.Context) error {
	const op errors.Op = "fn.render"

	// pr := printer.FromContextOrDie(ctx)

	root, err := pkg.New(e.PkgPath)
	if err != nil {
		return errors.E(op, types.UniquePath(e.PkgPath), err)
	}

	kf, err := root.Kptfile()
	if err != nil {
		return fmt.Errorf("failed to read kptfile: %w", err)
	}

	if kf.Hooks == nil {
		return fmt.Errorf("pkg must have hooks defined")
	}
	hook, found := kf.Hooks[e.Hook]
	if !found {
		return ErrHookNotFound
	}

	// initialize hydration context
	hctx := &hookContext{
		// root:                 root,
		fnResults:            fnresult.NewResultList(),
		imagePullPolicy:      e.ImagePullPolicy,
		allowExec:            e.AllowExec,
		excludeMetaResources: e.ExcludeMetaResources,
	}

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
	mutators, err := fnChain(ctx, hctx, root.UniquePath, hook)
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

// hookContext contains bits to track state of a package hydration.
// This is sort of global state that is available to hydration step at
// each pkg along the hydration walk.
type hookContext struct {
	// fnResults stores function results gathered
	// during pipeline execution.
	fnResults *fnresult.ResultList

	// imagePullPolicy controls the image pulling behavior.
	imagePullPolicy fnruntime.ImagePullPolicy

	// indicate if package meta resources such as Kptfile
	// to be excluded from the function in put during render.
	excludeMetaResources bool

	// allowExec determines if function binary executable are allowed
	// to be run during pipeline execution. Running function binaries is a
	// privileged operation, so explicit permission is required.
	allowExec bool

	// bookkeeping to ensure docker command availability check is done once
	// during rendering
	dockerCheckDone bool
}

// fnChain returns a slice of function runners given a list of functions defined in pipeline.
func fnChain(ctx context.Context, hctx *hookContext, pkgPath types.UniquePath, fns []kptfilev1.Function) ([]kio.Filter, error) {
	var runners []kio.Filter
	for i := range fns {
		var err error
		var runner kio.Filter
		fn := fns[i]
		if fn.Exec != "" && !hctx.allowExec {
			return nil, errAllowedExecNotSpecified
		}
		if fn.Image != "" && !hctx.dockerCheckDone {
			err := cmdutil.DockerCmdAvailable()
			if err != nil {
				return nil, err
			}
			hctx.dockerCheckDone = true
		}
		runner, err = fnruntime.NewRunner(ctx, &fn, pkgPath, hctx.fnResults, hctx.imagePullPolicy, false, false)
		if err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return runners, nil
}
