package resource

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/cad"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
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
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const gcloudName = "./gcloud-config.yaml"

// This will be replaced by variant constructor
func (r *Setter) GetGcloudFnConfigPath() string {
	return filepath.Join(r.Dest, gcloudName)
}

// THis will be supported by variant constructor
var IncludeMetaResourcesFlag = true

func NewAdd(ctx context.Context) *Setter {
	r := &Setter{ctx: ctx}
	c := &cobra.Command{
		Use:   "resource [--kind=namespace] [--context=false]",
		Short: `Add the KRM resource(s) in the local package`,
		Example: `
  # Set the package resources to the same namespace
  $ kpt editor add resource --kind=namespace
`,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}

	c.Flags().StringVarP(&r.kind, "kind", "k", "", "Kubernetes core resource `Kind`")
	c.RegisterFlagCompletionFunc("kind", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cad.ResourceKinds(), cobra.ShellCompDirectiveDefault
	})

	c.Flags().StringVarP(&r.context, "context", "c", "", "KRM resources correlated to existing `kind`")
	c.RegisterFlagCompletionFunc("context", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return r.GetContextFromSourceLocal(args), cobra.ShellCompDirectiveDefault
	})
	r.Command = c
	return r
}

func (s *Setter) GetContextFromSourceLocal(args []string) []string {
	var inputs []kio.Reader
	matchFilesGlob := kio.MatchAll
	if IncludeMetaResourcesFlag {
		matchFilesGlob = append(matchFilesGlob, kptfile.KptFileName)
	}
	if len(args) == 0 {
		// no pkg path specified, default to current working dir
		wd, err := os.Getwd()
		if err != nil {
			// return err
		}
		s.Dest = wd
	} else {
		// resolve and validate the provided path
		s.Dest = args[0]
	}
	var err error
	s.Dest, err = argutil.ResolveSymlink(s.ctx, s.Dest)
	if err != nil {
		return nil
	}
	resolvedPath, err := argutil.ResolveSymlink(s.ctx, s.Dest)
	if err != nil {
		return nil
	}
	functionConfigFilter, err := pkg.FunctionConfigFilterFunc(types.UniquePath(resolvedPath), IncludeMetaResourcesFlag)
	if err != nil {
		return nil
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
	var writer bytes.Buffer
	outputs = append(outputs, kio.ByteWriter{
		Writer:           &writer,
		FunctionConfig:   nil,
		ClearAnnotations: []string{kioutil.IndexAnnotation, kioutil.PathAnnotation}, // nolint:staticcheck
	})

	resourceReader := &ResoureReader{}
	err = kio.Pipeline{Inputs: inputs, Filters: []kio.Filter{resourceReader}, Outputs: outputs}.Execute()
	if e := runner.HandleError(s.ctx, err); e != nil {
		return nil
	}
	resourceFromCtx := []string{}
	for _, r := range resourceReader.KindLists {
		if rs, ok := cad.ResourceContextMap[r]; ok {
			resourceFromCtx = append(resourceFromCtx, rs...)
		}
	}
	return resourceFromCtx
}

type ResoureReader struct {
	KindLists []string
}

func (r *ResoureReader) Filter(o []*yaml.RNode) ([]*yaml.RNode, error) {
	for _, rn := range o {
		r.KindLists = append(r.KindLists, strings.ToLower(rn.GetKind()))
	}
	return o, nil
}

type Setter struct {
	kind    string
	context string

	// The kpt package directory
	Dest      string
	kf        *kptfile.KptFile
	Command   *cobra.Command
	ctx       context.Context
	fnResults *fnresult.ResultList
}

func (r *Setter) preRunE(c *cobra.Command, args []string) error {
	if r.kind == "" && r.context == "" {
		return fmt.Errorf("must specify either `kind` or `context` flag")
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

func (r *Setter) fromKubectlCreate(kind string) (string, error) {
	var out, errout bytes.Buffer
	args := []string{"create", kind, cad.PlaceHolder, "--dry-run=client", "-oyaml"}
	flagArgs := cad.ResourceKindArgs(kind)
	if flagArgs != nil {
		args = append(args, flagArgs...)
	}
	cmd := exec.Command("kubectl", args...)
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

/*
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
	execName, ok := cad.BuiltinTransformers[kind]
	if !ok {
		return ""
	}
	execPath := filepath.Join(dir, ".kpt", "fns", execName)
	if _, err := os.Stat(execPath); errors.Is(err, os.ErrNotExist) {
		return ""
	}
	return execPath
}
*/

func (r *Setter) runE(c *cobra.Command, args []string) error {
	var kind string
	if r.kind != "" {
		kind = r.kind
	} else {
		kind = r.context
	}
	krmResources, err := r.fromKubectlCreate(kind)
	if err != nil {
		return err
	}
	var inputs []kio.Reader
	if krmResources != "" {
		reader := strings.NewReader(krmResources)
		inputs = append(inputs, &kio.ByteReader{Reader: reader})
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
		Ctx: r.ctx,
		/*
			Function:              fnSpec,
			ExecArgs:              execArgs,
			OriginalExec:          execPath,

		*/
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
