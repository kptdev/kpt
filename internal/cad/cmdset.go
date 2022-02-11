package cad

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/GoogleContainerTools/kpt/thirdparty/kyaml/runfn"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

const gcloudName = "./gcloud-config.yaml"

// This will be replaced by variant constructor
func (r *Setter) GetGcloudFnConfigPath() string {
	return filepath.Join(r.Dest, gcloudName)
}

// THis will be supported by variant constructor
var IncludeMetaResourcesFlag = true

func NewSetter(ctx context.Context) *Setter {
	r := &Setter{ctx: ctx}
	c := &cobra.Command{
		Use:   "set [--kind=namespace] [--pkg=redis-bucket]",
		Short: `make the KRM resource(s) available in the local package`,
		Example: `
  # Set the package resources to the same namespace
  $ editor set -k=namespace
`,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}

	c.Flags().StringVarP(&r.kind, "kind", "k", "", "KRM resource `Kind`")
	c.RegisterFlagCompletionFunc("kind", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return kubectlKinds, cobra.ShellCompDirectiveDefault
	})

	c.Flags().StringVarP(&r.pkg, "pkg", "p", "",
		"the package name of KRM resources")
	r.Command = c
	return r
}

type Setter struct {
	kind string
	pkg  string

	// The kpt package directory
	Dest      string
	kf        *kptfile.KptFile
	Command   *cobra.Command
	ctx       context.Context
	fnResults *fnresult.ResultList
}

func (r *Setter) preRunE(c *cobra.Command, args []string) error {
	if r.kind == "" && r.pkg == "" {
		return fmt.Errorf("must specify a flag `kind` or a `pkg`")
	}
	if len(args) == 0 {
		// no pkg path specified, default to current working dir
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		r.Dest = wd
	} else {
		// resolve and validate the provided path
		r.Dest = args[0]
	}
	var err error
	r.Dest, err = argutil.ResolveSymlink(r.ctx, r.Dest)
	if err != nil {
		return err
	}
	r.kf, err = pkg.ReadKptfile(r.Dest)
	if err != nil {
		return err
	}
	return nil
}

func (r *Setter) fromKubeclCreate() (string, error) {
	name := r.GetDefaultName()
	if name == "" {
		return "", nil
	}
	var out, errout bytes.Buffer

	cmd := exec.Command("kubectl", "create", r.kind, name, "--dry-run=client", "-oyaml")
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	if errout.String() != "" {
		return "", fmt.Errorf(errout.String())
	}
	return out.String(), nil
}

func (r *Setter) getFunctionSpec(execPath string) (*runtimeutil.FunctionSpec, []string, error) {
	fn := &runtimeutil.FunctionSpec{}
	var execArgs []string
	s, err := shlex.Split(execPath)
	if err != nil {
		return nil, nil, fmt.Errorf("exec command %q must be valid: %w", execPath, err)
	}
	if len(s) > 0 {
		fn.Exec.Path = s[0]
		execArgs = s[1:]
	}
	return fn, execArgs, nil
}

// Should be changed to variant constructor.
func (r *Setter) GetDefaultName() string {
	kindToName := map[string]string{
		"namespace": r.kf.Name,
	}
	if r.kind != "" {
		if name, ok := kindToName[r.kind]; ok {
			return name
		}
	}
	return ""
}

// 	GetExeFnsPath choose which kpt fn executable(s) to run for the given `Kind`.
//	The mapping between fn and resource `kind` will be done by function/pkg discovery mechanism.
// TODO: run multiple fn execs in series (like render pipeline)
func GetExeFnsPath(kind string) string {
	dir := os.Getenv(gitutil.RepoCacheDirEnv)
	if dir != "" {
		return dir
	}
	// cache location unspecified, use UserHomeDir/.kpt/repos
	dir, _ = os.UserHomeDir()
	execName, ok := BuiltinTransformers[kind]
	if !ok {
		return ""
	}
	execPath := filepath.Join(dir, ".kpt", "fns", execName)
	if _, err := os.Stat(execPath); errors.Is(err, os.ErrNotExist) {
		return ""
	}
	return execPath
}

func (r *Setter) runE(c *cobra.Command, _ []string) error {
	var inputs []kio.Reader
	// Leverage kubectl to create k8s Resource (caveat, name is required --> r.kf.name )
	if r.kind != "" {
		krmResources, err := r.fromKubeclCreate()
		if err != nil {
			return err
		}
		if krmResources != "" {
			reader := strings.NewReader(krmResources)
			inputs = append(inputs, &kio.ByteReader{Reader: reader})
		}
	}
	// find the fn exec that should mutate/validate this `kind` resource.
	execPath := GetExeFnsPath(r.kind)
	if execPath == "" {
		// TODO: write resource to pkg dir.
		return nil
	}
	fnSpec, execArgs, err := r.getFunctionSpec(execPath)
	if err != nil {
		return err
	}

	matchFilesGlob := kio.MatchAll
	if IncludeMetaResourcesFlag {
		matchFilesGlob = append(matchFilesGlob, kptfile.KptFileName)
	}
	resolvedPath, err := argutil.ResolveSymlink(r.ctx, r.Dest)
	if err != nil {
		return err
	}
	functionConfigFilter, err := pkg.FunctionConfigFilterFunc(types.UniquePath(resolvedPath), IncludeMetaResourcesFlag)
	if err != nil {
		return err
	}

	inputs = append(inputs, kio.LocalPackageReader{
		PackagePath:        resolvedPath,
		MatchFilesGlob:     matchFilesGlob,
		FileSkipFunc:       functionConfigFilter,
		PreserveSeqIndent:  true,
		PackageFileName:    kptfile.KptFileName,
		IncludeSubpackages: true,
		WrapBareSeqNode:    true,
	})
	var outputs []kio.Writer
	configs, err := kio.LocalPackageReader{PackagePath: r.GetGcloudFnConfigPath(), PreserveSeqIndent: true, WrapBareSeqNode: true}.Read()
	if err != nil {
		return err
	}
	if len(configs) != 1 {
		return fmt.Errorf("expected exactly 1 functionConfig, found %d", len(configs))
	}
	functionConfig := configs[0]

	outputs = append(outputs, kio.ByteWriter{
		Writer:           printer.FromContextOrDie(r.ctx).OutStream(),
		FunctionConfig:   functionConfig,
		ClearAnnotations: []string{kioutil.IndexAnnotation, kioutil.PathAnnotation}, // nolint:staticcheck
	})
	var output io.Writer
	OutContent := bytes.Buffer{}
	output = &OutContent

	runFns := runfn.RunFns{
		Ctx:                   r.ctx,
		Function:              fnSpec,
		ExecArgs:              execArgs,
		OriginalExec:          execPath,
		Output:                output,
		Input:                 nil,
		KIOReaders:            inputs,
		Path:                  r.Dest,
		Network:               false,
		StorageMounts:         nil,
		ResultsDir:            "",
		Env:                   nil,
		AsCurrentUser:         false,
		FnConfig:              nil,
		FnConfigPath:          r.GetGcloudFnConfigPath(),
		IncludeMetaResources:  IncludeMetaResourcesFlag,
		ImagePullPolicy:       fnruntime.IfNotPresentPull,
		ContinueOnEmptyResult: true,
		Selector:              kptfile.Selector{},
	}

	err = runner.HandleError(r.ctx, runFns.Execute())
	if err != nil {
		return err
	}
	return cmdutil.WriteFnOutput(r.Dest, OutContent.String(), false, printer.FromContextOrDie(r.ctx).OutStream())
}
