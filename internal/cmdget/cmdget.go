// Copyright 2019 Google LLC
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

// Package cmdget contains the get command
package cmdget

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/GoogleContainerTools/kpt/thirdparty/kyaml/runfn"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const GeneratedDir = "generated"

// NewRunner returns a command runner
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:        "get REPO_URI[.git]/PKG_PATH[@VERSION] [LOCAL_DEST_DIRECTORY]",
		Args:       cobra.MinimumNArgs(1),
		Short:      docs.GetShort,
		Long:       docs.GetShort + "\n" + docs.GetLong,
		Example:    docs.GetExamples,
		RunE:       r.runE,
		PreRunE:    r.preRunE,
		SuggestFor: []string{"clone", "cp", "fetch"},
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	c.Flags().StringVar(&r.strategy, "strategy", string(kptfilev1.ResourceMerge),
		"update strategy that should be used when updating this package -- must be one of: "+
			strings.Join(kptfilev1.UpdateStrategiesAsStrings(), ","))
	c.Flags().BoolVar(&r.ReserveBuiltin, "reserve-builtin", false, "")
	c.Flags().BoolVar(&r.isDeploymentInstance, "for-deployment", false,
		"(Experimental) indicates if this package will be deployed to a cluster.")
	_ = c.RegisterFlagCompletionFunc("strategy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return kptfilev1.UpdateStrategiesAsStrings(), cobra.ShellCompDirectiveDefault
	})
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function
type Runner struct {
	ctx                  context.Context
	Get                  get.Command
	Command              *cobra.Command
	strategy             string
	isDeploymentInstance bool
	ReserveBuiltin       bool
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = "cmdget.preRunE"
	if len(args) == 1 {
		args = append(args, pkg.CurDir)
	} else {
		_, err := os.Lstat(args[1])
		if err == nil || os.IsExist(err) {
			resolvedPath, err := argutil.ResolveSymlink(r.ctx, args[1])
			if err != nil {
				return errors.E(op, err)
			}
			args[1] = resolvedPath
		}
	}
	t, err := parse.GitParseArgs(r.ctx, args)
	if err != nil {
		return errors.E(op, err)
	}

	r.Get.Git = &t.Git
	absDestPath, _, err := pathutil.ResolveAbsAndRelPaths(t.Destination)
	if err != nil {
		return err
	}

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, absDestPath)
	if err != nil {
		return errors.E(op, types.UniquePath(t.Destination), err)
	}
	r.Get.Destination = string(p.UniquePath)
	strategy, err := kptfilev1.ToUpdateStrategy(r.strategy)
	if err != nil {
		return err
	}
	r.Get.UpdateStrategy = strategy
	r.Get.IsDeploymentInstance = r.isDeploymentInstance
	return nil
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	const op errors.Op = "cmdget.runE"
	if err := r.Get.Run(r.ctx); err != nil {
		return errors.E(op, types.UniquePath(r.Get.Destination), err)
	}
	return r.RunPkgAutoPipeline()
}

func (r *Runner) ReadResourceListFromFnSource(resolvedPath string, fnConfig *yaml.RNode) ([]byte, error) {
	var inputs []kio.Reader
	inputs = append(inputs, kio.LocalPackageReader{
		PackagePath:        resolvedPath,
		MatchFilesGlob:     pkg.MatchAllKRM,
		PreserveSeqIndent:  true,
		PackageFileName:    kptfilev1.KptFileName,
		IncludeSubpackages: true,
		WrapBareSeqNode:    true,
	})
	var outputs []kio.Writer
	actual := &bytes.Buffer{}
	outputs = append(outputs, kio.ByteWriter{
		Writer:             actual,
		FunctionConfig:     fnConfig,
		WrappingKind:       kio.ResourceListKind,
		WrappingAPIVersion: kio.ResourceListAPIVersion,
	})
	err := kio.Pipeline{Inputs: inputs, Outputs: outputs}.Execute()
	if err != nil {
		return nil, err
	}
	return actual.Bytes(), err
}

func (r *Runner) GenerateNonKrmResources(includeNonKrmFiels []kptfilev1.LocalFile) ([]*fn.KubeObject, error) {
	var nonKrmObjects []*fn.KubeObject
	for _, localNonKrmFile := range includeNonKrmFiels {
		content, err := ioutil.ReadFile(filepath.Join(r.Get.Destination, localNonKrmFile.Path))
		if err != nil {
			return nil, err
		}
		newNonKrmFile := fn.NewNonKrmResource()
		obj, err := fn.NewFromTypedObject(newNonKrmFile)
		if err != nil {
			return nil, err
		}
		obj.SetName(localNonKrmFile.Name)
		obj.SetNestedString(string(content), "spec", "content")
		obj.SetNestedString(filepath.Base(localNonKrmFile.Path), "spec", "filename")
		nonKrmObjects = append(nonKrmObjects, obj)
	}
	return nonKrmObjects, nil
}

