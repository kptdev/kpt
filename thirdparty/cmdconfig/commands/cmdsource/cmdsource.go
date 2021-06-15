// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsource

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetSourceRunner returns a command for Source.
func GetSourceRunner(ctx context.Context, name string) *SourceRunner {
	r := &SourceRunner{
		WrapKind:       kio.ResourceListKind,
		WrapAPIVersion: kio.ResourceListAPIVersion,
		Ctx:            ctx,
	}
	c := &cobra.Command{
		Use:     "source [DIR] [flags]",
		Short:   fndocs.SourceShort,
		Long:    fndocs.SourceShort + "\n" + fndocs.SourceLong,
		Example: fndocs.SourceExamples,
		Args:    cobra.MaximumNArgs(1),
		RunE:    r.runE,
	}
	c.Flags().StringVarP(&r.Output, "output", "o", cmdutil.Stdout,
		fmt.Sprintf("output resources are written to stdout in provided format. Allowed values: %s|%s", cmdutil.Stdout, cmdutil.Unwrap))
	c.Flags().StringVar(&r.FunctionConfig, "fn-config", "",
		"path to function config file.")
	c.Flags().BoolVar(&r.IncludeMetaResources,
		"include-meta-resources", false, "include package meta resources in the command output")
	r.Command = c
	_ = c.MarkFlagFilename("fn-config", "yaml", "json", "yml")
	return r
}

func NewCommand(ctx context.Context, name string) *cobra.Command {
	return GetSourceRunner(ctx, name).Command
}

// SourceRunner contains the run function
type SourceRunner struct {
	Output               string
	WrapKind             string
	WrapAPIVersion       string
	FunctionConfig       string
	Command              *cobra.Command
	IncludeMetaResources bool
	Ctx                  context.Context
}

func (r *SourceRunner) runE(c *cobra.Command, args []string) error {
	if r.Output != cmdutil.Stdout && r.Output != cmdutil.Unwrap {
		return fmt.Errorf("invalid input for --output flag %q, must be %q or %q", r.Output, cmdutil.Stdout, cmdutil.Unwrap)
	}
	if len(args) == 0 {
		// default to current working directory
		args = append(args, ".")
	}
	// if there is a function-config specified, emit it
	var functionConfig *yaml.RNode
	if r.FunctionConfig != "" {
		configs, err := kio.LocalPackageReader{PackagePath: r.FunctionConfig}.Read()
		if err != nil {
			return err
		}
		if len(configs) != 1 {
			return fmt.Errorf("expected exactly 1 functionConfig, found %d", len(configs))
		}
		functionConfig = configs[0]
	}

	var inputs []kio.Reader
	matchFilesGlob := kio.MatchAll
	if r.IncludeMetaResources {
		matchFilesGlob = append(matchFilesGlob, kptfile.KptFileName)
	}
	for _, a := range args {
		pkgPath, err := filepath.Abs(a)
		if err != nil {
			return fmt.Errorf("cannot convert input path %q to absolute path: %w", a, err)
		}
		functionConfigFilter, err := pkg.FunctionConfigFilterFunc(types.UniquePath(pkgPath), r.IncludeMetaResources)
		if err != nil {
			return err
		}
		inputs = append(inputs, kio.LocalPackageReader{
			PackagePath:    a,
			MatchFilesGlob: matchFilesGlob,
			FileSkipFunc:   functionConfigFilter,
		})
	}

	var outputs []kio.Writer
	if r.Output == cmdutil.Stdout {
		outputs = append(outputs, kio.ByteWriter{
			Writer:                printer.FromContextOrDie(r.Ctx).OutStream(),
			KeepReaderAnnotations: true,
			WrappingKind:          r.WrapKind,
			WrappingAPIVersion:    r.WrapAPIVersion,
			FunctionConfig:        functionConfig,
		})
	} else {
		outputs = append(outputs, kio.ByteWriter{
			Writer:           printer.FromContextOrDie(r.Ctx).OutStream(),
			FunctionConfig:   functionConfig,
			ClearAnnotations: []string{kioutil.IndexAnnotation, kioutil.PathAnnotation},
		})
	}

	err := kio.Pipeline{
		Inputs:  inputs,
		Outputs: outputs,
		Filters: []kio.Filter{kio.FilterFunc(cmdutil.AreKrmFilter)},
	}.Execute()

	return runner.HandleError(r.Ctx, err)
}