func (r *Runner) RunPkgAutoPipeline() error {
	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, r.Get.Destination)
	if err != nil {
		return err
	}
	kf, err := p.Kptfile()
	if err != nil {
		return err
	}
	var nonKrmObjects []*fn.KubeObject
	if kf.PkgAutoRun.InclNonKrmFiles != nil {
		nonKrmObjects, err = r.GenerateNonKrmResources(kf.PkgAutoRun.InclNonKrmFiles)
		if err != nil {
			return err
		}
	}
	err = kptfileutil.WriteFile(r.Get.Destination, kf)
	for _, kptFunction := range kf.PkgAutoRun.BuiltInFunctions {
		p, _ := pkg.New(filesys.FileSystemOrOnDisk{}, r.Get.Destination)
		configs, _ := kio.LocalPackageReader{PackagePath: filepath.Join(p.UniquePath.String(), kptFunction.ConfigPath), PreserveSeqIndent: true, WrapBareSeqNode: true}.Read()
		if len(configs) != 1 {
			return fmt.Errorf("expected exactly 1 functionConfig, found %d", len(configs))
		}
		functionConfig := configs[0]
		rawResourceList, err := r.ReadResourceListFromFnSource(p.UniquePath.String(), functionConfig)

		if err != nil {
			return err
		}
		rl, err := fn.ParseResourceList(rawResourceList)
		if err != nil {
			return err
		}
		rl.Items = append(rl.Items, nonKrmObjects...)
		if err = r.RunFunction(functionConfig, rl, kptFunction); err != nil {
			return err
		}
	}
	return nil
}

func getFunctionSpec(image, exec string) (*runtimeutil.FunctionSpec, error) {
	fn := &runtimeutil.FunctionSpec{}
	if image != "" {
		fn.Container.Image = image
	} else if exec != "" {
		s, err := shlex.Split(exec)
		if err != nil {
			return nil, fmt.Errorf("exec command %q must be valid: %w", exec, err)
		}
		if len(s) > 0 {
			fn.Exec.Path = s[0]
		}
	}
	return fn, nil
}

func (r *Runner) RunFunction(functionConfig *yaml.RNode, inputResourceList *fn.ResourceList, function kptfilev1.Function) error {
	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, r.Get.Destination)
	output := &bytes.Buffer{}
	content, _ := inputResourceList.ToYAML()
	input := bytes.NewReader(content)
	fnSpec, err := getFunctionSpec(function.Image, function.Exec)
	if err != nil {
		return err
	}
	run := runfn.RunFns{
		Ctx:                   r.ctx,
		Input:                 input,
		Output:                output,
		ImagePullPolicy:       fnruntime.IfNotPresentPull,
		AsCurrentUser:         false,
		ContinueOnEmptyResult: true,
		Function:              fnSpec,
		OriginalExec:          function.Exec,
		Path:                  p.UniquePath.String(),
		FnConfig:              functionConfig,
	}
	if err = runner.HandleError(r.ctx, run.Execute()); err != nil {
		return err
	}
	reformatedOut, err := r.CleanupBuiltInObjects(output.String())
	if err != nil {
		return err
	}
	writer := &kio.LocalPackageReadWriter{
		PackagePath:        string(p.UniquePath),
		MatchFilesGlob:     pkg.MatchAllKRM,
		PreserveSeqIndent:  true,
		PackageFileName:    kptfilev1.KptFileName,
		IncludeSubpackages: true,
	}
	return kio.Pipeline{
		Inputs: []kio.Reader{&kio.ByteReader{
			Reader:             bytes.NewBuffer(reformatedOut),
			PreserveSeqIndent:  true,
			WrapBareSeqNode:    true,
			WrappingKind:       kio.ResourceListKind,
			WrappingAPIVersion: kio.ResourceListAPIVersion,
		}},
		Outputs: []kio.Writer{writer},
	}.Execute()
}

func (r *Runner) CleanupBuiltInObjects(data string) ([]byte, error) {
	rl, err := fn.ParseResourceList([]byte(data))
	if err != nil {
		return nil, err
	}

	isNonKrmObject := func(o *fn.KubeObject) bool {
		return o.GetKind() == fn.NonKrmKind
	}
	generatedForInternalUse := func(o *fn.KubeObject) bool {
		if o.GetAnnotation(fn.GeneratorBuiltinIdentifier) == "" {
			return false
		}
		if r.ReserveBuiltin {
			return false
		}
		return true
	}
	items := rl.Items.WhereNot(isNonKrmObject).WhereNot(generatedForInternalUse)

	for _, object := range items {
		if object.GetAnnotation(fn.GeneratorIdentifier) != "" || object.GetAnnotation(fn.GeneratorBuiltinIdentifier) != "" {
			curFilePath := object.GetAnnotation(fn.PathAnnotation)
			newFilePath := filepath.Join(GeneratedDir, filepath.Base(curFilePath))
			object.SetAnnotation(fn.PathAnnotation, newFilePath)
		}
	}
	rl.Items = items
	return rl.ToYAML()
}
